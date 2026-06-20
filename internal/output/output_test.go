package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDotenv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.env")
	err := WriteDotenv(path, []KV{
		{"RELEASE_VERSION", "1.2.0"},
		{"RELEASE_TAG", "v1.2.0"},
		{"RELEASE_MILESTONE", "Sprint 4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	for _, want := range []string{
		"RELEASE_VERSION=1.2.0\n",
		"RELEASE_TAG=v1.2.0\n",
		"RELEASE_MILESTONE=Sprint 4\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dotenv missing %q:\n%s", want, got)
		}
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := WriteFile(path, "hello"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello" {
		t.Fatalf("got %q", string(data))
	}
}
