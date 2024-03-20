package sender

import "github.com/0xPolygonHermez/zkevm-pool-manager/config/types"

// Config for pool-manager sender
type Config struct {
	// SequencerURL defines the URL for the sequencer RPC where the sender will send the pending txs
	SequencerURL string `mapstructure:"SequencerURL"`

	// ResendTxsCheckInterval is the time the sender waits to check in there are new txs in the pool
	ResendTxsCheckInterval types.Duration `mapstructure:"ResendTxsCheckInterval"`

	// Workers is the number of sender workers to send txs to the sequencer
	Workers uint16 `mapstructure:"Workers"`

	// QueueSize is the size of the queue for L2 txs that need to be sent to the sequencer
	QueueSize uint16 `mapstructure:"QueueSize"`
}
