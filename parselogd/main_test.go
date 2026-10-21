package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KrzysztofHajdamowicz/asternic-stats-lite/parselogd/internal/testdb"
)

const report = `SELECT qs.datetime, qs.uniqueid, q.queue, a.agent, e.event,
	IFNULL(qs.info1, ''), IFNULL(qs.info2, ''), IFNULL(qs.info3, '')
	FROM queue_stats qs
	JOIN qname q ON q.qname_id = qs.qname
	JOIN qagent a ON a.agent_id = qs.qagent
	JOIN qevent e ON e.event_id = qs.qevent
	ORDER BY qs.queue_stats_id`

func TestMain(m *testing.M) {
	// testdata/queue_stats.golden was produced by parselog.php with this zone.
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	}
	time.Local = loc
	os.Exit(m.Run())
}

func testConfig(dsn, queueLog string) config {
	return config{queueLog: queueLog, dsn: dsn, convertLocal: true, poll: 10 * time.Millisecond, batch: 7, once: true}
}

func runOnce(t *testing.T, cfg config) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if err := run(context.Background(), cfg, log); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func queryReport(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(report)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() //nolint:errcheck // test
	var out []string
	for rows.Next() {
		f := make([]string, 8)
		ptrs := make([]any, len(f))
		for i := range f {
			ptrs[i] = &f[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		out = append(out, strings.Join(f, "\t"))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

func compare(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d rows, want %d", len(got), len(want))
	}
	for i := 0; i < len(got) || i < len(want); i++ {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			t.Errorf("row %d:\n got  %q\n want %q", i+1, g, w)
		}
	}
}

func TestImportMatchesGolden(t *testing.T) {
	dsn := testdb.New(t)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test

	dir := t.TempDir()
	queueLog := filepath.Join(dir, "queue_log")
	copyFile(t, "testdata/queue_log", queueLog)
	golden, err := os.ReadFile("testdata/queue_stats.golden")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimRight(string(golden), "\n"), "\n")

	cfg := testConfig(dsn, queueLog)
	runOnce(t, cfg)
	compare(t, queryReport(t, db), want)

	// Running again imports nothing twice.
	runOnce(t, cfg)
	compare(t, queryReport(t, db), want)

	// New lines, a partial line, then rotation by rename.
	appendFile(t, queueLog, "1729498700|1729498699.60|support|NONE|ENTERQUEUE||500100203|1\n1729498760|1729498699.60|sup")
	runOnce(t, cfg)
	appendFile(t, queueLog, "port|NONE|ABANDON|1|1|60\n")
	if err := os.Rename(queueLog, queueLog+".1"); err != nil {
		t.Fatal(err)
	}
	appendFile(t, queueLog, "1729498800|NONE|sales|SIP/202|AGENTLOGIN|\n")
	runOnce(t, cfg)

	want = append(want,
		"2024-10-21 10:18:20\t1729498699.60\tsupport\tNONE\tENTERQUEUE\t\t500100203\t1",
		"2024-10-21 10:19:20\t1729498699.60\tsupport\tNONE\tABANDON\t1\t1\t60",
		"2024-10-21 10:20:00\tNONE\tsales\tSIP/202\tAGENTLOGIN\t\t\t",
	)
	compare(t, queryReport(t, db), want)

	var queues, agents int
	if err := db.QueryRow("SELECT COUNT(*) FROM qname").Scan(&queues); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM qagent").Scan(&agents); err != nil {
		t.Fatal(err)
	}
	if queues != 4 || agents != 4 {
		t.Errorf("qname rows = %d, qagent rows = %d, want 4 and 4", queues, agents)
	}
}

// After the PHP parselog the database has events but no saved position:
// only events from 10 s before the newest stored one onwards are imported.
func TestFirstStartAfterPHPParselog(t *testing.T) {
	dsn := testdb.New(t)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test

	if _, err := db.Exec(`INSERT INTO queue_stats (uniqueid, datetime, qname, qagent, qevent)
		VALUES ('old', '2024-10-21 10:10:30', 0, 0, 0)`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	queueLog := filepath.Join(dir, "queue_log")
	copyFile(t, "testdata/queue_log", queueLog)
	runOnce(t, testConfig(dsn, queueLog))

	got := queryReport(t, db)
	if len(got) == 0 || !strings.HasPrefix(got[0], "2024-10-21 10:10:30\t1729498199.40") {
		t.Fatalf("first imported row = %q, want the EXITWITHTIMEOUT at 10:10:30", got)
	}
	if len(got) != 7 {
		t.Errorf("imported %d rows, want 7", len(got))
	}
}

func TestPurge(t *testing.T) {
	dsn := testdb.New(t)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test

	queueLog := filepath.Join(t.TempDir(), "queue_log")
	copyFile(t, "testdata/queue_log", queueLog)
	cfg := testConfig(dsn, queueLog)
	runOnce(t, cfg)

	cfg.purge = true
	runOnce(t, cfg)
	for _, table := range []string{"queue_stats", "qname", "qagent", "parselog_state"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s has %d rows after purge", table, n)
		}
	}
}

func TestParseFlags(t *testing.T) {
	t.Setenv("dbhost", "database")
	t.Setenv("dbuser", "u")
	t.Setenv("dbpass", "p@ss")
	t.Setenv("dbname", "qstats")
	t.Setenv("PARSELOG_DSN", "")
	cfg, err := parseFlags([]string{"convertlocal"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.convertLocal {
		t.Error("convertlocal argument not honoured")
	}
	if want := "u:p@ss@tcp(database:3306)/qstats"; !strings.HasPrefix(cfg.dsn, want) {
		t.Errorf("dsn = %q, want prefix %q", cfg.dsn, want)
	}
	if _, err := parseFlags([]string{"bogus"}); err == nil {
		t.Error("unknown argument accepted")
	}
}
