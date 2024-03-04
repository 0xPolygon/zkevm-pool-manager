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

[Pool]
User = "pool_user"
Password = "pool_password"
Name = "pool_db"
Host = "zkevm-pool-db"
Port = "5432"
EnableLog = false
MaxConns = 200
`
