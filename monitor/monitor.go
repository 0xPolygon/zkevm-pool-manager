package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
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
	l2Tx      types.L2Transaction
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

func (m *Monitor) AddL2Transaction(l2Tx *types.L2Transaction) {
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
	log.Debugf("monitor request for tx %s added to the queue channel", request.l2Tx.Tag(), request.l2Tx.Tag())
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
		m.workerProcessRequest(monitorRequest, rpcClient, workerNum)
	}
}

func (m *Monitor) scheduleRequestRetry(request *monitorRequest) {
	request.nextRetry = time.Now().Add(m.cfg.RetryWaitInterval.Duration)
	m.addRequestToRetryList(request)
}

func (m *Monitor) addRequestToRetryList(request *monitorRequest) {
	m.requestRetryList.add(request)

	if m.requestRetryList.len() == 1 {
		log.Debugf("signal monitor request retries")
		m.requestRetryCond.Signal()
	}
}

func (m *Monitor) workerProcessRequest(request *monitorRequest, rpcClient *ethclient.Client, workerNum int) {
	log.Infof("monitor-worker[%03d]: monitoring tx %s", workerNum, request.l2Tx.Tag())

	receipt, err := rpcClient.TransactionReceipt(context.Background(), common.HexToHash(request.l2Tx.Hash))
	if err != nil {
		if !errors.Is(err, ethereum.NotFound) {
			log.Errorf("monitor-worker[%03d]: error getting receipt for tx %s, schedule retry, error: %v", workerNum, request.l2Tx.Tag(), err)
		} else {
			log.Debugf("monitor-worker[%03d]: receipt for tx %s still not available, schedule retry", workerNum, request.l2Tx.Tag())
		}
		m.scheduleRequestRetry(request)
	} else {
		l2TxStatus := types.TxStatusConfirmed
		if receipt.Status == 0 {
			l2TxStatus = types.TxStatusFailed
		}

		err := m.poolDB.UpdateL2TransactionStatus(context.Background(), request.l2Tx.Id, l2TxStatus, "")
		if err != nil {
			log.Errorf("monitor-worker[%03d]: error updating status for tx %s, schedule retry, error: %v", workerNum, request.l2Tx.Tag(), err)
			m.scheduleRequestRetry(request)
		} else {
			log.Infof("monitor-worker[%03d]: receipt for tx %s received, status: %d", workerNum, request.l2Tx.Tag(), receipt.Status)
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
				log.Debugf("monitor tx %s has expired, updating status", request.l2Tx.Tag())
				err := m.poolDB.UpdateL2TransactionStatus(context.Background(), request.l2Tx.Id, types.TxStatusExpired, "")
				if err != nil {
					log.Errorf("error updating tx %s status (%s) in the pool db, error: %v", request.l2Tx.Tag(), types.TxStatusExpired, err)
				}
				m.requestRetryList.delete(request)
			} else if request.nextRetry.Before(now) {
				log.Debugf("retry monitor tx %s that was schedule to %v", request.l2Tx.Tag(), request.nextRetry)
				m.requestRetryList.delete(request)
				m.enqueueMonitorRequest(request)
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
	for page := 0; ; page++ {
		l2Txs, err := m.poolDB.GetL2TransactionsToMonitor(context.Background(), page)
		if err != nil {
			log.Errorf("error when getting txs to monitor from the pool database, error: %v", err)
			return
		}
		log.Infof("load %d txs from the pool database", len(l2Txs))
		if len(l2Txs) == 0 {
			log.Infof("no txs to monitor from the pool database")
			return
		}

		for _, l2Tx := range l2Txs {
			m.AddL2Transaction(l2Tx)
		}
	}

}

func (m *Monitor) Summary() {
	log.Infof("Summary monitor: request retry:%v", m.requestRetryList.len())
}
