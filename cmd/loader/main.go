// / Command loader ingests CSV files exported from the NTSB MDB into Postgres.
//
// Usage:
//
//	DATABASE_URL=postgres://... loader --table events    --source data/events.csv
//	DATABASE_URL=postgres://... loader --table aircraft  --source data/aircraft.csv
//
// Tables must be loaded in dependency order: events first, then aircraft.
// Subsequent tables (narratives, findings, events_sequence) will depend
// on aircraft.
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
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/config"
	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/db"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/load"
	"github.com/kirkspaseff/aceshigh/internal/stats"
	"github.com/kirkspaseff/aceshigh/internal/transform"
)

// tableLoader is the per-table read+transform+load entry point.
// Each registered loader is responsible for its own batching, transform
// errors, and metric counters; main.go just dispatches to it.
type tableLoader func(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error

// tables registers a loader for each supported --table value. Adding a
// new table = add an entry here + the corresponding transform/load code.
var tables = map[string]tableLoader{
	"events":          loadEvents,
	"aircraft":        loadAircraft,
	"narratives":      loadNarratives,
	"findings":        loadFindings,
	"events_sequence": loadEventsSequence,
}

func main() {
	var (
		table      = flag.String("table", "", "table to load (required); one of: "+tableNames())
		sourcePath = flag.String("source", "", "path to CSV file (required)")
	)
	flag.Parse()

	if *table == "" {
		fmt.Fprintln(os.Stderr, "error: --table is required")
		flag.Usage()
		os.Exit(2)
	}
	if *sourcePath == "" {
		fmt.Fprintln(os.Stderr, "error: --source is required")
		flag.Usage()
		os.Exit(2)
	}

	loader, ok := tables[*table]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unknown table %q; valid: %s\n", *table, tableNames())
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, cfg, *table, *sourcePath, loader); err != nil {
		log.Fatalf("loader: %v", err)
	}
}

func tableNames() string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func run(ctx context.Context, cfg *config.Config, table, sourcePath string, loader tableLoader) error {
	start := time.Now()
	log.Printf("loader: table=%s source=%s", table, sourcePath)

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
	if err := loader(ctx, pool, f, counter); err != nil {
		// Print stats even on partial failure — they help debug.
		fmt.Println("\nstats (run failed):")
		counter.WriteReport(os.Stdout)
		return err
	}

	log.Printf("loader: complete in %s", time.Since(start))
	fmt.Println("\nstats:")
	counter.WriteReport(os.Stdout)
	return nil
}

// ---------- Per-table loaders ----------
//
// These follow the same shape: stream CSV → transform → bulk-load.
// The pattern is duplicated rather than abstracted because each table
// has a different domain type and the type-specific code makes the
// shape clearer.

func loadEvents(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}

	parseStart := time.Now()
	var rows []*domain.Event
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("events.read_errors")
			return fmt.Errorf("read row: %w", err)
		}
		ev, err := transform.Event(reader, s)
		if err != nil {
			continue // counted in transform
		}
		rows = append(rows, ev)
		s.Inc("events.parsed")
	}
	log.Printf("loader: parsed %d events in %s", len(rows), time.Since(parseStart))

	loadStart := time.Now()
	if err := load.Events(ctx, pool, rows, s); err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	log.Printf("loader: loaded events in %s", time.Since(loadStart))
	return nil
}

func loadAircraft(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}

	parseStart := time.Now()
	var rows []*domain.Aircraft
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("aircraft.read_errors")
			return fmt.Errorf("read row: %w", err)
		}
		a, err := transform.Aircraft(reader, s)
		if err != nil {
			continue // counted in transform
		}
		rows = append(rows, a)
		s.Inc("aircraft.parsed")
	}
	log.Printf("loader: parsed %d aircraft in %s", len(rows), time.Since(parseStart))

	loadStart := time.Now()
	if err := load.Aircraft(ctx, pool, rows, s); err != nil {
		return fmt.Errorf("load aircraft: %w", err)
	}
	log.Printf("loader: loaded aircraft in %s", time.Since(loadStart))
	return nil
}

func loadNarratives(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}

	parseStart := time.Now()
	var rows []*domain.Narrative
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("narratives.read_errors")
			return fmt.Errorf("read row: %w", err)
		}
		n, err := transform.Narrative(reader, s)
		if err != nil {
			continue
		}
		rows = append(rows, n)
		s.Inc("narratives.parsed")
	}
	log.Printf("loader: parsed %d narratives in %s", len(rows), time.Since(parseStart))

	loadStart := time.Now()
	if err := load.Narratives(ctx, pool, rows, s); err != nil {
		return fmt.Errorf("load narratives: %w", err)
	}
	log.Printf("loader: loaded narratives in %s", time.Since(loadStart))
	return nil
}

func loadFindings(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}

	parseStart := time.Now()
	var rows []*domain.Finding
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("findings.read_errors")
			return fmt.Errorf("read row: %w", err)
		}
		f, err := transform.Finding(reader, s)
		if err != nil {
			continue
		}
		rows = append(rows, f)
		s.Inc("findings.parsed")
	}
	log.Printf("loader: parsed %d findings in %s", len(rows), time.Since(parseStart))

	loadStart := time.Now()
	if err := load.Findings(ctx, pool, rows, s); err != nil {
		return fmt.Errorf("load findings: %w", err)
	}
	log.Printf("loader: loaded findings in %s", time.Since(loadStart))
	return nil
}

func loadEventsSequence(ctx context.Context, pool *pgxpool.Pool, r io.Reader, s *stats.Counter) error {
	reader, err := acsv.NewReader(r)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}

	parseStart := time.Now()
	var rows []*domain.EventSequence
	for {
		err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.Inc("events_sequence.read_errors")
			return fmt.Errorf("read row: %w", err)
		}
		e, err := transform.EventSequence(reader, s)
		if err != nil {
			continue
		}
		rows = append(rows, e)
		s.Inc("events_sequence.parsed")
	}
	log.Printf("loader: parsed %d events_sequence rows in %s", len(rows), time.Since(parseStart))

	loadStart := time.Now()
	if err := load.EventsSequence(ctx, pool, rows, s); err != nil {
		return fmt.Errorf("load events_sequence: %w", err)
	}
	log.Printf("loader: loaded events_sequence in %s", time.Since(loadStart))
	return nil
}
