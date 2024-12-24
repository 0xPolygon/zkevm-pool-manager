package config

// DefaultValues is the default configuration
const DefaultValues = `
[Log]
Environment = "development" # "production" or "development"
Level = "info"
Outputs = ["stderr"]

[Server]
Host = "0.0.0.0"
Port = 8545
ReadTimeout = "60s"
WriteTimeout = "60s"
MaxRequestsPerIPAndSecond = 500
EnableHttpLog = true
BatchRequestsEnabled = false
BatchRequestsLimit = 20

[DB]
User = "pool_user"
Password = "pool_password"
Name = "pool_db"
Host = "zkevm-pool-db"
Port = "5432"
EnableLog = false
MaxConns = 200
QueryLimit = 10000

[Sender]
SequencerURL = "http://localhost:8467"
ResendTxsCheckInterval = "5s"
Workers = 5
QueueSize = 25

[Monitor]
L2NodeURL = "http://localhost:8467"
Workers = 5
QueueSize = 25
RetryWaitInterval = "3s"
InitialWaitInterval = "3s"
TxLifeTimeMax = "30m"

[Metrics]
Host = "0.0.0.0"
Port = 9091
Enabled = false
ProfilingHost = "0.0.0.0"
ProfilingPort = 6060
ProfilingEnabled = false
`
