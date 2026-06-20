package config

import (
	"encoding/base64"
	"testing"
)

func TestLoadBase64AndDefaults(t *testing.T) {
	raw := "categories:\n  - title: Feat\n    labels: [feature]\n"
	enc := base64.StdEncoding.EncodeToString([]byte(raw))
	cfg, err := Load(enc)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version.Initial != "0.1.0" {
		t.Fatalf("default initial = %q, want 0.1.0", cfg.Version.Initial)
	}
	if cfg.Version.DefaultIncrement != "patch" {
		t.Fatalf("default increment = %q, want patch", cfg.Version.DefaultIncrement)
	}
	if cfg.LabelsMode != "any" {
		t.Fatalf("default labels-mode = %q, want any", cfg.LabelsMode)
	}
	if cfg.Sort != "merged_at" || cfg.SortDirection != "desc" {
		t.Fatalf("default sort = %q/%q", cfg.Sort, cfg.SortDirection)
	}
	if cfg.Changelog.File != "CHANGELOG.md" {
		t.Fatalf("default changelog file = %q", cfg.Changelog.File)
	}
	if len(cfg.Categories) != 1 || cfg.Categories[0].Title != "Feat" {
		t.Fatalf("categories = %+v", cfg.Categories)
	}
}

func TestLoadRawYAML(t *testing.T) {
	raw := "version:\n  initial: 1.0.0\ncategories:\n  - title: X\n    labels: [a]\n"
	cfg, err := Load(raw)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version.Initial != "1.0.0" {
		t.Fatalf("initial = %q", cfg.Version.Initial)
	}
}

func TestLoadRejectsEmpty(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("want error on empty config")
	}
	if _, err := Load("   \n\t"); err == nil {
		t.Fatal("want error on whitespace-only config")
	}
}

func TestValidateBadLabelsMode(t *testing.T) {
	raw := "labels-mode: sometimes\ncategories:\n  - title: X\n    labels: [a]\n"
	if _, err := Load(raw); err == nil {
		t.Fatal("want validation error for bad labels-mode")
	}
}

func TestValidateRequiresCategoriesOrUncategorized(t *testing.T) {
	if _, err := Load("version:\n  initial: 1.0.0\n"); err == nil {
		t.Fatal("want validation error when no categories and no uncategorized-title")
	}
	// uncategorized-title alone is enough
	if _, err := Load("uncategorized-title: Other\n"); err != nil {
		t.Fatalf("uncategorized-title should satisfy validation: %v", err)
	}
}
