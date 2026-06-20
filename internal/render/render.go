package render

import (
	"strconv"
	"strings"

	"github.com/Bugs5382/gitlab-release-drafter/internal/categorize"
	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// Context carries body-level placeholder values.
type Context struct {
	Version     string
	PreviousTag string
	Date        string
	Milestone   string
}

// Notes renders the release body from the config templates.
func Notes(cats []categorize.CategoryResult, contributors []model.Author, cfg *config.Config, ctx Context) string {
	changes := renderChanges(cats, cfg, ctx)
	contribs := renderContributors(contributors, cfg)

	r := strings.NewReplacer(
		"{changes}", changes,
		"{contributors}", contribs,
		"{version}", ctx.Version,
		"{previous_tag}", ctx.PreviousTag,
		"{date}", ctx.Date,
		"{milestone}", ctx.Milestone,
	)
	return r.Replace(cfg.Template)
}

func renderChanges(cats []categorize.CategoryResult, cfg *config.Config, ctx Context) string {
	var hasMR bool
	for _, c := range cats {
		if len(c.MRs) > 0 {
			hasMR = true
			break
		}
	}
	if !hasMR {
		return cfg.NoChangesTemplate
	}

	var b strings.Builder
	first := true
	for _, c := range cats {
		if len(c.MRs) == 0 {
			continue
		}
		if !first {
			b.WriteString("\n\n")
		}
		first = false
		heading := strings.ReplaceAll(cfg.CategoryTemplate, "{title}", c.Title)
		b.WriteString(heading)
		for _, mr := range c.MRs {
			b.WriteString("\n")
			b.WriteString(renderChange(cfg.ChangeTemplate, mr, ctx))
		}
	}
	return b.String()
}

func renderChange(tmpl string, mr model.MergeRequest, ctx Context) string {
	r := strings.NewReplacer(
		"{title}", mr.Title,
		"{iid}", strconv.Itoa(mr.IID),
		"{author}", mr.Author.Username,
		"{url}", mr.WebURL,
		"{labels}", strings.Join(mr.Labels, ", "),
		"{milestone}", ctx.Milestone,
	)
	return r.Replace(tmpl)
}

func renderContributors(authors []model.Author, cfg *config.Config) string {
	parts := make([]string, 0, len(authors))
	for _, a := range authors {
		r := strings.NewReplacer("{username}", a.Username, "{name}", a.Name)
		parts = append(parts, r.Replace(cfg.Contributors.Template))
	}
	return strings.Join(parts, cfg.Contributors.Separator)
}

// CollectContributors returns unique MR authors in first-seen order, omitting
// any usernames in exclude.
func CollectContributors(mrs []model.MergeRequest, exclude []string) []model.Author {
	ex := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		ex[e] = struct{}{}
	}
	seen := map[string]struct{}{}
	var out []model.Author
	for _, mr := range mrs {
		u := mr.Author.Username
		if u == "" {
			continue
		}
		if _, skip := ex[u]; skip {
			continue
		}
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, mr.Author)
	}
	return out
}
