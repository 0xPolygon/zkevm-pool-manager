package server

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
)

type poolDBInterface interface {
	AddL2Transaction(ctx context.Context, tx *types.L2Transaction) (uint64, error)
	UpdateL2TransactionStatus(ctx context.Context, id uint64, newStatus string, errorMsg string) error
}

type senderInterface interface {
	SendL2Transaction(l2Tx *types.L2Transaction) error
}
