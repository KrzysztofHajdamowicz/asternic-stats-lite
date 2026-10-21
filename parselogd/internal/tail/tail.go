// Package tail follows a growing log file the way `tail -F` does, and reports
// a resumable position with every line.
//
// It polls instead of using inotify so it behaves the same on bind mounts,
// network filesystems and Docker Desktop. Both logrotate styles are handled:
//
//   - rename + create: the old file is read to its end (with a short grace
//     period for late writes), then the new file is read from the start;
//   - copytruncate: a file that shrinks below the read offset is re-read from
//     the start.
//
// A Position identifies the file by inode plus a checksum of its first bytes,
// so a restart finds the right file even if it has been rotated (to
// queue_log.1, queue_log-20241021, ...) while the process was down, and an
// inode reused by an unrelated new file is not mistaken for the old one.
package tail

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	headSize    = 1024     // bytes covered by the file fingerprint
	readSize    = 64 << 10 // read chunk
	maxLineSize = 64 << 10 // longer "lines" are cut and returned as is
)

// Position is a resumable read position.
type Position struct {
	Inode   uint64
	Offset  int64  // byte offset just after the last returned line
	HeadLen int    // number of leading bytes HeadSum covers
	HeadSum string // hex SHA-256 of the first HeadLen bytes of the file
}

// Line is one line of the file, without the trailing newline.
type Line struct {
	Text string
	Pos  Position // position just after this line
}

// Follower reads complete lines from a file and follows it across rotations.
type Follower struct {
	path  string
	grace time.Duration
	log   *slog.Logger
	now   func() time.Time

	resume *Position // start position, consumed by the first open

	f         *os.File
	inode     uint64
	offset    int64  // offset of the next byte to read from f
	buf       []byte // bytes read but not yet returned as lines
	head      []byte // first bytes of f, for fingerprints
	draining  bool   // f is a rotated-away file; move on to path at EOF
	rotatedAt time.Time
}

// New returns a Follower for path. If resume is not nil, reading continues
// from that position; otherwise the current file is read from the start.
// grace is how long a rotated file keeps being read for late writes.
func New(path string, resume *Position, grace time.Duration, log *slog.Logger) *Follower {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Follower{path: path, resume: resume, grace: grace, log: log, now: time.Now}
}

// Close releases the open file.
func (t *Follower) Close() error {
	if t.f == nil {
		return nil
	}
	err := t.f.Close()
	t.f = nil
	return err
}

// Read returns up to max complete lines. It returns no lines and no error
// when there is nothing new; call it again later. A missing file is not an
// error: the follower waits for it to appear.
func (t *Follower) Read(max int) ([]Line, error) {
	if t.f == nil {
		ok, err := t.openInitial()
		if err != nil || !ok {
			return nil, err
		}
	}

	if err := t.checkTruncated(); err != nil {
		return nil, err
	}

	var lines []Line
	for len(lines) < max {
		line, ok, err := t.nextLine()
		if err != nil {
			return lines, err
		}
		if ok {
			lines = append(lines, line)
			continue
		}
		// At EOF of the current file.
		if len(lines) > 0 && !t.draining {
			break
		}
		switched, last, err := t.checkRotation()
		if err != nil {
			return lines, err
		}
		if last != nil {
			lines = append(lines, *last)
		}
		if !switched {
			break
		}
	}
	return lines, nil
}

// nextLine returns the next complete line from buf, reading more data from
// the file as needed. ok is false at EOF.
func (t *Follower) nextLine() (Line, bool, error) {
	for {
		if i := bytes.IndexByte(t.buf, '\n'); i >= 0 || len(t.buf) >= maxLineSize {
			end := i + 1
			text := t.buf[:max(i, 0)]
			if i < 0 {
				end, text = maxLineSize, t.buf[:maxLineSize]
			}
			l := Line{Text: string(text)}
			t.buf = t.buf[end:]
			pos, err := t.position()
			if err != nil {
				return Line{}, false, err
			}
			l.Pos = pos
			return l, true, nil
		}
		n, err := t.fill()
		if err != nil {
			return Line{}, false, err
		}
		if n == 0 {
			return Line{}, false, nil
		}
	}
}

func (t *Follower) fill() (int, error) {
	chunk := make([]byte, readSize)
	n, err := t.f.Read(chunk)
	t.buf = append(t.buf, chunk[:n]...)
	t.offset += int64(n)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

// position is the position just after the bytes returned so far.
func (t *Follower) position() (Position, error) {
	off := t.offset - int64(len(t.buf))
	n := int(min(off, headSize))
	if len(t.head) < n {
		head := make([]byte, n)
		if _, err := t.f.ReadAt(head, 0); err != nil && !errors.Is(err, io.EOF) {
			return Position{}, err
		}
		t.head = head
	}
	return Position{Inode: t.inode, Offset: off, HeadLen: n, HeadSum: checksum(t.head[:n])}, nil
}

// checkRotation runs at EOF. It reports whether the follower moved on to
// another file (so reading should continue), and returns a trailing line
// without newline left over in a file that is being abandoned.
func (t *Follower) checkRotation() (bool, *Line, error) {
	if !t.draining {
		fi, err := os.Stat(t.path)
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil, nil // between rename and create; keep waiting
		}
		if err != nil {
			return false, nil, err
		}
		if inodeOf(fi) == t.inode {
			return false, nil, nil
		}
		t.log.Info("queue log rotated", "path", t.path)
		t.draining = true
		t.rotatedAt = t.now()
	}

	// Draining a rotated file: give the writer a moment to reopen its log.
	if t.now().Sub(t.rotatedAt) < t.grace {
		return false, nil, nil
	}
	var last *Line
	if len(t.buf) > 0 {
		t.buf = append(t.buf, '\n')
		l, _, err := t.nextLine()
		if err != nil {
			return false, nil, err
		}
		last = &l
	}
	ok, err := t.openAt(t.path, 0, false)
	if err != nil || !ok {
		return false, last, err
	}
	return true, last, nil
}

// checkTruncated rewinds to the start if the file was truncated in place
// (logrotate copytruncate): it shrank below our offset, or it was truncated
// and has already been written past our offset again.
func (t *Follower) checkTruncated() error {
	if t.draining {
		return nil
	}
	fi, err := t.f.Stat()
	if err != nil {
		return err
	}
	if fi.Size() >= t.offset && t.headUnchanged() {
		return nil
	}
	t.log.Warn("queue log truncated, reading from the start", "path", t.path, "size", fi.Size(), "offset", t.offset)
	if _, err := t.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	t.offset, t.buf, t.head = 0, nil, nil
	return nil
}

func (t *Follower) headUnchanged() bool {
	if len(t.head) == 0 {
		return true
	}
	cur := make([]byte, len(t.head))
	n, _ := t.f.ReadAt(cur, 0)
	return bytes.Equal(cur[:n], t.head)
}

// openInitial opens the file to start from, honouring the resume position.
func (t *Follower) openInitial() (bool, error) {
	pos := t.resume
	if pos == nil {
		return t.openAt(t.path, 0, false)
	}

	fi, err := os.Stat(t.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err == nil && inodeOf(fi) == pos.Inode && fi.Size() >= pos.Offset && matchHead(t.path, pos) {
		t.resume = nil
		return t.openAt(t.path, pos.Offset, false)
	}

	// The file we were reading may have been rotated while we were down.
	if old := findRotated(t.path, pos); old != "" {
		t.log.Info("finishing rotated queue log first", "path", old, "offset", pos.Offset)
		t.resume = nil
		if _, err := t.openAt(old, pos.Offset, true); err != nil {
			return false, err
		}
		t.rotatedAt = t.now().Add(-t.grace) // nobody writes to it any more
		return true, nil
	}

	if err != nil {
		return false, nil // current file missing; wait for it
	}
	t.log.Warn("saved position does not match any queue log file, reading the current file from the start",
		"path", t.path, "inode", pos.Inode, "offset", pos.Offset)
	t.resume = nil
	return t.openAt(t.path, 0, false)
}

func (t *Follower) openAt(path string, offset int64, draining bool) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	fi, err := f.Stat()
	if err == nil && offset > 0 {
		_, err = f.Seek(offset, io.SeekStart)
	}
	if err != nil {
		_ = f.Close()
		return false, err
	}
	_ = t.Close()
	t.f, t.inode, t.offset, t.buf, t.head = f, inodeOf(fi), offset, nil, nil
	t.draining = draining
	return true, nil
}

// findRotated looks for a sibling of path (queue_log.1, queue_log-20241021,
// ...) that is the file pos refers to. Compressed files are ignored.
func findRotated(path string, pos *Position) string {
	matches, _ := filepath.Glob(globEscape(path) + "?*")
	for _, m := range matches {
		switch strings.ToLower(filepath.Ext(m)) {
		case ".gz", ".bz2", ".xz", ".zst", ".lz4", ".zip":
			continue
		}
		fi, err := os.Stat(m)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		if inodeOf(fi) == pos.Inode && fi.Size() >= pos.Offset && matchHead(m, pos) {
			return m
		}
	}
	return ""
}

func matchHead(path string, pos *Position) bool {
	if pos.HeadLen == 0 {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck // read-only
	head := make([]byte, pos.HeadLen)
	if _, err := io.ReadFull(f, head); err != nil {
		return false
	}
	return checksum(head) == pos.HeadSum
}

func checksum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func inodeOf(fi os.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Ino) //nolint:unconvert // Ino is not uint64 on every platform
	}
	return 0
}

func globEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`*?[\`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// String is used in log messages.
func (p Position) String() string {
	return fmt.Sprintf("inode=%d offset=%d", p.Inode, p.Offset)
}
