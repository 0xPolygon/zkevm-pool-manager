package sender

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
)

type poolDBInterface interface {
	UpdateL2TransactionStatus(ctx context.Context, txHash string, newStatus string, errorMsg string) error
	GetL2TransactionsToResend(ctx context.Context) ([]*db.L2Transaction, error)
	GetL2TransactionsToSend(ctx context.Context) ([]*db.L2Transaction, error)
}

type monitorInterface interface {
	AddL2Transaction(l2Tx *db.L2Transaction)
}
