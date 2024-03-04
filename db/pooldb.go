package db

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	// TxStatusPending represents a tx that has not been processed
	TxStatusPending string = "pending"
	// TxStatusInvalid represents an invalid tx
	TxStatusInvalid string = "invalid"
	// TxStatusSelected represents a tx that has been selected
	TxStatusSelected string = "selected"
	// TxStatusFailed represents a tx that has been failed after processing
	TxStatusFailed string = "failed"
)

// Transaction represents a pool tx
type Transaction struct {
	types.Transaction
	Status     string
	ReceivedAt time.Time
}

// NewTransaction creates a new transaction
func NewTransaction(tx types.Transaction) *Transaction {
	poolTx := Transaction{
		Transaction: tx,
		Status:      TxStatusPending,
		ReceivedAt:  time.Now(),
	}

	return &poolTx
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

// GetSender gets the sender from the transaction's signature
func GetSender(tx types.Transaction) (common.Address, error) {
	signer := types.NewEIP155Signer(tx.ChainId())
	sender, err := signer.Sender(&tx)
	if err != nil {
		return common.Address{}, err
	}
	return sender, nil
}

// AddTx adds a transaction to the pool
func (p *PoolDB) AddTx(ctx context.Context, tx Transaction) error {
	hash := tx.Hash().Hex()

	b, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	encoded := hex.EncodeToHex(b)

	b, err = tx.MarshalJSON()
	if err != nil {
		return err
	}
	decoded := string(b)

	gasPrice := tx.GasPrice().Uint64()
	nonce := tx.Nonce()

	sql := `
		INSERT INTO pool.transaction 
		(hash, received_at,	from_address, gas_price, nonce,	status,	encoded, decoded) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (hash) DO UPDATE SET received_at = $2, from_address = $3, gas_price = $4, nonce = $5, status = $6, encoded = $7, decoded = $8
	`

	// Get FromAddress from the JSON data
	data, err := GetSender(tx.Transaction)
	if err != nil {
		return err
	}
	fromAddress := data.String()

	_, err = p.db.Exec(ctx, sql, hash, tx.ReceivedAt, fromAddress, gasPrice, nonce, tx.Status, encoded, decoded)
	if err != nil {
		return err
	}

	return nil
}
