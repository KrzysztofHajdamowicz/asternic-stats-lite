// Package store writes queue_log events into the Asternic Stats Lite MySQL
// schema (tables qname, qagent, qevent and queue_stats, unchanged) and keeps
// the read position in an extra table, parselog_state, in the same
// transaction as the events, so every line is stored exactly once.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/KrzysztofHajdamowicz/asternic-stats-lite/parselogd/internal/queuelog"
	"github.com/KrzysztofHajdamowicz/asternic-stats-lite/parselogd/internal/tail"
)

const dateTimeLayout = "2006-01-02 15:04:05"

// fieldLen is the width of the varchar columns in the Asternic schema.
const fieldLen = 40

const createState = `CREATE TABLE IF NOT EXISTS parselog_state (
  id tinyint unsigned NOT NULL,
  inode bigint unsigned NOT NULL,
  file_offset bigint NOT NULL,
  head_len int NOT NULL,
  head_sum char(64) NOT NULL,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
)`

// Options change how events are stored.
type Options struct {
	// ConvertLocal stores Local/XXX@context agents as Agent/XXX.
	ConvertLocal bool
	// Location is the time zone queue_stats.datetime is written in. The PHP
	// application reads it as local time, so this must match the time zone
	// the old parselog ran with. Defaults to time.Local.
	Location *time.Location
}

// Store is a connection to the Asternic database.
type Store struct {
	db     *sql.DB
	opts   Options
	log    *slog.Logger
	events map[string]int64 // qevent.event -> event_id
	queues map[string]int64 // qname.queue -> qname_id
	agents map[string]int64 // qagent.agent -> agent_id
}

// Open connects to the database, creates the parselog_state table if needed
// and loads the event dictionary.
func Open(ctx context.Context, db *sql.DB, opts Options, log *slog.Logger) (*Store, error) {
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	s := &Store{db: db, opts: opts, log: log, queues: map[string]int64{}, agents: map[string]int64{}}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	// Only create the table when it is missing: the database user from the
	// installation guide has no CREATE privilege.
	if _, err := db.ExecContext(ctx, "SELECT 1 FROM parselog_state LIMIT 0"); err != nil {
		if _, err := db.ExecContext(ctx, createState); err != nil {
			return nil, fmt.Errorf("create parselog_state (add it from sql/qstats.sql if the user may not create tables): %w", err)
		}
	}
	if err := s.loadEvents(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadEvents(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "SELECT event_id, event FROM qevent")
	if err != nil {
		return fmt.Errorf("load qevent: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Err is checked
	s.events = map[string]int64{}
	for rows.Next() {
		var id int64
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		s.events[name.String] = id
	}
	return rows.Err()
}

// Position returns the saved read position, or nil if there is none.
func (s *Store) Position(ctx context.Context) (*tail.Position, error) {
	var p tail.Position
	err := s.db.QueryRowContext(ctx,
		"SELECT inode, file_offset, head_len, head_sum FROM parselog_state WHERE id = 1").
		Scan(&p.Inode, &p.Offset, &p.HeadLen, &p.HeadSum)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// LastEventTime returns the time of the newest stored event, if any. It is
// used on the first start to skip lines the PHP parselog already imported.
func (s *Store) LastEventTime(ctx context.Context) (time.Time, bool, error) {
	var v sql.NullString
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(datetime) FROM queue_stats").Scan(&v); err != nil {
		return time.Time{}, false, err
	}
	if !v.Valid {
		return time.Time{}, false, nil
	}
	t, err := time.ParseInLocation(dateTimeLayout, v.String, s.opts.Location)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse queue_stats.datetime %q: %w", v.String, err)
	}
	return t, true, nil
}

// Save stores the events of lines and the position after the last line in
// one transaction. Lines that are not events are skipped, as are events
// older than notBefore (Unix time; 0 disables the check). It returns the
// number of events inserted.
func (s *Store) Save(ctx context.Context, lines []tail.Line, notBefore int64) (n int, err error) {
	if len(lines) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	// Dictionary rows created in this transaction only become visible to
	// later batches once it commits.
	newQueues, newAgents := map[string]int64{}, map[string]int64{}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	insert, err := tx.PrepareContext(ctx, `INSERT IGNORE INTO queue_stats
		(uniqueid, datetime, qname, qagent, qevent, info1, info2, info3)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insert.Close() //nolint:errcheck // closed with the transaction

	for _, l := range lines {
		e, perr := queuelog.Parse(l.Text)
		if perr != nil || e.Time < notBefore {
			continue
		}
		if s.opts.ConvertLocal {
			e.Agent = queuelog.ConvertLocal(e.Agent)
		}
		// Like the PHP parselog, queues and agents are registered even for
		// events that are not stored, so they show up in the report filters.
		qid, err := s.dictID(ctx, tx, s.queues, newQueues, "qname", "qname_id", "queue", e.Queue)
		if err != nil {
			return 0, err
		}
		aid, err := s.dictID(ctx, tx, s.agents, newAgents, "qagent", "agent_id", "agent", e.Agent)
		if err != nil {
			return 0, err
		}
		eid, ok := s.events[e.Event]
		if !ok {
			continue
		}
		res, err := insert.ExecContext(ctx,
			truncate(e.UniqueID),
			time.Unix(e.Time, 0).In(s.opts.Location).Format(dateTimeLayout),
			qid, aid, eid,
			truncate(e.Data1), truncate(e.Data2), truncate(e.Data3))
		if isDataError(err) {
			s.log.Warn("skipping line the database rejects", "line", l.Text, "error", err)
			continue
		}
		if err != nil {
			return 0, err
		}
		if c, _ := res.RowsAffected(); c > 0 {
			n++
		}
	}

	p := lines[len(lines)-1].Pos
	if _, err := tx.ExecContext(ctx, `INSERT INTO parselog_state (id, inode, file_offset, head_len, head_sum)
		VALUES (1, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE inode = VALUES(inode), file_offset = VALUES(file_offset),
		head_len = VALUES(head_len), head_sum = VALUES(head_sum)`,
		p.Inode, p.Offset, p.HeadLen, p.HeadSum); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for k, v := range newQueues {
		s.queues[k] = v
	}
	for k, v := range newAgents {
		s.agents[k] = v
	}
	return n, nil
}

// dictID returns the id of name in a dictionary table (qname or qagent),
// inserting it if it is new. An empty name maps to 0, as in the PHP code.
func (s *Store) dictID(ctx context.Context, tx *sql.Tx, cache, pending map[string]int64, table, idCol, nameCol, name string) (int64, error) {
	if name == "" {
		return 0, nil
	}
	name = truncate(name)
	if id, ok := cache[name]; ok {
		return id, nil
	}
	if id, ok := pending[name]; ok {
		return id, nil
	}
	var id int64
	// Table and column names are constants from this file, never input.
	err := tx.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? ORDER BY %s LIMIT 1", idCol, table, nameCol, idCol), name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		res, ierr := tx.ExecContext(ctx, fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (?)", table, nameCol), name)
		if ierr != nil {
			return 0, fmt.Errorf("insert into %s: %w", table, ierr)
		}
		if id, err = res.LastInsertId(); err != nil {
			return 0, err
		}
		pending[name] = id
		return id, nil
	}
	if err != nil {
		return 0, fmt.Errorf("look up %s: %w", table, err)
	}
	cache[name] = id
	return id, nil
}

// Purge deletes all imported data and the saved position, like
// `parselog.php purge`.
func (s *Store) Purge(ctx context.Context) error {
	for _, table := range []string{"qname", "qagent", "queue_stats", "parselog_state"} {
		//nolint:gosec // table names are constants
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("purge %s: %w", table, err)
		}
	}
	s.queues, s.agents = map[string]int64{}, map[string]int64{}
	return nil
}

// truncate makes s valid UTF-8 and cuts it to the varchar(40) columns of the
// schema. Strict SQL mode would otherwise reject the whole row.
func truncate(s string) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if len(s) <= fieldLen {
		return s
	}
	r := []rune(s)
	if len(r) <= fieldLen {
		return s
	}
	return string(r[:fieldLen])
}

// isDataError reports errors caused by the content of a single row (bad
// character set, value out of range), which would fail on every retry.
func isDataError(err error) bool {
	var me *mysql.MySQLError
	if !errors.As(err, &me) {
		return false
	}
	switch me.Number {
	case 1264, // ER_WARN_DATA_OUT_OF_RANGE
		1265, // WARN_DATA_TRUNCATED
		1292, // ER_TRUNCATED_WRONG_VALUE
		1366, // ER_TRUNCATED_WRONG_VALUE_FOR_FIELD
		1406: // ER_DATA_TOO_LONG
		return true
	}
	return false
}
