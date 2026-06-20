package changelog

import (
	"strings"
	"testing"
)

func TestRenderSections(t *testing.T) {
	got := RenderSections([]Section{
		{Title: "Added", Items: []string{"login", "logout"}},
		{Title: "Fixed", Items: []string{"crash"}},
	})
	want := "### Added\n\n- login\n- logout\n\n### Fixed\n\n- crash"
	if got != want {
		t.Fatalf("RenderSections =\n%q\nwant\n%q", got, want)
	}
}

func TestPromoteEmptyCreatesSkeletonAndFirstRelease(t *testing.T) {
	out, err := Promote("", PromoteParams{
		Version:  "0.1.0",
		Date:     "2026-06-15",
		TagName:  "v0.1.0",
		Sections: []Section{{Title: "Added", Items: []string{"initial"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Changelog") {
		t.Fatalf("missing title:\n%s", out)
	}
	if !strings.Contains(out, "## [Unreleased]") {
		t.Fatalf("missing Unreleased:\n%s", out)
	}
	if !strings.Contains(out, "## [0.1.0] - 2026-06-15") {
		t.Fatalf("missing version section:\n%s", out)
	}
	if !strings.Contains(out, "- initial") {
		t.Fatalf("missing item:\n%s", out)
	}
}

func TestPromoteInsertsAbovePriorAndResetsUnreleased(t *testing.T) {
	existing := `# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- something staged

## [1.0.0] - 2026-01-01

### Added

- first

[Unreleased]: https://gl/x/-/compare/v1.0.0...HEAD
[1.0.0]: https://gl/x/-/compare/v0.9.0...v1.0.0
`
	out, err := Promote(existing, PromoteParams{
		Version:     "1.1.0",
		Date:        "2026-06-15",
		TagName:     "v1.1.0",
		PreviousTag: "v1.0.0",
		CompareURL:  "https://gl/x/-/compare/{from}...{to}",
		Sections:    []Section{{Title: "Fixed", Items: []string{"a bug"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	idxNew := strings.Index(out, "## [1.1.0] - 2026-06-15")
	idxOld := strings.Index(out, "## [1.0.0] - 2026-01-01")
	idxUnrel := strings.Index(out, "## [Unreleased]")
	if idxNew < 0 || idxOld < 0 || idxUnrel < 0 {
		t.Fatalf("missing a section:\n%s", out)
	}
	if !(idxUnrel < idxNew && idxNew < idxOld) {
		t.Fatalf("ordering wrong (unrel=%d new=%d old=%d):\n%s", idxUnrel, idxNew, idxOld, out)
	}
	// Unreleased reset: the staged content must no longer appear.
	if strings.Contains(out, "something staged") {
		t.Fatalf("Unreleased not reset:\n%s", out)
	}
	// Link block updated for the new version + Unreleased now compares from new tag.
	if !strings.Contains(out, "[1.1.0]: https://gl/x/-/compare/v1.0.0...v1.1.0") {
		t.Fatalf("missing new version link:\n%s", out)
	}
	if !strings.Contains(out, "[Unreleased]: https://gl/x/-/compare/v1.1.0...HEAD") {
		t.Fatalf("Unreleased link not updated:\n%s", out)
	}
	// Old link preserved.
	if !strings.Contains(out, "[1.0.0]: https://gl/x/-/compare/v0.9.0...v1.0.0") {
		t.Fatalf("old link lost:\n%s", out)
	}
}

func TestRenderUnreleased(t *testing.T) {
	got := RenderUnreleased([]Section{{Title: "Added", Items: []string{"x"}}})
	if !strings.Contains(got, "### Added") || !strings.Contains(got, "- x") {
		t.Fatalf("RenderUnreleased = %q", got)
	}
}
