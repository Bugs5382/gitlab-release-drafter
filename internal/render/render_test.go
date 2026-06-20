package render

import (
	"strings"
	"testing"

	"github.com/Bugs5382/gitlab-release-drafter/internal/categorize"
	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func cfg() *config.Config {
	c := &config.Config{
		Template:          "{changes}\n\n## Contributors\n{contributors}",
		ChangeTemplate:    "- {title} (!{iid}) by @{author}",
		NoChangesTemplate: "No notable changes.",
		CategoryTemplate:  "### {title}",
	}
	c.Contributors.Template = "@{username}"
	c.Contributors.Separator = ", "
	return c
}

func TestNotesHappyPath(t *testing.T) {
	cats := []categorize.CategoryResult{
		{Title: "Features", MRs: []model.MergeRequest{
			{IID: 5, Title: "Add login", WebURL: "u5", Author: model.Author{Username: "alice"}},
		}},
		{Title: "Fixes", MRs: []model.MergeRequest{
			{IID: 7, Title: "Fix crash", WebURL: "u7", Author: model.Author{Username: "bob"}},
		}},
	}
	contribs := []model.Author{{Username: "alice"}, {Username: "bob"}}
	out := Notes(cats, contribs, cfg(), Context{Version: "1.2.0"})

	want := []string{
		"### Features",
		"- Add login (!5) by @alice",
		"### Fixes",
		"- Fix crash (!7) by @bob",
		"## Contributors",
		"@alice, @bob",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("output missing %q:\n%s", w, out)
		}
	}
}

func TestNotesNoChanges(t *testing.T) {
	out := Notes(nil, nil, cfg(), Context{})
	if !strings.Contains(out, "No notable changes.") {
		t.Fatalf("want no-changes template, got:\n%s", out)
	}
}

func TestCollectContributorsDedupeAndExclude(t *testing.T) {
	mrs := []model.MergeRequest{
		{Author: model.Author{Username: "alice"}},
		{Author: model.Author{Username: "bot"}},
		{Author: model.Author{Username: "alice"}},
		{Author: model.Author{Username: "carol"}},
	}
	got := CollectContributors(mrs, []string{"bot"})
	if len(got) != 2 || got[0].Username != "alice" || got[1].Username != "carol" {
		t.Fatalf("contributors = %+v", got)
	}
}
