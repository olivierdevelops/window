package orchfeatures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeNativeFS_ReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	nfs := MakeNativeFS()
	got, err := nfs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("ReadFile = %q, want %q", got, "hello world")
	}
}

func TestMakeNativeFS_WriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	nfs := MakeNativeFS()
	if err := nfs.WriteFile(path, []byte("written"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "written" {
		t.Errorf("file content = %q, want %q", got, "written")
	}
}

func TestMakeNativeFS_ReadDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	nfs := MakeNativeFS()
	entries, err := nfs.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadDir returned %d entries, want 3", len(entries))
	}

	dirCount, fileCount := 0, 0
	for _, e := range entries {
		if e.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}
	if dirCount != 1 {
		t.Errorf("dir count = %d, want 1", dirCount)
	}
	if fileCount != 2 {
		t.Errorf("file count = %d, want 2", fileCount)
	}
}

func TestMakeNativeFS_DirEntryConversion(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello from file")
	path := filepath.Join(dir, "entry.txt")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "mydir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	nfs := MakeNativeFS()
	entries, err := nfs.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	byName := make(map[string]interface{})
	for _, e := range entries {
		byName[e.Name] = e
	}

	type entry struct {
		Name  string
		IsDir bool
		Size  int64
	}

	if _, ok := byName["entry.txt"]; !ok {
		t.Fatal("entry.txt not found in ReadDir results")
	}
	if _, ok := byName["mydir"]; !ok {
		t.Fatal("mydir not found in ReadDir results")
	}

	for _, e := range entries {
		switch e.Name {
		case "entry.txt":
			if e.IsDir {
				t.Error("entry.txt should not be a dir")
			}
			if e.Size != int64(len(content)) {
				t.Errorf("entry.txt size = %d, want %d", e.Size, len(content))
			}
		case "mydir":
			if !e.IsDir {
				t.Error("mydir should be a dir")
			}
		}
	}
}
