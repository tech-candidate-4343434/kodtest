package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/tech-candidate-4343434/guestbook/internal/api"
	"github.com/tech-candidate-4343434/guestbook/internal/store"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"
)

const migrationsDir = "sql/migrations"

//go:embed sql/migrations/*.sql
var migrations embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	addr := flag.String("addr", ":8080", "http listen address")
	dsn := flag.String("db", "file:guestbook.db", "sqlite database dsn")
	var logLevel slog.Level
	flag.TextVar(&logLevel, "log-level", slog.LevelInfo, "log level")
	flag.Parse()

	if *addr == "" {
		return fmt.Errorf("-addr flag is required and cannot be empty")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	db, err := sql.Open("sqlite", *dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("could not set goose dialect: %w", err)
	}

	goose.SetBaseFS(migrations)
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	mux := http.NewServeMux()
	api.NewHandler(store.New(db)).Routes(mux)
	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("starting server", "addr", srv.Addr)
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down server", "addr", srv.Addr)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
