package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/rtrzebinski/simple-memorizer-4/internal/backend"
	"github.com/rtrzebinski/simple-memorizer-4/internal/backend/storage/postgres"
	"github.com/rtrzebinski/simple-memorizer-4/internal/frontend/components"
	"github.com/rtrzebinski/simple-memorizer-4/internal/mysignal"
	"github.com/rtrzebinski/simple-memorizer-4/internal/probes"
	"log"
	"net/http"
	"time"
)

type config struct {
	Db struct {
		Driver string `envconfig:"DB_DRIVER" default:"postgres"`
		DSN    string `envconfig:"DB_DSN" default:"postgres://postgres:postgres@localhost:5430/postgres?sslmode=disable"`
	}
	Api struct {
		Port string `envconfig:"API_PORT" default:":8000"`
	}
	Web struct {
		ProbeAddr       string        `envconfig:"WEB_PROBE_HOST" default:"0.0.0.0:9090"`
		ShutdownTimeout time.Duration `envconfig:"WEB_SHUTDOWN_TIMEOUT" default:"30s"`
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		panic(err)
	}
}

// The main function is the entry point where the app is configured and started.
// It is executed in 2 different environments: A client (the web browser) and a
// server.
func run(ctx context.Context) error {
	log.Println("application starting")

	// The first thing to do is to associate the home component with a path.
	//
	// This is done by calling the Route() function,  which tells go-app what
	// component to display for a given path, on both client and server-side.
	app.Route("/", &components.Home{})

	// Associate other frontend routes
	app.Route("/exercises", &components.Exercises{})

	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// When executed on the server-side, RunWhenOnBrowser() does nothing, which
	// lets room for server implementation without the need for pre compiling
	// instructions.
	app.RunWhenOnBrowser()

	// Handle home page
	http.Handle("/", &app.Handler{
		Name:        "Home",
		Description: "Home page",
		Icon: app.Icon{
			Default:    "/web/logo-192.png",
			Large:      "/web/logo-512.png",
			AppleTouch: "/web/logo-192.png",
		},
		Scripts: []string{
			"/web/swiped-events.js",
		},
		Styles: []string{
			// todo find a way to only load on a learning page
			//"/web/hello.css",
		},
	})

	// Configuration
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	// Database connection
	db, err := sql.Open(cfg.Db.Driver, cfg.Db.DSN)
	if err != nil {
		return err
	}

	// Dependencies
	r := postgres.NewReader(db)
	w := postgres.NewWriter(db)

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	probeServer := probes.SetupProbeServer(cfg.Web.ProbeAddr, db)

	// Start probe server and send errors to the channel
	go func() {
		log.Printf("initializing probe server on host: %s", cfg.Web.ProbeAddr)
		serverErrors <- probeServer.ListenAndServe()
	}()

	// Start API server and send errors to the channel
	go func() {
		log.Printf("initializing API server on port: %s", cfg.Api.Port)
		serverErrors <- backend.ListenAndServe(r, w, cfg.Api.Port)
	}()

	// Signal notifier
	done := mysignal.NewNotifier(ctx)

	log.Println("application running")

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-done.Done():
		log.Print("start shutdown")

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		// Shutdown gracefully on signal received
		if err := probeServer.Shutdown(ctx); err != nil {
			log.Print(fmt.Errorf("failed to gracefully shutdown the probe server %w", err))

			if err = probeServer.Close(); err != nil {
				return fmt.Errorf("could not stop probe server gracefully: %w", err)
			}
		}
	}

	return nil
}
