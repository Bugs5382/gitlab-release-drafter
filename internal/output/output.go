package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// KV is an ordered key/value pair for dotenv output.
type KV struct {
	Key   string
	Value string
}

// WriteDotenv writes KEY=value lines suitable for a GitLab dotenv report artifact.
func WriteDotenv(path string, kvs []KV) error {
	var b strings.Builder
	for _, kv := range kvs {
		fmt.Fprintf(&b, "%s=%s\n", kv.Key, kv.Value)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WriteFile writes content to path (e.g. release-notes.md, CHANGELOG.md).
func WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// Summary prints a one-line human summary to w.
func Summary(w io.Writer, command, detail string) {
	fmt.Fprintf(w, "gitlab-release-drafter %s: %s\n", command, detail)
}
