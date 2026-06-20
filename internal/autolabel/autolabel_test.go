package autolabel

import (
	"testing"

	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func rules() []config.Autolabel {
	return []config.Autolabel{
		{Label: "type::fix", Branch: []string{"^fix/", "^bugfix/"}, Title: []string{`^fix(\(.+\))?:`}},
		{Label: "type::feature", Branch: []string{"^feat/"}},
		{Label: "documentation", Files: []string{"docs/**", "**/*.md"}},
	}
}

func TestMatchBranch(t *testing.T) {
	mr := model.MergeRequest{SourceBranch: "fix/login"}
	got := Match(mr, rules())
	if len(got) != 1 || got[0] != "type::fix" {
		t.Fatalf("got %v", got)
	}
}

func TestMatchTitleRegex(t *testing.T) {
	mr := model.MergeRequest{SourceBranch: "x", Title: "fix(api): handle nil"}
	got := Match(mr, rules())
	if len(got) != 1 || got[0] != "type::fix" {
		t.Fatalf("got %v", got)
	}
}

func TestMatchFilesGlob(t *testing.T) {
	mr := model.MergeRequest{ChangedFiles: []string{"docs/guide/intro.md"}}
	got := Match(mr, rules())
	if len(got) != 1 || got[0] != "documentation" {
		t.Fatalf("got %v", got)
	}
}

func TestMatchDoublestarTopLevelMd(t *testing.T) {
	mr := model.MergeRequest{ChangedFiles: []string{"README.md"}}
	got := Match(mr, rules())
	if len(got) != 1 || got[0] != "documentation" {
		t.Fatalf("**/*.md should match top-level README.md, got %v", got)
	}
}

func TestMatchDedupAndSkipExisting(t *testing.T) {
	mr := model.MergeRequest{
		SourceBranch: "feat/x",
		ChangedFiles: []string{"docs/a.md"},
		Labels:       []string{"documentation"}, // already present, must be skipped
	}
	got := Match(mr, rules())
	if len(got) != 1 || got[0] != "type::feature" {
		t.Fatalf("got %v", got)
	}
}
