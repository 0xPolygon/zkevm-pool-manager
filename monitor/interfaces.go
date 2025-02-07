package monitor

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
)

type poolDBInterface interface {
	UpdateL2TransactionStatus(ctx context.Context, id uint64, newStatus string, errorMsg string) error
	GetL2TransactionsToMonitor(ctx context.Context, page int) ([]*types.L2Transaction, error)
}
