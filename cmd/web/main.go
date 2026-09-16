package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mlkad/groupie-tracker/internal/api"
	"github.com/mlkad/groupie-tracker/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP network address")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*addr = ":" + p
	}

	client := api.NewClient()
	cache := api.NewCache(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cache.Load(ctx); err != nil {
		log.Fatalf("loading data: %v", err)
	}

	srv, err := server.New(cache)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	//graceful shutdown
	httpSrv := &http.Server{
		Addr:         *addr,
		Handler:      srv.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("starting server on %s", *addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM) //os.Interrupt = SIGINT

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	case <-stop:
		log.Println("shutting down")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		log.Println("stopped")
	}
}
