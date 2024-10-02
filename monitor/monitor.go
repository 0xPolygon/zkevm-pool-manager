package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Monitor struct {
	cfg              Config
	poolDB           poolDBInterface
	requestChan      chan *monitorRequest
	requestRetryList *monitorRequestList
	requestRetryCond *sync.Cond
}

type monitorRequest struct {
	l2Tx      db.L2Transaction
	nextRetry time.Time
}

func NewMonitor(cfg Config, poolDB poolDBInterface) *Monitor {
	return &Monitor{
		cfg:              cfg,
		poolDB:           poolDB,
		requestChan:      make(chan *monitorRequest, cfg.QueueSize),
		requestRetryList: newMonitorRequestList(),
		requestRetryCond: sync.NewCond(&sync.Mutex{}),
	}
}

func (m *Monitor) Start() {
	log.Infof("starting %d monitor workers", m.cfg.Workers)
	for i := 0; i < int(m.cfg.Workers); i++ {
		go m.startMonitorWorker(i)
	}

	go m.checkMonitorRequestRetries()

	log.Infof("monitoring txs from the pool database")
	m.monitorL2TransactionsFromPoolDB()
}

func (m *Monitor) AddL2Transaction(l2Tx *db.L2Transaction) {
	request := &monitorRequest{
		l2Tx: *l2Tx,
	}

	if m.cfg.InitialWaitInterval.Duration > 0 {
		request.nextRetry = time.Now().Add(m.cfg.InitialWaitInterval.Duration)
		m.addRequestToRetryList(request)
	} else {
		m.enqueueMonitorRequest(request)
	}
}

func (m *Monitor) enqueueMonitorRequest(request *monitorRequest) {
	log.Debugf("monitor request for tx %s added to the queue channel", request.l2Tx.Hash)
	// Enqueue monitorRequest in the channel. We do in a go func to avoid blocking in case the channel buffer is full
	go func() { m.requestChan <- request }()
}

func (m *Monitor) startMonitorWorker(workerNum int) {
	rpcClient, err := ethclient.Dial(m.cfg.L2NodeURL)

	if err != nil {
		log.Errorf("monitor-worker[%03d]: error creating rpc client for %s, err: %v", workerNum, m.cfg.L2NodeURL, err)
		return
	}

	log.Debugf("monitor-worker[%03d]: started", workerNum)
	for monitorRequest := range m.requestChan {
		m.processMonitorRequest(monitorRequest, rpcClient, workerNum)
	}
}

func (m *Monitor) scheduleRequestRetry(request *monitorRequest, workerNum int) {
	request.nextRetry = time.Now().Add(m.cfg.RetryWaitInterval.Duration)
	log.Debugf("monitor-worker[%03d]: scheduled retry monitor tx %s at %v", workerNum, request.l2Tx.Hash, request.nextRetry)

	m.addRequestToRetryList(request)
}

func (m *Monitor) addRequestToRetryList(request *monitorRequest) {
	m.requestRetryList.add(request)

	if m.requestRetryList.len() == 1 {
		log.Debugf("monitor-worker[%03d]: signal monitor request retries")
		m.requestRetryCond.Signal()
	}
}

func (m *Monitor) processMonitorRequest(request *monitorRequest, rpcClient *ethclient.Client, workerNum int) {
	log.Infof("monitor-worker[%03d]: monitoring tx %s", workerNum, request.l2Tx.Hash)

	receipt, err := rpcClient.TransactionReceipt(context.Background(), common.HexToHash(request.l2Tx.Hash))
	if err != nil {
		if !errors.Is(err, ethereum.NotFound) {
			log.Errorf("monitor-worker[%03d]: error getting receipt for tx %s, error: %v", workerNum, request.l2Tx.Hash, err)
		} else {
			log.Debugf("monitor-worker[%03d]: receipt for tx %s still not available, schedule retry", workerNum, request.l2Tx.Hash)
		}
		m.scheduleRequestRetry(request, workerNum)
	} else {
		l2TxStatus := db.TxStatusConfirmed
		if receipt.Status == 0 {
			l2TxStatus = db.TxStatusFailed
		}

		err := m.poolDB.UpdateL2TransactionStatus(context.Background(), request.l2Tx.Hash, l2TxStatus, "")
		if err != nil {
			log.Errorf("monitor-worker[%03d]: error updating status for tx %s, error: %v", workerNum, request.l2Tx.Hash, err)
			m.scheduleRequestRetry(request, workerNum)
		} else {
			log.Infof("monitor-worker[%03d]: receipt for tx %s received, status: %d", workerNum, request.l2Tx.Hash, receipt.Status)
			m.requestRetryList.delete(request)
		}
	}
}

func (m *Monitor) checkMonitorRequestRetries() {
	for {
		if m.requestRetryList.len() > 0 {
			now := time.Now()
			request := m.requestRetryList.getByIndex(0)
			// Check if tx has reached max lifetime
			if request.l2Tx.ReceivedAt.Add(m.cfg.TxLifeTimeMax.Duration).Before(now) {
				log.Debugf("monitor tx %s has expired, updating status", request.l2Tx.Hash)
				m.poolDB.UpdateL2TransactionStatus(context.Background(), request.l2Tx.Hash, db.TxStatusExpired, "")
				if request.nextRetry.Before(now) && request.l2Tx.Status != "confirmed" && request.l2Tx.Status != "sent" {
					log.Debugf("retry monitor tx %s that was schedule to %v", request.l2Tx.Hash, request.nextRetry)
					if m.requestRetryList.delete(request) {
						log.Warnf("delete monitor tx %s successful", request.l2Tx.Hash)
					}
					// Only enqueue requests that are not expired to avoid infinite loop
					if request.l2Tx.Status != "expired" {
						m.enqueueMonitorRequest(request)
					}
				}
			} else {
				sleepTime := request.nextRetry.Sub(now)
				time.Sleep(sleepTime)
			}
		} else {
			// wait for new monitorRequest to retry
			m.requestRetryCond.L.Lock()
			m.requestRetryCond.Wait()
			m.requestRetryCond.L.Unlock()
			log.Debugf("continuing processing monitor txs requests retries")
		}
	}
}

func (m *Monitor) monitorL2TransactionsFromPoolDB() {
	l2Txs, err := m.poolDB.GetL2TransactionsToMonitor(context.Background())
	if err != nil {
		log.Errorf("error when getting txs to monitor from the pool database, error: %v", err)
	}

	for _, l2Tx := range l2Txs {
		m.AddL2Transaction(l2Tx)
	}
}
