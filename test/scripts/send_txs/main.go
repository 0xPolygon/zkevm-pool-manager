package main

import (
	"context"
	"math/big"
	"strings"
	"time"

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
	l2NodeURL := "http://localhost:8467"
	privateKey := "0x26e86e45f6fc45ec6e2ecd128cec80fa1d1505e5507dcd2ae58c3130a7a97b48"
	chainID := uint64(999999)

	log.Infof("connecting to pool-manager %s", poolManagerURL)
	poolmgrClient, err := ethclient.Dial(poolManagerURL)
	chkErr(err)
	log.Infof("connected")

	log.Infof("connecting to L2 node %s", poolManagerURL)
	l2NodeClient, err := ethclient.Dial(l2NodeURL)
	chkErr(err)
	log.Infof("connected")

	auth := GetAuth(privateKey, chainID)
	chkErr(err)

	const receiverAddr = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"

	transferAmount := big.NewInt(1)

	nonce, err := l2NodeClient.NonceAt(ctx, auth.From, nil)
	chkErr(err)

	txs := []common.Hash{}
	for i := 0; i < 1; i++ {
		nonce := nonce + uint64(i)
		to := common.HexToAddress(receiverAddr)
		tx := ethTransfer(ctx, poolmgrClient, auth, to, transferAmount, nonce)
		rlp, err := tx.MarshalBinary()
		chkErr(err)

		log.Infof("tx sent: %s, nonce: %d, rlp: %s", tx.Hash(), nonce, hex.EncodeToHex(rlp))

		txs = append(txs, tx.Hash())
	}

	for _, txHash := range txs {
		waitForReceipt(ctx, l2NodeClient, txHash)
	}
}

func waitForReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) {
	for {
		// log.Infof("getting receipt for tx %s", txHash)
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			log.Infof("receipt for tx %s received, status: %d", txHash, receipt.Status)
			return
		}
		// log.Infof("receipt for tx %s still not available, error: %v", txHash, err)
		time.Sleep(500 * time.Microsecond)
	}
}

func ethTransfer(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, to common.Address, amount *big.Int, nonce uint64) *types.Transaction {

	gasPrice := new(big.Int).SetUint64(1000000000)
	gasLimit := uint64(2000000)

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

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
