package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/0xPolygonHermez/zkevm-data-streamer"
	version "github.com/0xPolygonHermez/zkevm-pool-manager"
	"github.com/0xPolygonHermez/zkevm-pool-manager/config"
	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/metrics"
	"github.com/0xPolygonHermez/zkevm-pool-manager/monitor"
	"github.com/0xPolygonHermez/zkevm-pool-manager/sender"
	server "github.com/0xPolygonHermez/zkevm-pool-manager/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

const appName = "zkevm-pool-manager"

var (
	configFileFlag = cli.StringFlag{
		Name:     config.FlagCfg,
		Aliases:  []string{"c"},
		Usage:    "Configuration `FILE`",
		Required: true,
	}
	migrationsFlag = cli.BoolFlag{
		Name:     config.FlagNoMigrations,
		Aliases:  []string{"n"},
		Usage:    "Disable run migrations in pool database",
		Required: false,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = "zkEVM Pool Manager component"
	app.Version = zkevm.Version
	flags := []cli.Flag{&configFileFlag}
	app.Commands = []*cli.Command{
		{
			Name:    "version",
			Aliases: []string{},
			Usage:   "Application version and build",
			Action:  versionCmd,
		},
		{
			Name:    "run",
			Aliases: []string{},
			Usage:   "Run zkEVM Pool manager",
			Action:  start,
			Flags:   append(flags, &migrationsFlag),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		println()
		println("ERROR:", err.Error())
		os.Exit(1)
	}
}

func start(cliCtx *cli.Context) error {
	// Load config file
	c, err := config.Load(cliCtx, true)
	if err != nil {
		return err
	}

	// Setup logger
	log.Init(c.Log)
	if c.Log.Environment == log.EnvironmentDevelopment {
		zkevm.PrintVersion(os.Stdout)
		log.Info("starting application...")
	} else if c.Log.Environment == log.EnvironmentProduction {
		logVersion()
	}

	// Run migrations if the 'no-migrations' flag is not set
	if !cliCtx.Bool(config.FlagNoMigrations) {
		log.Infof("running database migrations, host: %s:%s, db: %s, user: %s", c.DB.Host, c.DB.Port, c.DB.Name, c.DB.User)
		runPoolMigrations(c.DB)
	}
	checkPoolMigrations(c.DB)

	var cancelFuncs []context.CancelFunc

	poolDB, err := db.NewPoolDB(c.DB)
	if err != nil {
		log.Fatalf("error when creating pool DB instance, error: %v", err)
	}

	monitor := monitor.NewMonitor(c.Monitor, poolDB)
	go monitor.Start()

	sender := sender.NewSender(c.Sender, poolDB, monitor)
	go sender.Start()

	server := server.NewServer(c.Server, poolDB, sender)
	go server.Start()

	if c.Metrics.Enabled {
		go startMetricsHttpServer(c.Metrics)
	}

	if c.Metrics.ProfilingEnabled {
		go startProfilingHttpServer(c.Metrics)
	}

	go func() {
		for {
			select {
			case <-cliCtx.Done():
				log.Infof("Exiting loop...")
				return
			default:
				time.Sleep(30 * time.Second)
				monitor.Summary()
			}
		}
	}()

	waitSignal(cancelFuncs)

	return nil
}

func versionCmd(*cli.Context) error {
	zkevm.PrintVersion(os.Stdout)
	return nil
}

func runPoolMigrations(c db.Config) {
	log.Infof("running database migrations for %v", db.PoolMigrationName)
	err := db.RunMigrationsUp(c, db.PoolMigrationName)
	if err != nil {
		log.Fatal(err)
	}
}

func checkPoolMigrations(c db.Config) {
	err := db.CheckMigrations(c, db.PoolMigrationName)
	if err != nil {
		log.Fatal(err)
	}
}

func waitSignal(cancelFuncs []context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for sig := range signals {
		switch sig {
		case os.Interrupt, os.Kill:
			log.Info("terminating application gracefully...")

			exitStatus := 0
			for _, cancel := range cancelFuncs {
				cancel()
			}
			os.Exit(exitStatus)
		}
	}
}

func logVersion() {
	log.Infow(
		// node version is already logged by default
		"Git revision", version.GitRev,
		"Git branch", version.GitBranch,
		"Go version", runtime.Version(),
		"Built", version.BuildDate,
		"OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
}

func startProfilingHttpServer(c metrics.Config) {
	const two = 2
	mux := http.NewServeMux()
	address := fmt.Sprintf("%s:%d", c.ProfilingHost, c.ProfilingPort)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("failed to create tcp listener for profiling: %v", err)
		return
	}
	mux.HandleFunc(metrics.ProfilingIndexEndpoint, pprof.Index)
	mux.HandleFunc(metrics.ProfileEndpoint, pprof.Profile)
	mux.HandleFunc(metrics.ProfilingCmdEndpoint, pprof.Cmdline)
	mux.HandleFunc(metrics.ProfilingSymbolEndpoint, pprof.Symbol)
	mux.HandleFunc(metrics.ProfilingTraceEndpoint, pprof.Trace)
	profilingServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: two * time.Minute,
		ReadTimeout:       two * time.Minute,
	}
	log.Infof("profiling server listening on port %d", c.ProfilingPort)
	if err := profilingServer.Serve(lis); err != nil {
		if err == http.ErrServerClosed {
			log.Warnf("http server for profiling stopped")
			return
		}
		log.Errorf("closed http connection for profiling server: %v", err)
		return
	}
}

func startMetricsHttpServer(c metrics.Config) {
	const ten = 10
	mux := http.NewServeMux()
	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("failed to create tcp listener for metrics: %v", err)
		return
	}
	mux.Handle(metrics.Endpoint, promhttp.Handler())

	metricsServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: ten * time.Second,
		ReadTimeout:       ten * time.Second,
	}
	log.Infof("metrics server listening on port %d", c.Port)
	if err := metricsServer.Serve(lis); err != nil {
		if err == http.ErrServerClosed {
			log.Warnf("http server for metrics stopped")
			return
		}
		log.Errorf("closed http connection for metrics server: %v", err)
		return
	}
}
