package sender

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
)

type poolDBInterface interface {
	UpdateL2TransactionStatus(ctx context.Context, id uint64, newStatus string, errorMsg string) error
	GetL2TransactionsToResend(ctx context.Context) ([]*types.L2Transaction, error)
	GetL2TransactionsToSend(ctx context.Context) ([]*types.L2Transaction, error)
}

type monitorInterface interface {
	AddL2Transaction(l2Tx *types.L2Transaction)
}
