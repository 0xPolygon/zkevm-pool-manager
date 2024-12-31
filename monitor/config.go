package monitor

import "github.com/0xPolygonHermez/zkevm-pool-manager/config/types"

// Config for pool-manager monitor
type Config struct {
	// L2NodeURL defines the URL L2 node (RPC) used by the monitor to query for txs receipts
	L2NodeURL string `mapstructure:"L2NodeURL"`

	// Workers is the number of monitor workers to query for txs receipts
	Workers uint16 `mapstructure:"Workers"`

	// QueueSize is the size of the queue for L2 txs that need to be monitored to get the tx receipt
	QueueSize uint16 `mapstructure:"QueueSize"`

	// InitialWaitInterval is the time the monitor worker will wait before to try get the tx receipt for first time
	InitialWaitInterval types.Duration `mapstructure:"InitialWaitInterval"`

	// RetryWaitInterval is the time the monitor worker will wait before to retry to get the tx receipt if it still doesn't exists
	RetryWaitInterval types.Duration `mapstructure:"RetryWaitInterval"`

	// TxLifetimeMax is the time a tx can be monitored waiting for the receipt
	TxLifeTimeMax types.Duration `mapstructure:"TxLifeTimeMax"`

	// RPCReadTimeout is the timeout for the RPC client to read the response from the L2 node
	RPCReadTimeout types.Duration `mapstructure:"RPCReadTimeout"`
}
