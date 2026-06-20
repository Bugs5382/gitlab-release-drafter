package categorize

import (
	"testing"
	"time"

	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func baseCfg() *config.Config {
	c := &config.Config{
		Categories: []config.Category{
			{Title: "Features", Labels: []string{"feature", "enhancement"}},
			{Title: "Fixes", Labels: []string{"fix"}},
		},
		LabelsMode:    "any",
		ExcludeLabels: []string{"skip-changelog"},
		Sort:          "merged_at",
		SortDirection: "desc",
	}
	return c
}

func TestAnyModeFirstMatchWins(t *testing.T) {
	cfg := baseCfg()
	mrs := []model.MergeRequest{
		{IID: 1, Labels: []string{"feature"}},
		{IID: 2, Labels: []string{"fix"}},
		{IID: 3, Labels: []string{"feature", "fix"}}, // matches Features first
	}
	got := Categorize(mrs, cfg)
	if len(got) != 2 {
		t.Fatalf("categories = %d, want 2", len(got))
	}
	if got[0].Title != "Features" || len(got[0].MRs) != 2 {
		t.Fatalf("Features = %+v", got[0])
	}
	if got[1].Title != "Fixes" || len(got[1].MRs) != 1 {
		t.Fatalf("Fixes = %+v", got[1])
	}
}

func TestAllMode(t *testing.T) {
	cfg := baseCfg()
	cfg.LabelsMode = "all"
	cfg.Categories = []config.Category{{Title: "Both", Labels: []string{"a", "b"}}}
	mrs := []model.MergeRequest{
		{IID: 1, Labels: []string{"a"}},      // not all
		{IID: 2, Labels: []string{"a", "b"}}, // all
	}
	got := Categorize(mrs, cfg)
	if len(got) != 1 || len(got[0].MRs) != 1 || got[0].MRs[0].IID != 2 {
		t.Fatalf("all-mode = %+v", got)
	}
}

func TestExcludeLabels(t *testing.T) {
	cfg := baseCfg()
	mrs := []model.MergeRequest{
		{IID: 1, Labels: []string{"feature", "skip-changelog"}},
		{IID: 2, Labels: []string{"feature"}},
	}
	got := Categorize(mrs, cfg)
	if len(got) != 1 || len(got[0].MRs) != 1 || got[0].MRs[0].IID != 2 {
		t.Fatalf("exclude = %+v", got)
	}
}

func TestUncategorizedDropVsKeep(t *testing.T) {
	cfg := baseCfg()
	mrs := []model.MergeRequest{
		{IID: 1, Labels: []string{"feature"}},
		{IID: 2, Labels: []string{"random"}},
	}
	// default: drop
	got := Categorize(mrs, cfg)
	if len(got) != 1 {
		t.Fatalf("drop: categories = %d", len(got))
	}
	// keep
	cfg.UncategorizedTitle = "Other"
	got = Categorize(mrs, cfg)
	if len(got) != 2 || got[1].Title != "Other" || got[1].MRs[0].IID != 2 {
		t.Fatalf("keep: %+v", got)
	}
}

func TestSortWithinCategory(t *testing.T) {
	cfg := baseCfg()
	cfg.Categories = []config.Category{{Title: "F", Labels: []string{"feature"}}}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mrs := []model.MergeRequest{
		{IID: 1, Labels: []string{"feature"}, MergedAt: t0},
		{IID: 2, Labels: []string{"feature"}, MergedAt: t0.Add(time.Hour)},
	}
	got := Categorize(mrs, cfg)
	// desc by merged_at -> IID 2 first
	if got[0].MRs[0].IID != 2 {
		t.Fatalf("sort desc failed: %+v", got[0].MRs)
	}
}
