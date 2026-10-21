package tail

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newFollower(t *testing.T, path string, resume *Position) (*Follower, *clock) {
	t.Helper()
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	f := New(path, resume, 5*time.Second, nil)
	f.now = c.now
	t.Cleanup(func() { _ = f.Close() })
	return f, c
}

func appendFile(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, f *Follower, max int) ([]string, Position) {
	t.Helper()
	lines, err := f.Read(max)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	var texts []string
	var last Position
	for _, l := range lines {
		texts = append(texts, l.Text)
		last = l.Pos
	}
	return texts, last
}

func expect(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFollowAppendsAndPartialLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue_log")
	f, _ := newFollower(t, path, nil)

	got, _ := read(t, f, 100) // file does not exist yet
	expect(t, got)

	appendFile(t, path, "a\nb\nc")
	got, pos := read(t, f, 100)
	expect(t, got, "a", "b")
	if pos.Offset != 4 {
		t.Fatalf("offset = %d, want 4", pos.Offset)
	}

	appendFile(t, path, "\nd\n")
	got, pos = read(t, f, 1)
	expect(t, got, "c")
	got, pos2 := read(t, f, 100)
	expect(t, got, "d")
	if pos.Offset != 6 || pos2.Offset != 8 {
		t.Fatalf("offsets = %d, %d, want 6, 8", pos.Offset, pos2.Offset)
	}
	got, _ = read(t, f, 100)
	expect(t, got)
}

func TestResume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue_log")
	appendFile(t, path, "a\nb\n")
	f, _ := newFollower(t, path, nil)
	_, pos := read(t, f, 1)

	appendFile(t, path, "c\n")
	f2, _ := newFollower(t, path, &pos)
	got, _ := read(t, f2, 100)
	expect(t, got, "b", "c")
}

func TestRenameRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue_log")
	appendFile(t, path, "a\n")
	f, c := newFollower(t, path, nil)
	got, _ := read(t, f, 100)
	expect(t, got, "a")

	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	got, _ = read(t, f, 100) // rotation noticed, grace period starts
	expect(t, got)

	appendFile(t, path+".1", "late") // writer has not reopened yet
	appendFile(t, path, "new1\n")
	got, _ = read(t, f, 100)
	expect(t, got) // partial line, still in grace period

	c.t = c.t.Add(6 * time.Second)
	got, pos := read(t, f, 100)
	expect(t, got, "late", "new1")
	if pos.Offset != 5 {
		t.Fatalf("offset = %d, want 5", pos.Offset)
	}
}

func TestCopyTruncate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue_log")
	appendFile(t, path, "aaaa\nbbbb\n")
	f, _ := newFollower(t, path, nil)
	got, _ := read(t, f, 100)
	expect(t, got, "aaaa", "bbbb")

	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	appendFile(t, path, "c\n")
	got, _ = read(t, f, 100)
	expect(t, got, "c")

	// Truncated and written past the old offset before we looked.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	appendFile(t, path, "dddd\neeee\n")
	got, _ = read(t, f, 100)
	expect(t, got, "dddd", "eeee")
}

func TestResumeAfterRotationWhileDown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue_log")
	appendFile(t, path, "a\nb\n")
	f, _ := newFollower(t, path, nil)
	_, pos := read(t, f, 1)
	_ = f.Close()

	// While we are down: one more line, rotation with dateext, new file.
	appendFile(t, path, "c\n")
	if err := os.Rename(path, path+"-20241021"); err != nil {
		t.Fatal(err)
	}
	appendFile(t, path+".2.gz", "ignored\n")
	appendFile(t, path, "d\n")

	f2, _ := newFollower(t, path, &pos)
	got, _ := read(t, f2, 100)
	expect(t, got, "b", "c", "d")
}

func TestResumeWithReplacedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue_log")
	appendFile(t, path, "a\nb\n")
	f, _ := newFollower(t, path, nil)
	_, pos := read(t, f, 1)
	_ = f.Close()

	// Same inode, different content: the saved offset is meaningless.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	appendFile(t, path, "x\ny\nz\n")

	f2, _ := newFollower(t, path, &pos)
	got, _ := read(t, f2, 100)
	expect(t, got, "x", "y", "z")
}
