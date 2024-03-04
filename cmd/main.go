package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/0xPolygonHermez/zkevm-data-streamer"
	version "github.com/0xPolygonHermez/zkevm-pool-manager"
	"github.com/0xPolygonHermez/zkevm-pool-manager/config"
	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/0xPolygonHermez/zkevm-pool-manager/server"
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
		Aliases:  []string{"no-mig"},
		Usage:    "Disable run migrations in pool database",
		Required: false,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = appName
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
			Usage:   "Run zkevm-pool-manager",
			Action:  start,
			Flags:   append(flags, &migrationsFlag),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func start(cliCtx *cli.Context) error {
	c, err := config.Load(cliCtx, true)
	if err != nil {
		return err
	}
	setupLog(c.Log)

	if c.Log.Environment == log.EnvironmentDevelopment {
		zkevm.PrintVersion(os.Stdout)
		log.Info("starting application...")
	} else if c.Log.Environment == log.EnvironmentProduction {
		logVersion()
	}

	// Only runs migration if the component is the synchronizer and if the flag is deactivated
	if !cliCtx.Bool(config.FlagNoMigrations) {
		log.Infof("running database migrations, host: %s:%s, db: %s, user: %s", c.DB.Host, c.DB.Port, c.DB.Name, c.DB.User)
		runPoolMigrations(c.DB)
	}
	checkPoolMigrations(c.DB)

	var cancelFuncs []context.CancelFunc

	go runPoolManagerServer(*c)

	waitSignal(cancelFuncs)

	return nil
}

func versionCmd(*cli.Context) error {
	zkevm.PrintVersion(os.Stdout)
	return nil
}

func setupLog(c log.Config) {
	log.Init(c)
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

func runPoolManagerServer(cfg config.Config) {
	poolDB, err := db.NewPoolDB(cfg.DB)
	if err != nil {
		log.Fatalf("error when creating PoolDB instance, err: %v", err)
	}

	err = server.NewServer(cfg.Server, poolDB).Start()
	if err != nil {
		log.Fatalf("error when creating pool manager server, err: %v", err)
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
