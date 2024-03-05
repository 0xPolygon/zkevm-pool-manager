package server

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
)

type poolDBInterface interface {
	AddTx(ctx context.Context, tx db.Transaction) error
}
