package monitor

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
)

type poolDBInterface interface {
	UpdateL2TransactionStatus(ctx context.Context, txHash string, newStatus string, errorMsg string) error
	GetL2TransactionsToMonitor(ctx context.Context) ([]*db.L2Transaction, error)
}
