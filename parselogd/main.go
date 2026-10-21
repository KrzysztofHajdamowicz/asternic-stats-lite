// Command parselogd imports the Asterisk queue_log into the Asternic Call
// Center Stats Lite database. It replaces parselog/parselog.php: instead of
// re-reading the whole file from cron, it follows the file like `tail -F`,
// survives log rotation and resumes from the position saved in the database.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata" // the container image has no zoneinfo

	"github.com/go-sql-driver/mysql"

	"github.com/KrzysztofHajdamowicz/asternic-stats-lite/parselogd/internal/store"
	"github.com/KrzysztofHajdamowicz/asternic-stats-lite/parselogd/internal/tail"
)

var version = "dev"

type config struct {
	queueLog     string
	dsn          string
	convertLocal bool
	poll         time.Duration
	grace        time.Duration
	batch        int
	once         bool
	purge        bool
	verbose      bool
}

func main() {
	os.Exit(mainCode())
}

func mainCode() int {
	cfg, err := parseFlags(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	level := slog.LevelInfo
	if cfg.verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("parselogd failed", "error", err)
		return 1
	}
	return 0
}

func parseFlags(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("parselogd", flag.ContinueOnError)
	fs.StringVar(&cfg.queueLog, "queue-log", env("QUEUE_LOG", "/var/log/asterisk/queue_log"), "queue_log file to follow (env QUEUE_LOG)")
	fs.StringVar(&cfg.dsn, "dsn", os.Getenv("PARSELOG_DSN"), "MySQL DSN, e.g. user:pass@tcp(host:3306)/qstatslite (env PARSELOG_DSN; default built from env dbhost, dbuser, dbpass, dbname)")
	fs.BoolVar(&cfg.convertLocal, "convert-local", envBool("CONVERT_LOCAL"), "store Local/XXX@context agents as Agent/XXX (env CONVERT_LOCAL)")
	fs.DurationVar(&cfg.poll, "poll-interval", envDuration("POLL_INTERVAL", time.Second), "how often to check the file for new lines (env POLL_INTERVAL)")
	fs.DurationVar(&cfg.grace, "rotate-grace", envDuration("ROTATE_GRACE", 5*time.Second), "how long to keep reading a rotated file for late writes (env ROTATE_GRACE)")
	fs.IntVar(&cfg.batch, "batch-size", 1000, "maximum lines per database transaction")
	fs.BoolVar(&cfg.once, "once", false, "import what is in the file now and exit, instead of following it")
	fs.BoolVar(&cfg.purge, "purge", false, "delete all imported data and the saved position, then exit")
	fs.BoolVar(&cfg.verbose, "verbose", envBool("VERBOSE"), "log every batch (env VERBOSE)")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if *showVersion {
		fmt.Println("parselogd", version)
		os.Exit(0)
	}
	// Positional arguments of the old parselog.php.
	for _, a := range fs.Args() {
		switch a {
		case "convertlocal":
			cfg.convertLocal = true
		case "purge":
			cfg.purge = true
		default:
			return cfg, fmt.Errorf("unknown argument %q", a)
		}
	}
	if cfg.batch < 1 {
		return cfg, errors.New("-batch-size must be at least 1")
	}
	if cfg.dsn == "" {
		cfg.dsn = dsnFromEnv()
	}
	return cfg, nil
}

// dsnFromEnv builds a DSN from the variables the Docker Compose setup and the
// PHP config already use.
func dsnFromEnv() string {
	c := mysql.NewConfig()
	c.User = os.Getenv("dbuser")
	c.Passwd = os.Getenv("dbpass")
	c.DBName = env("dbname", "qstatslite")
	c.Net = "tcp"
	c.Addr = env("dbhost", "localhost")
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		c.Addr = net.JoinHostPort(c.Addr, "3306")
	}
	return c.FormatDSN()
}

func run(ctx context.Context, cfg config, log *slog.Logger) error {
	db, err := sql.Open("mysql", cfg.dsn)
	if err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	defer db.Close() //nolint:errcheck // shutting down
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Hour)

	opts := store.Options{ConvertLocal: cfg.convertLocal, Location: time.Local}
	var st *store.Store
	err = retry(ctx, log, "connect to database", func() error {
		st, err = store.Open(ctx, db, opts, log)
		return err
	})
	if err != nil {
		return err
	}

	if cfg.purge {
		if err := st.Purge(ctx); err != nil {
			return err
		}
		log.Info("purged qname, qagent, queue_stats and parselog_state")
		return nil
	}

	pos, err := st.Position(ctx)
	if err != nil {
		return fmt.Errorf("read saved position: %w", err)
	}
	var notBefore int64
	if pos == nil {
		// First start, possibly after the PHP parselog: skip what is already
		// imported, with the same 10 s overlap parselog.php used.
		last, ok, err := st.LastEventTime(ctx)
		if err != nil {
			return err
		}
		if ok {
			notBefore = last.Unix() - 10
			log.Info("no saved position, skipping events already in queue_stats", "before", last.Add(-10*time.Second).Format(time.DateTime))
		}
	} else {
		log.Info("resuming", "position", pos.String())
	}

	log.Info("following queue log", "version", version, "path", cfg.queueLog, "convert_local", cfg.convertLocal, "time_zone", time.Local.String())
	fol := tail.New(cfg.queueLog, pos, cfg.grace, log)
	defer fol.Close() //nolint:errcheck // read-only

	var pending []tail.Line
	var lastReadErr string
	backoff := time.Second
	for {
		if len(pending) == 0 {
			pending, err = fol.Read(cfg.batch)
			switch {
			case err != nil && cfg.once:
				return err
			case err != nil && err.Error() != lastReadErr: // log each new problem once
				log.Error("reading queue log", "path", cfg.queueLog, "error", err)
				lastReadErr = err.Error()
			case err == nil && lastReadErr != "":
				log.Info("reading queue log again", "path", cfg.queueLog)
				lastReadErr = ""
			}
		}
		if len(pending) == 0 {
			if cfg.once && err == nil {
				return nil
			}
			if !sleep(ctx, cfg.poll) {
				return ctx.Err()
			}
			continue
		}

		// Let a batch in progress finish on shutdown.
		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		n, err := st.Save(saveCtx, pending, notBefore)
		cancel()
		if err != nil {
			log.Error("saving events, will retry", "error", err, "retry_in", backoff)
			if !sleep(ctx, backoff) {
				return ctx.Err()
			}
			backoff = min(backoff*2, time.Minute)
			continue
		}
		backoff = time.Second
		log.Debug("batch saved", "lines", len(pending), "events", n, "position", pending[len(pending)-1].Pos.String())
		pending = nil
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func retry(ctx context.Context, log *slog.Logger, what string, f func() error) error {
	backoff := time.Second
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Warn("cannot "+what+", retrying", "error", err, "retry_in", backoff)
		if !sleep(ctx, backoff) {
			return ctx.Err()
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string) bool {
	b, _ := strconv.ParseBool(os.Getenv(key))
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return d
	}
	return def
}
