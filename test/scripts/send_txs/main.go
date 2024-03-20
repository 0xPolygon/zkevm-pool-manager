package main

import (
	"context"
	"math/big"
	"strings"

	"github.com/0xPolygonHermez/zkevm-pool-manager/hex"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	ctx := context.Background()

	poolManagerURL := "http://localhost:8545"
	privateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	chainID := uint64(1234)

	log.Infof("connecting to %s", poolManagerURL)
	client, err := ethclient.Dial(poolManagerURL)
	chkErr(err)
	log.Infof("connected")

	auth := GetAuth(privateKey, chainID)
	chkErr(err)

	const receiverAddr = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"

	transferAmount := big.NewInt(1)
	nonce := uint64(0)

	for i := 0; i < 1; i++ {
		nonce := nonce + uint64(i)
		to := common.HexToAddress(receiverAddr)
		tx := ethTransfer(ctx, client, auth, to, transferAmount, &nonce)
		rlp, err := tx.MarshalBinary()
		chkErr(err)

		log.Infof("tx sent: %s, rlp: %s", tx.Hash().Hex(), hex.EncodeToHex(rlp))
	}
}

func ethTransfer(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, to common.Address, amount *big.Int, nonce *uint64) *types.Transaction {

	gasPrice := new(big.Int).SetUint64(10)
	gasLimit := uint64(100)

	tx := types.NewTransaction(*nonce, to, amount, gasLimit, gasPrice, nil)

	signedTx, err := auth.Signer(auth.From, tx)
	chkErr(err)

	err = client.SendTransaction(ctx, signedTx)

	chkErr(err)

	return signedTx
}

// GetAuth configures and returns an auth object.
func GetAuth(privateKeyStr string, chainID uint64) *bind.TransactOpts {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyStr, "0x"))
	chkErr(err)

	var txOpts *bind.TransactOpts
	txOpts, err = bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(0).SetUint64(chainID))
	chkErr(err)

	return txOpts
}

func chkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
