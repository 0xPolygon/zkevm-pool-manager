package sender

import (
	"context"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
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
	l2Tx db.L2Transaction
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

func (s *Sender) SendL2Transaction(l2Tx *db.L2Transaction) error {
	request := &sendRequest{
		l2Tx: *l2Tx,
		wg:   new(sync.WaitGroup),
	}

	request.wg.Add(1)
	s.enqueueSenderRequest(request)
	request.wg.Wait()

	if request.err != nil {
		log.Infof("error sending tx %s to sequencer, error: %v", l2Tx.Hash, request.err)
		s.poolDB.UpdateL2TransactionStatus(context.Background(), l2Tx.Hash, db.TxStatusInvalid, request.err.Error())
	} else {
		log.Infof("tx %s sent to sequencer", l2Tx.Hash)
		s.poolDB.UpdateL2TransactionStatus(context.Background(), l2Tx.Hash, db.TxStatusSent, "")
		s.monitor.AddL2Transaction(l2Tx)
	}

	return request.err
}

func (s *Sender) enqueueSenderRequest(request *sendRequest) {
	log.Debugf("send request for tx %s added to the queue channel", request.l2Tx.Hash)
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
		err := s.processSendRequest(sendRequest, seqClient, workerNum)

		sendRequest.err = err
		sendRequest.wg.Done()
	}
}

func (s *Sender) processSendRequest(request *sendRequest, seqClient *ethclient.Client, workerNum int) error {
	log.Debugf("sender-worker[%03d]: sending tx %s", workerNum, request.l2Tx.Hash)

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

		for _, tx := range txs {
			err := s.SendL2Transaction(tx)
			if err != nil {
				log.Errorf("error sending tx %s to sequencer, error: %v", tx.Hash, err)
				s.poolDB.UpdateL2TransactionStatus(context.Background(), tx.Hash, db.TxStatusInvalid, err.Error())
			} else {
				log.Debugf("tx %s sent to sequencer", tx.Hash)
				s.poolDB.UpdateL2TransactionStatus(context.Background(), tx.Hash, db.TxStatusSent, "")
			}
		}

		if len(txs) == 0 {
			time.Sleep(s.cfg.ResendTxsCheckInterval.Duration)
		}
	}
}

func (s *Sender) sendL2TransactionsFromPoolDB() {
	l2Txs, err := s.poolDB.GetL2TransactionsToSend(context.Background())
	if err != nil {
		log.Errorf("error when getting txs to send from the pool database, error: %v", err)
	}

	for _, l2Tx := range l2Txs {
		s.SendL2Transaction(l2Tx)
	}
}
