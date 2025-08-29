package db

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PoolDB represent a postgres pool database to store transactions
type PoolDB struct {
	db *pgxpool.Pool
}

// NewPostgresPoolStorage creates and initializes an instance of PostgresPoolStorage
func NewPoolDB(cfg Config) (*PoolDB, error) {
	poolDB, err := NewSQLDB(cfg)
	if err != nil {
		return nil, err
	}

	return &PoolDB{db: poolDB}, nil
}

// AddTx adds a L2 transaction to the pool
func (p *PoolDB) AddL2Transaction(ctx context.Context, tx *types.L2Transaction) (uint64, error) {
	const sql = `
		INSERT INTO pool.transaction 
		(hash, received_at,	updated_at, from_address, gas_price, nonce,	status,	ip, encoded, decoded) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id uint64

	err := p.db.QueryRow(ctx, sql, tx.Hash, tx.ReceivedAt, time.Now(), tx.FromAddress, tx.GasPrice, tx.Nonce, tx.Status, tx.IP, tx.Encoded, tx.Decoded).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PoolDB) GetL2TransactionsByStatus(ctx context.Context, status string) ([]*types.L2Transaction, error) {
	const resendTxsSQL = "SELECT id, hash, received_at, from_address, gas_price, nonce, status, ip, encoded, decoded FROM pool.transaction WHERE status = $1"

	rows, err := p.db.Query(ctx, resendTxsSQL, status)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	txs := make([]*types.L2Transaction, len(rows.RawValues()))
	for rows.Next() {
		tx := &types.L2Transaction{}

		err := rows.Scan(&tx.Id, &tx.Hash, &tx.ReceivedAt, &tx.FromAddress, &tx.GasPrice, &tx.Nonce, &tx.Status, &tx.IP, &tx.Encoded, &tx.Decoded)
		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (p *PoolDB) GetL2TransactionsToResend(ctx context.Context) ([]*types.L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, types.TxStatusResend)
}

func (p *PoolDB) GetL2TransactionsToMonitor(ctx context.Context) ([]*types.L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, types.TxStatusSent)
}

func (p *PoolDB) GetL2TransactionsToSend(ctx context.Context) ([]*types.L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, types.TxStatusPending)
}

func (p *PoolDB) UpdateL2TransactionStatus(ctx context.Context, id uint64, newStatus string, errorMsg string) error {
	const updateStatusSQL = "UPDATE pool.transaction SET updated_at = $2, status = $3, error = $4 WHERE id = $1"

	_, err := p.db.Exec(ctx, updateStatusSQL, id, time.Now(), newStatus, errorMsg)
	if err != nil {
		return err
	}

	return nil
}
