package config

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/metrics"
	"github.com/0xPolygonHermez/zkevm-pool-manager/monitor"
	"github.com/0xPolygonHermez/zkevm-pool-manager/sender"
	server "github.com/0xPolygonHermez/zkevm-pool-manager/server"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

const (
	// FlagYes is the flag for yes.
	FlagYes = "yes"
	// FlagCfg is the flag for cfg.
	FlagCfg = "cfg"
	// FlagNoMigrations is the flag for migrations.
	FlagNoMigrations = "no-migrations"
	// FlagHTTPAPI is the flag for http.api.
	FlagHTTPAPI = "http.api"
)

type Config struct {
	// Log configuration
	Log log.Config

	// Server configuration
	Server server.Config

	// DB configuration
	DB db.Config

	// Sender configuration
	Sender sender.Config

	// Monitor configuration
	Monitor monitor.Config

	// Metrics configuration
	Metrics metrics.Config
}

// Default parses the default configuration values.
func Default() (*Config, error) {
	var cfg Config
	viper.SetConfigType("toml")

	err := viper.ReadConfig(bytes.NewBuffer([]byte(DefaultValues)))
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Load loads the configuration
func Load(ctx *cli.Context, loadNetworkConfig bool) (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, err
	}
	configFilePath := ctx.String(FlagCfg)
	if configFilePath != "" {
		dirName, fileName := filepath.Split(configFilePath)

		fileExtension := strings.TrimPrefix(filepath.Ext(fileName), ".")
		fileNameWithoutExtension := strings.TrimSuffix(fileName, "."+fileExtension)

		viper.AddConfigPath(dirName)
		viper.SetConfigName(fileNameWithoutExtension)
		viper.SetConfigType(fileExtension)
	}
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("ZKEVM_POOL_MANAGER")
	err = viper.ReadInConfig()
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if ok {
			log.Infof("config file not found")
		} else {
			log.Infof("error reading config file: ", err)
			return nil, err
		}
	}

	decodeHooks := []viper.DecoderConfigOption{
		// this allows arrays to be decoded from env var separated by ",", example: MY_VAR="value1,value2,value3"
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc(), mapstructure.StringToSliceHookFunc(","))),
	}

	err = viper.Unmarshal(&cfg, decodeHooks...)
	if err != nil {
		return nil, err
	}

	validate(cfg)

	return cfg, nil
}

func validate(cfg *Config) {
	if cfg.Monitor.QueueSize < cfg.Monitor.Workers {
		log.Fatalf("invalid configuration: Monitor.QueueSize must be greater or equal than Monitor.NumberWorkers")
	}
	if cfg.Monitor.QueueSize < cfg.Monitor.Workers {
		log.Fatalf("invalid configuration: Monitor.QueueSize must be greater or equal than Monitor.NumberWorkers")
	}
}
