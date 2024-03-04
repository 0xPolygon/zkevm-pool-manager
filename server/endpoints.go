package server

import (
	"context"
	"net/http"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/hex"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// Endpoints contains implementations for the pool-manager endpoints
type Endpoints struct {
	cfg    Config
	poolDB *db.PoolDB
}

// NewEndpoints creates an new instance of endpoints
func NewEndpoints(cfg Config, poolDB *db.PoolDB) *Endpoints {
	e := &Endpoints{cfg: cfg, poolDB: poolDB}
	return e
}

func (e *Endpoints) SendRawTransaction(httpRequest *http.Request, input string) (interface{}, Error) {
	tx, err := hexToTx(input)
	if err != nil {
		return RPCErrorResponse(InvalidParamsErrorCode, "invalid tx input", err, false)
	}

	log.Debugf("adding tx %s to the pool", tx.Hash().Hex())

	dbTx := db.NewTransaction(*tx)
	err = e.poolDB.AddTx(context.Background(), *dbTx)
	if err != nil {
		log.Errorf("error adding tx to pool db, error: %v", err)
		return RPCErrorResponse(DefaultErrorCode, err.Error(), nil, false)
	}

	log.Infof("added tx %s to the pool", tx.Hash().Hex())

	return tx.Hash().Hex(), nil
}

func hexToTx(str string) (*ethTypes.Transaction, error) {
	tx := new(ethTypes.Transaction)

	b, err := hex.DecodeHex(str)
	if err != nil {
		return nil, err
	}

	if err := tx.UnmarshalBinary(b); err != nil {
		return nil, err
	}

	return tx, nil
}
