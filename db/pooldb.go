package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
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

// L2Transaction represents a pool tx
type L2Transaction struct {
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
func (p *PoolDB) AddL2Transaction(ctx context.Context, tx *L2Transaction) error {
	const sql = `
		INSERT INTO pool.transaction 
		(hash, received_at,	updated_at, from_address, gas_price, nonce,	status,	ip, encoded, decoded) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (hash) DO UPDATE SET received_at = $2, updated_at = $3, from_address = $4, gas_price = $5, nonce = $6, status = $7, ip = $8, encoded = $9, decoded = $10
	`

	_, err := p.db.Exec(ctx, sql, tx.Hash, tx.ReceivedAt, time.Now(), tx.FromAddress, tx.GasPrice, tx.Nonce, tx.Status, tx.IP, tx.Encoded, tx.Decoded)
	if err != nil {
		return err
	}

	return nil
}

func (p *PoolDB) GetL2TransactionsByStatus(ctx context.Context, status string) ([]*L2Transaction, error) {
	const resendTxsSQL = "SELECT hash, received_at, from_address, gas_price, nonce, status, ip, encoded, decoded FROM pool.transaction WHERE status = $1"

	rows, err := p.db.Query(ctx, resendTxsSQL, status)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	txs := make([]*L2Transaction, len(rows.RawValues()))
	for rows.Next() {
		tx := &L2Transaction{}

		err := rows.Scan(&tx.Hash, &tx.ReceivedAt, &tx.FromAddress, &tx.GasPrice, &tx.Nonce, &tx.Status, &tx.IP, &tx.Encoded, &tx.Decoded)
		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (p *PoolDB) GetL2TransactionsToResend(ctx context.Context) ([]*L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, TxStatusResend)
}

func (p *PoolDB) GetL2TransactionsToMonitor(ctx context.Context) ([]*L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, TxStatusSent)
}

func (p *PoolDB) GetL2TransactionsToSend(ctx context.Context) ([]*L2Transaction, error) {
	return p.GetL2TransactionsByStatus(ctx, TxStatusPending)
}

func (p *PoolDB) UpdateL2TransactionStatus(ctx context.Context, txHash string, newStatus string, errorMsg string) error {
	const updateStatusSQL = "UPDATE pool.transaction SET updated_at = $2, status = $3, error = $4 WHERE hash = $1"

	_, err := p.db.Exec(ctx, updateStatusSQL, txHash, time.Now(), newStatus, errorMsg)
	if err != nil {
		return err
	}

	return nil
}
