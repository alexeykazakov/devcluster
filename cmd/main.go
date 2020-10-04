package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/codeready-toolchain/devcluster/pkg/mongodb"

	"github.com/codeready-toolchain/devcluster/pkg/cluster"

	"github.com/codeready-toolchain/devcluster/pkg/configuration"
	"github.com/codeready-toolchain/devcluster/pkg/log"
	"github.com/codeready-toolchain/devcluster/pkg/server"
)

func main() {
	// create logger and registry
	log.Init("devcluster-service")

	config := configuration.New()
	disconnect, err := mongodb.InitDefaultClient(config)
	if err != nil {
		panic(err.Error())
	}
	cleanupDatabase()
	defer disconnect()

	cluster.InitDefaultClusterService(config)
	cluster.DefaultClusterService.StartDeletingExpiredClusters(600) // Re-check every 10 minutes
	// If there are still provisioning requests left from previous sessions then resume them
	err = cluster.DefaultClusterService.ResumeProvisioningRequests()
	if err != nil {
		panic(err.Error())
	}

	srv := server.New(config)

	err = srv.SetupRoutes()
	if err != nil {
		panic(err.Error())
	}

	routesToPrint := srv.GetRegisteredRoutes()
	log.Infof(nil, "Configured routes: %s", routesToPrint)

	log.Infof(nil, "Creating users...")
	_, err = cluster.DefaultClusterService.CreateUsers(50, 1000)
	if err != nil {
		log.Error(nil, err, "error creating users")
		panic(err.Error())
	}

	// listen concurrently to allow for graceful shutdown
	go func() {
		log.Infof(nil, "Service Revision %s built on %s", configuration.Commit, configuration.BuildTime)
		log.Infof(nil, "Listening on %q...", srv.Config().GetHTTPAddress())
		if err := srv.HTTPServer().ListenAndServe(); err != nil {
			log.Error(nil, err, err.Error())
		}
	}()

	gracefulShutdown(srv.HTTPServer(), srv.Config().GetGracefulTimeout())
}

func cleanupDatabase() {
	names, err := mongodb.Devcluster().ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		panic(err)
	}
	for _, name := range names {
		err = mongodb.Devcluster().Collection(name).Drop(context.Background())
		if err != nil {
			panic(err)
		}
	}
}

func gracefulShutdown(hs *http.Server, timeout time.Duration) {
	// For a channel used for notification of just one signal value, a buffer of
	// size 1 is sufficient.
	stop := make(chan os.Signal, 1)

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM
	// (Ctrl+/). SIGKILL, SIGQUIT will not be caught.
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	sigReceived := <-stop
	log.Infof(nil, "Signal received: %+v", sigReceived.String())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log.Infof(nil, "Shutdown with timeout: %s", timeout.String())
	if err := hs.Shutdown(ctx); err != nil {
		log.Errorf(nil, err, "Shutdown error")
	} else {
		log.Info(nil, "Server stopped.")
	}
}
