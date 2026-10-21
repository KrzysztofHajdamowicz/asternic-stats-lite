// Package queuelog parses lines of the Asterisk queue_log file.
//
// A line looks like:
//
//	1729500000|1729499990.42|support|SIP/101|CONNECT|10|1729499990.43|3
//
// i.e. timestamp|uniqueid|queue|agent|event|data1|data2|data3. Newer Asterisk
// versions log more data fields; like the original PHP parselog, everything
// after the seventh separator is kept in Data3.
package queuelog

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Entry is one parsed queue_log line.
type Entry struct {
	Time     int64 // Unix timestamp
	UniqueID string
	Queue    string
	Agent    string
	Event    string
	Data1    string
	Data2    string
	Data3    string
}

// ErrSkip is returned for lines that carry no event, e.g. empty lines or
// lines whose timestamp is not a plain number.
var ErrSkip = errors.New("not a queue_log event line")

// Parse parses a single line (with or without the trailing newline).
func Parse(line string) (Entry, error) {
	// Same characters PHP's rtrim() strips by default.
	line = strings.TrimRight(line, " \t\n\r\x00\x0B")

	parts := strings.SplitN(line, "|", 8)
	for len(parts) < 8 {
		parts = append(parts, "")
	}

	ts := parts[0]
	if ts == "" || strings.IndexFunc(ts, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return Entry{}, ErrSkip
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return Entry{}, ErrSkip
	}

	return Entry{
		Time:     t,
		UniqueID: parts[1],
		Queue:    parts[2],
		Agent:    parts[3],
		Event:    parts[4],
		Data1:    parts[5],
		Data2:    parts[6],
		Data3:    parts[7],
	}, nil
}

var localChannel = regexp.MustCompile(`(?i)^Local/([^@]*)`)

// ConvertLocal rewrites "Local/101@from-queue/n" to "Agent/101", matching the
// "convertlocal" mode of the original parselog.
func ConvertLocal(agent string) string {
	if m := localChannel.FindStringSubmatch(agent); m != nil {
		return "Agent/" + m[1]
	}
	return agent
}
