package native

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFile(t *testing.T) {
	f, err := os.CreateTemp("", "native_fs_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello")
	f.Close()

	data, err := ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", data, "hello")
	}
}

func TestReadFile_NotFound(t *testing.T) {
	_, err := ReadFile("/tmp/definitely_does_not_exist_xyz_123.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "write_test.txt")
	if err := WriteFile(path, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", data, "world")
	}
}

func TestReadDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bb"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	entries, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	names := map[string]DirEntry{}
	for _, e := range entries {
		names[e.Name] = e
	}

	if e, ok := names["a.txt"]; !ok || e.IsDir || e.Size != 1 {
		t.Errorf("a.txt entry unexpected: %+v", e)
	}
	if e, ok := names["b.txt"]; !ok || e.IsDir || e.Size != 2 {
		t.Errorf("b.txt entry unexpected: %+v", e)
	}
	if e, ok := names["subdir"]; !ok || !e.IsDir {
		t.Errorf("subdir entry unexpected: %+v", e)
	}
}

func TestReadDir_NotFound(t *testing.T) {
	_, err := ReadDir("/tmp/definitely_no_such_dir_xyz_123")
	if err == nil {
		t.Error("expected error for missing directory")
	}
}

func TestWatchFile(t *testing.T) {
	f, err := os.CreateTemp("", "native_watch_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("initial")
	f.Close()

	changed := make(chan []byte, 1)
	stop, err := WatchFile(f.Name(), func(content []byte) {
		select {
		case changed <- content:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	// Write new content to trigger watcher
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(f.Name(), []byte("updated"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case content := <-changed:
		if string(content) != "updated" {
			t.Errorf("got %q, want %q", content, "updated")
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for file watch event")
	}
}
