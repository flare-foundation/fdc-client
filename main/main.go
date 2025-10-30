package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"

	"github.com/flare-foundation/fdc-client/client/collector"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/manager"
	"github.com/flare-foundation/fdc-client/client/shared"
	"github.com/flare-foundation/fdc-client/client/timing"
	"github.com/flare-foundation/fdc-client/server"
)

const (
	systemDirectory string = "configs/systemConfigs" // relative to project root
)

var CfgFlag = flag.String("config", "configs/userConfig.toml", "Configuration file (toml format)")

func main() {
	flag.Parse()
	userConfigRaw, systemConfig, err := config.ReadConfigs(*CfgFlag, systemDirectory)
	if err != nil {
		logger.Panicf("cannot read configs: %s", err)
	}
	logger.Set(userConfigRaw.Logging)

	err = timing.Set(systemConfig.Timing)
	if err != nil {
		logger.Panicf("cannot set timing: %s", err)
	}

	attestationTypeConfig, err := config.ParseAttestationTypes(userConfigRaw.AttestationTypeConfig)
	if err != nil {
		logger.Panicf("att types config: %s", err)
	}

	// Prepare context, that can cancel all goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// Prepare shared data connections that collector, manager and server will use
	sharedDataPipes := shared.NewDataPipes()

	// Start attestation client collector
	col := collector.New(userConfigRaw, systemConfig, sharedDataPipes)
	col.WaitForDBToSync(ctx)
	go col.Run(ctx)

	// Start attestation client manager
	mngr, err := manager.New(userConfigRaw, attestationTypeConfig, sharedDataPipes)
	if err != nil {
		logger.Panicf("failed to create the manager: %s", err)
	}
	go mngr.Run(ctx)

	// Run attestation client server
	srv := server.New(&sharedDataPipes.Rounds, userConfigRaw.ProtocolID, userConfigRaw.RestServer)
	go srv.Run(ctx)
	logger.Info("Running server")

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, os.Interrupt, syscall.SIGTERM)
	// Block until a termination signal is received.
	select {
	case <-cancelChan:
		logger.Info("Received an interrupt signal, shutting down after 2 minutes")
		time.Sleep(2 * time.Minute)
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down...")
	}

	cancel()
	srv.Shutdown()
}
