package server

import "github.com/0xPolygonHermez/zkevm-pool-manager/config/types"

// Config for pool-manager JSON-RPC server
type Config struct {
	// Host defines the network adapter that will be used to serve the HTTP requests
	Host string `mapstructure:"Host"`

	// Port defines the port to serve the endpoints via HTTP
	Port int `mapstructure:"Port"`

	// ReadTimeout is the HTTP server read timeout. Check net/http.server.ReadTimeout and net/http.server.ReadHeaderTimeout
	ReadTimeout types.Duration `mapstructure:"ReadTimeout"`

	// WriteTimeout is the HTTP server write timeout. Check net/http.server.WriteTimeout
	WriteTimeout types.Duration `mapstructure:"WriteTimeout"`

	// MaxRequestsPerIPAndSecond defines how much requests a single IP can send within a single second
	MaxRequestsPerIPAndSecond float64 `mapstructure:"MaxRequestsPerIPAndSecond"`

	// EnableHttpLog allows the user to enable or disable the logs related to the HTTP requests to be captured by the server.
	EnableHttpLog bool `mapstructure:"EnableHttpLog"`

	// BatchRequestsEnabled defines if the batch requests are enabled or disabled
	BatchRequestsEnabled bool `mapstructure:"BatchRequestsEnabled"`

	// BatchRequestsLimit defines the limit of requests that can be incorporated into each batch request
	BatchRequestsLimit uint `mapstructure:"BatchRequestsLimit"`
}
