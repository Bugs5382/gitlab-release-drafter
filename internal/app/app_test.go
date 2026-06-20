package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/gitlab"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	raw := `
version:
  increments:
    major: ["breaking"]
    minor: ["feature"]
    patch: ["fix"]
categories:
  - title: "Features"
    labels: ["feature"]
  - title: "Fixes"
    labels: ["fix"]
template: |
  {changes}
changelog:
  section-map:
    "Features": "Added"
    "Fixes": "Fixed"
`
	cfg, err := config.Load(raw)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg
}

func newDeps(t *testing.T, f *gitlab.Fake, env Env) (*Deps, *bytes.Buffer) {
	t.Helper()
	if env.OutDir == "" {
		env.OutDir = t.TempDir()
	}
	var out bytes.Buffer
	return &Deps{
		Client: f,
		Config: testConfig(t),
		Env:    env,
		Stdout: &out,
		Now:    time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
	}, &out
}

func TestPreviewNoMilestone(t *testing.T) {
	f := gitlab.NewFake() // no milestones at all
	d, out := newDeps(t, f, Env{CommitBranch: "feat/x", DefaultBranch: "main"})

	if err := Preview(context.Background(), d); err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !strings.Contains(out.String(), "no milestone") {
		t.Fatalf("want 'no milestone' in output, got: %s", out.String())
	}
	env, _ := os.ReadFile(filepath.Join(d.Env.OutDir, "release.env"))
	if !strings.Contains(string(env), "RELEASE_VERSION=\n") {
		t.Fatalf("want empty RELEASE_VERSION, got: %s", string(env))
	}
}

func TestPreviewHappyPath(t *testing.T) {
	f := gitlab.NewFake()
	f.NearestMilestone = &model.Milestone{IID: 3, Title: "Sprint 3"}
	f.MergedByMS[3] = []model.MergeRequest{
		{IID: 10, Title: "Add SSO", Labels: []string{"feature"}, Author: model.Author{Username: "alice"}},
	}
	f.ExistingTags = []string{"v1.0.0"}
	d, _ := newDeps(t, f, Env{CommitBranch: "main", DefaultBranch: "main"})

	if err := Preview(context.Background(), d); err != nil {
		t.Fatalf("Preview: %v", err)
	}
	env, _ := os.ReadFile(filepath.Join(d.Env.OutDir, "release.env"))
	if !strings.Contains(string(env), "RELEASE_VERSION=1.1.0\n") {
		t.Fatalf("want RELEASE_VERSION=1.1.0, got: %s", string(env))
	}
	if !strings.Contains(string(env), "RELEASE_TAG=v1.1.0\n") {
		t.Fatalf("want RELEASE_TAG=v1.1.0, got: %s", string(env))
	}
	notes, err := os.ReadFile(filepath.Join(d.Env.OutDir, "release-notes.md"))
	if err != nil {
		t.Fatalf("notes: %v", err)
	}
	if !strings.Contains(string(notes), "Add SSO") {
		t.Fatalf("notes missing MR title: %s", string(notes))
	}
}

func TestLabel(t *testing.T) {
	f := gitlab.NewFake()
	f.PipelineMRs = []model.MergeRequest{{IID: 42, SourceBranch: "feat/login"}}
	d := &Deps{Client: f, Stdout: &bytes.Buffer{}, Now: time.Now()}
	d.Config = testConfig(t)
	d.Config.Autolabeler = []config.Autolabel{{Label: "type::feature", Branch: []string{"^feat/"}}}

	if err := Label(context.Background(), d); err != nil {
		t.Fatalf("Label: %v", err)
	}
	got := f.AddedLabels[42]
	if len(got) != 1 || got[0] != "type::feature" {
		t.Fatalf("AddedLabels[42] = %v", got)
	}
}

func TestReleaseScheduledScan(t *testing.T) {
	f := gitlab.NewFake()
	f.ClosedUnreleased = []model.Milestone{{IID: 5, Title: "Sprint 5", Closed: true}}
	f.MergedByMS[5] = []model.MergeRequest{
		{IID: 20, Title: "Fix panic", Labels: []string{"fix"}, Author: model.Author{Username: "bob"}},
	}
	f.ExistingTags = []string{"v1.0.0"}
	d, _ := newDeps(t, f, Env{DefaultBranch: "main", ProjectURL: "https://gl/x"})

	if err := Release(context.Background(), d); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if len(f.UpsertReleases) != 1 {
		t.Fatalf("UpsertReleases = %d, want 1", len(f.UpsertReleases))
	}
	r := f.UpsertReleases[0]
	if r.TagName != "v1.0.1" {
		t.Fatalf("release tag = %q, want v1.0.1", r.TagName)
	}
	if !strings.Contains(r.Description, "Fix panic") {
		t.Fatalf("release body missing MR: %s", r.Description)
	}
	cl, err := os.ReadFile(filepath.Join(d.Env.OutDir, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("changelog: %v", err)
	}
	if !strings.Contains(string(cl), "## [1.0.1] - 2026-06-15") {
		t.Fatalf("changelog missing version section: %s", string(cl))
	}
	if !strings.Contains(string(cl), "### Fixed") {
		t.Fatalf("changelog missing mapped section: %s", string(cl))
	}
}

func TestReleaseAutoTag(t *testing.T) {
	f := gitlab.NewFake()
	f.ClosedUnreleased = []model.Milestone{{IID: 5, Title: "Sprint 5", Closed: true}}
	f.MergedByMS[5] = []model.MergeRequest{
		{IID: 20, Title: "Fix panic", Labels: []string{"fix"}},
	}
	f.ExistingTags = []string{"v1.0.0"}
	d, _ := newDeps(t, f, Env{DefaultBranch: "main", AutoTag: true})

	if err := Release(context.Background(), d); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if ref, ok := f.CreatedTags["v1.0.1"]; !ok || ref != "main" {
		t.Fatalf("CreatedTags = %v, want v1.0.1->main", f.CreatedTags)
	}
}
