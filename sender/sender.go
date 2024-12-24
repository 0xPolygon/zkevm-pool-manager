package sender

import (
	"context"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v4"
)

type Sender struct {
	cfg         Config
	poolDB      poolDBInterface
	monitor     monitorInterface
	requestChan chan *sendRequest
}

type sendRequest struct {
	l2Tx types.L2Transaction
	wg   *sync.WaitGroup
	err  error
}

func NewSender(cfg Config, poolDB poolDBInterface, monitor monitorInterface) *Sender {
	return &Sender{
		cfg:         cfg,
		poolDB:      poolDB,
		monitor:     monitor,
		requestChan: make(chan *sendRequest, cfg.QueueSize)}
}

func (s *Sender) Start() {
	log.Infof("starting %d sender workers", s.cfg.Workers)

	for i := 0; i < int(s.cfg.Workers); i++ {
		go s.startSenderWorker(i)
	}

	go s.checkL2TransactionsToResend()

	log.Infof("sending txs from the pool database")
	s.sendL2TransactionsFromPoolDB()
}

func (s *Sender) SendL2Transaction(l2Tx *types.L2Transaction) error {
	request := &sendRequest{
		l2Tx: *l2Tx,
		wg:   new(sync.WaitGroup),
	}

	request.wg.Add(1)
	s.enqueueSenderRequest(request)
	request.wg.Wait()

	if request.err != nil {
		err := s.poolDB.UpdateL2TransactionStatus(context.Background(), l2Tx.Id, types.TxStatusInvalid, request.err.Error())
		if err != nil {
			log.Errorf("error updating tx %s status (%s) in the pool db, error: %v", l2Tx.Tag(), types.TxStatusInvalid, err)
		}
	} else {
		err := s.poolDB.UpdateL2TransactionStatus(context.Background(), l2Tx.Id, types.TxStatusSent, "")
		if err != nil {
			log.Errorf("error updating tx %s status (%s) in the pool db, error: %v", l2Tx.Tag(), types.TxStatusSent, err)
		}
		s.monitor.AddL2Transaction(l2Tx)
	}

	return request.err
}

func (s *Sender) enqueueSenderRequest(request *sendRequest) {
	log.Debugf("send request for tx %s added to the queue channel", request.l2Tx.Tag())
	// Enqueue monitorRequest in the channel. We do in a go func to avoid blocking in case the channel buffer is full
	go func() { s.requestChan <- request }()
}

func (s *Sender) startSenderWorker(workerNum int) {
	seqClient, err := ethclient.Dial(s.cfg.SequencerURL)

	if err != nil {
		log.Errorf("sender-worker[%03d]: error creating sequencer client for %s, err: %v", workerNum, s.cfg.SequencerURL, err)
		return
	}

	log.Debugf("sender-worker[%03d]: started", workerNum)
	for sendRequest := range s.requestChan {
		err := s.workerProcessRequest(sendRequest, seqClient, workerNum)

		sendRequest.err = err
		sendRequest.wg.Done()
	}
}

func (s *Sender) workerProcessRequest(request *sendRequest, seqClient *ethclient.Client, workerNum int) error {
	log.Debugf("sender-worker[%03d]: sending tx %s", workerNum, request.l2Tx.Tag())

	return seqClient.Client().CallContext(context.Background(), nil, "eth_sendRawTransaction", request.l2Tx.Encoded)
}

func (s *Sender) checkL2TransactionsToResend() {
	for {
		txs, err := s.poolDB.GetL2TransactionsToResend(context.Background())
		if err != nil && err != pgx.ErrNoRows {
			log.Errorf("error loading txs to resend from pool, error: %v", err)
			time.Sleep(s.cfg.ResendTxsCheckInterval.Duration)
			continue
		}

		if len(txs) > 0 {
			log.Infof("resending txs to the sequencer")
		}

		for _, l2Tx := range txs {
			err := s.SendL2Transaction(l2Tx)
			if err != nil {
				log.Infof("resending tx %s to sequencer returns error: %v", l2Tx.Tag(), err)
			} else {
				log.Infof("tx %s resent to sequencer", l2Tx.Tag())
			}
		}

		if len(txs) == 0 {
			time.Sleep(s.cfg.ResendTxsCheckInterval.Duration)
		}
	}
}

func (s *Sender) sendL2TransactionsFromPoolDB() {
	for page := 0; ; page++ {
		l2Txs, err := s.poolDB.GetL2TransactionsToSend(context.Background(), page)
		if err != nil {
			log.Errorf("error when getting txs to send from the pool database, error: %v", err)
			return
		}
		log.Infof("sending txs to the sequencer, len: %d", len(l2Txs))
		if len(l2Txs) == 0 {
			return
		}
		for _, l2Tx := range l2Txs {
			err := s.SendL2Transaction(l2Tx)
			if err != nil {
				log.Infof("sending tx %s to sequencer returns error: %v", l2Tx.Tag(), err)
			} else {
				log.Infof("tx %s sent to sequencer", l2Tx.Tag())
			}
		}
	}
}
