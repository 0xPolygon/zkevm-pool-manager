package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/hex"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// Endpoints contains implementations for the pool-manager JSON-RPC endpoints
type Endpoints struct {
	cfg    Config
	poolDB poolDBInterface
	sender senderInterface
}

// NewEndpoints creates an new instance of pool-manager JSON-RPC endpoints
func NewEndpoints(cfg Config, poolDB poolDBInterface, sender senderInterface) *Endpoints {
	e := &Endpoints{cfg: cfg, poolDB: poolDB, sender: sender}
	return e
}

func (e *Endpoints) SendRawTransaction(httpRequest *http.Request, input string) (interface{}, Error) {
	// Get the IP address of the request
	ip := ""
	ips := httpRequest.Header.Get("X-Forwarded-For")
	if ips != "" {
		ip = strings.Split(ips, ",")[0]
	}

	tx, err := hexToTx(input)
	if err != nil {
		log.Errorf("invalid tx input, error: %v", err)
		return nil, NewServerErrorWithData(InvalidParamsErrorCode, "invalid tx input", nil)
	}

	txJSON, err := tx.MarshalJSON()
	if err != nil {
		log.Errorf("error getting JSON marshal for tx %s, error: %v", tx.Hash(), err)
		return nil, NewServerErrorWithData(ParserErrorCode, "error parsing tx", nil)
	}
	decoded := string(txJSON)

	// Get from address
	fromAddress, err := GetSender(*tx)
	if err != nil {
		log.Warnf("error getting from address for tx %s, error: %v", tx.Hash(), err)
	}

	log.Debugf("adding tx %s to the pool", tx.Hash())

	l2Tx := &types.L2Transaction{
		Hash:        tx.Hash().String(),
		ReceivedAt:  time.Now(),
		FromAddress: fromAddress.String(),
		GasPrice:    tx.GasPrice().Uint64(),
		Nonce:       tx.Nonce(),
		Status:      types.TxStatusPending,
		IP:          ip,
		Encoded:     input,
		Decoded:     decoded,
	}

	l2Tx.Id, err = e.poolDB.AddL2Transaction(context.Background(), l2Tx)
	if err != nil {
		log.Errorf("error adding tx %s to pool db, error: %v", l2Tx.Tag(), err)
	}

	err = e.sender.SendL2Transaction(l2Tx)

	if err != nil {
		log.Infof("sending tx %s to sequencer returns error: %v", l2Tx.Tag(), err)
		return nil, NewServerErrorWithData(DefaultErrorCode, err.Error(), nil)
	}

	log.Infof("tx %s sent to sequencer and added to the pool database", l2Tx.Tag())

	return tx.Hash().String(), nil
}

// GetSender gets the sender from the transaction's signature
func GetSender(tx ethTypes.Transaction) (common.Address, error) {
	signer := ethTypes.NewEIP155Signer(tx.ChainId())
	sender, err := signer.Sender(&tx)
	if err != nil {
		return common.Address{}, err
	}
	return sender, nil
}

func hexToTx(str string) (*ethTypes.Transaction, error) {
	tx := new(ethTypes.Transaction)

	b, err := hex.DecodeHex(str)
	if err != nil {
		return nil, err
	}

	if err := tx.UnmarshalBinary(b); err != nil {
		return nil, err
	}

	return tx, nil
}
