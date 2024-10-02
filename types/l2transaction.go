package types

import (
	"fmt"
	"time"
)

const (
	// TxStatusPending represents a tx that has not been processed
	TxStatusPending string = "pending"
	// TxStatusInvalid represents an invalid tx
	TxStatusInvalid string = "invalid"
	// TxStatusConfirmed represents a tx that has been selected
	TxStatusConfirmed string = "confirmed"
	// TxStatusSent represents a tx that has been sent to the sequencer
	TxStatusSent string = "sent"
	// TxStatusFailed represents a tx that has been failed after processing
	TxStatusFailed string = "failed"
	// TxStatusResend represents a tx that has been requested to be resent to the sequencer
	TxStatusResend string = "resend"
	// TxStatusResend represents a tx that has reached TxLifeTimeMax time wating to receive the receipt
	TxStatusExpired string = "expired"
)

// L2Transaction represents a L2 transaction
type L2Transaction struct {
	Id          uint64
	Hash        string
	ReceivedAt  time.Time
	FromAddress string
	GasPrice    uint64
	Nonce       uint64
	Status      string
	IP          string
	Encoded     string
	Decoded     string
}

func (t *L2Transaction) Tag() string {
	return fmt.Sprintf("[%d]:%s", t.Id, t.Hash)
}
