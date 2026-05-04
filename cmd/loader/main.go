// Usage:
//
//	DATABASE_URL=postgres://... loader --source ./data/events.csv
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kirkspaseff/aceshigh/internal/config"
	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/db"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/load"
	"github.com/kirkspaseff/aceshigh/internal/stats"
	"github.com/kirkspaseff/aceshigh/internal/transform"
)

// batchSize is the number of rows we accumulate in memory before flushing
// to Postgres. ~30k events fits comfortably in one batch; larger datasets
// would benefit from streaming.
const batchSize = 50_000

func main() {
	var (
		sourcePath = flag.String("source", "", "path to events CSV file (required)")
	)
	flag.Parse()

	if *sourcePath == "" {
		fmt.Fprintln(os.Stderr, "error: --source is required")
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Cancel on Ctrl+C / SIGTERM so a long-running load can be interrupted
	// cleanly. The defer cancel() guarantees cleanup if main returns.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, cfg, *sourcePath); err != nil {
		log.Fatalf("loader: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config, sourcePath string) error {
	start := time.Now()
	log.Printf("loader: source=%s", sourcePath)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()
	log.Printf("loader: connected to database")

	f, err := os.Open(filepath.Clean(sourcePath))
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	counter := stats.New()

	events, err := readEvents(f, counter)
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}
	log.Printf("loader: parsed %d events in %s", len(events), time.Since(start))

	loadStart := time.Now()
	if err := load.Events(ctx, pool, events, counter); err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	log.Printf("loader: loaded events in %s", time.Since(loadStart))

	log.Printf("loader: complete in %s", time.Since(start))
	fmt.Println("\nstats:")
	counter.WriteReport(os.Stdout)

	return nil
}

// readEvents streams the CSV, transforms each row, and accumulates the
// results. For 30k rows this is fine in memory; if we ever process larger
// datasets we'd flush to load.Events in batches inside the loop.
func readEvents(r io.Reader, s *stats.Counter) ([]*domain.Event, error) {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return nil, err
	}

	events := make([]*domain.Event, 0, batchSize)
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("events.read_errors")
			return nil, fmt.Errorf("read row: %w", err)
		}

		ev, err := transform.Event(reader, s)
		if err != nil {
			// transform.Event already counted the error; keep going.
			continue
		}
		events = append(events, ev)
		s.Inc("events.parsed")
	}
	return events, nil
}
