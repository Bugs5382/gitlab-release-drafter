package app

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Bugs5382/gitlab-release-drafter/internal/categorize"
	"github.com/Bugs5382/gitlab-release-drafter/internal/changelog"
	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/gitlab"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
	"github.com/Bugs5382/gitlab-release-drafter/internal/output"
	"github.com/Bugs5382/gitlab-release-drafter/internal/render"
	"github.com/Bugs5382/gitlab-release-drafter/internal/version"
)

// Env carries CI/runtime values that influence command behavior.
type Env struct {
	Milestone     string // GLRD_MILESTONE override (title or IID)
	CommitBranch  string // CI_COMMIT_BRANCH
	DefaultBranch string // CI_DEFAULT_BRANCH
	CommitTag     string // CI_COMMIT_TAG
	ProjectURL    string // CI_PROJECT_URL (for changelog compare links)
	AutoTag       bool   // create the resolved tag during release
	OutDir        string // directory for artifacts and CHANGELOG
}

// Deps bundles everything the commands need.
type Deps struct {
	Client gitlab.Client
	Config *config.Config
	Env    Env
	Stdout io.Writer
	Now    time.Time
}

func (d *Deps) date() string { return d.Now.Format("2006-01-02") }

func (d *Deps) ref() string {
	if d.Env.CommitBranch != "" {
		return d.Env.CommitBranch
	}
	if d.Env.DefaultBranch != "" {
		return d.Env.DefaultBranch
	}
	return "HEAD"
}

func (d *Deps) outPath(name string) string {
	return filepath.Join(d.Env.OutDir, name)
}

// resolved holds the version computation for a milestone.
type resolved struct {
	base    version.Semver
	next    version.Semver
	inc     version.Increment
	tag     string
	prevTag string
	name    string
}

func (d *Deps) resolveVersion(ctx context.Context, mrs []model.MergeRequest) (resolved, error) {
	tags, err := d.Client.Tags(ctx)
	if err != nil {
		return resolved{}, err
	}
	cfg := d.Config
	base := version.Base(tags, cfg.Version.TagTemplate, cfg.Version.Initial)
	inc := version.Resolve(mrs, cfg.Version.Increments, cfg.Version.DefaultIncrement)
	next := version.Bump(base, inc)
	return resolved{
		base:    base,
		next:    next,
		inc:     inc,
		tag:     version.RenderTag(cfg.Version.TagTemplate, next),
		prevTag: version.RenderTag(cfg.Version.TagTemplate, base),
		name:    version.RenderTag(cfg.Version.NameTemplate, next),
	}, nil
}

func (d *Deps) renderNotes(mrs []model.MergeRequest, ms *model.Milestone, r resolved) (string, []categorize.CategoryResult) {
	cfg := d.Config
	cats := categorize.Categorize(mrs, cfg)
	var included []model.MergeRequest
	for _, c := range cats {
		included = append(included, c.MRs...)
	}
	contribs := render.CollectContributors(included, cfg.Contributors.Exclude)
	notes := render.Notes(cats, contribs, cfg, render.Context{
		Version:     r.next.String(),
		PreviousTag: r.prevTag,
		Date:        d.date(),
		Milestone:   ms.Title,
	})
	return notes, cats
}

func (d *Deps) writePreviewArtifacts(notes string, ms *model.Milestone, r resolved) error {
	if err := output.WriteDotenv(d.outPath("release.env"), []output.KV{
		{Key: "RELEASE_VERSION", Value: r.next.String()},
		{Key: "RELEASE_TAG", Value: r.tag},
		{Key: "RELEASE_MILESTONE", Value: ms.Title},
		{Key: "RELEASE_INCREMENT", Value: r.inc.String()},
	}); err != nil {
		return err
	}
	return output.WriteFile(d.outPath("release-notes.md"), notes)
}

// Preview resolves the version and renders notes for the relevant milestone.
func Preview(ctx context.Context, d *Deps) error {
	ms, err := d.selectMilestone(ctx, true)
	if err != nil {
		return err
	}
	if ms == nil {
		output.Summary(d.Stdout, "preview", "no milestone for this ref; nothing to preview")
		return output.WriteDotenv(d.outPath("release.env"), []output.KV{
			{Key: "RELEASE_VERSION", Value: ""},
			{Key: "RELEASE_TAG", Value: ""},
			{Key: "RELEASE_MILESTONE", Value: ""},
			{Key: "RELEASE_INCREMENT", Value: ""},
		})
	}

	mrs, err := d.Client.MergedMRs(ctx, *ms)
	if err != nil {
		return err
	}
	r, err := d.resolveVersion(ctx, mrs)
	if err != nil {
		return err
	}
	notes, _ := d.renderNotes(mrs, ms, r)
	if err := d.writePreviewArtifacts(notes, ms, r); err != nil {
		return err
	}
	output.Summary(d.Stdout, "preview", fmt.Sprintf("%s would cut %s (%s)", ms.Title, r.tag, r.inc))
	return nil
}

// Label applies autolabeler rules to the current pipeline's MR(s).
func Label(ctx context.Context, d *Deps) error {
	mrs, err := d.Client.MRsForPipeline(ctx)
	if err != nil {
		return err
	}
	count := 0
	for _, mr := range mrs {
		labels := autolabelMatch(mr, d.Config.Autolabeler)
		if len(labels) == 0 {
			continue
		}
		if err := d.Client.AddMRLabels(ctx, mr.IID, labels); err != nil {
			return err
		}
		count += len(labels)
	}
	output.Summary(d.Stdout, "label", fmt.Sprintf("applied %d label(s) across %d MR(s)", count, len(mrs)))
	return nil
}

// Release resolves and publishes Releases for the relevant milestone(s).
func Release(ctx context.Context, d *Deps) error {
	milestones, err := d.releaseTargets(ctx)
	if err != nil {
		return err
	}
	if len(milestones) == 0 {
		output.Summary(d.Stdout, "release", "no milestone to release; nothing to do")
		return nil
	}
	for i := range milestones {
		if err := d.releaseOne(ctx, &milestones[i]); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deps) releaseOne(ctx context.Context, ms *model.Milestone) error {
	mrs, err := d.Client.MergedMRs(ctx, *ms)
	if err != nil {
		return err
	}
	r, err := d.resolveVersion(ctx, mrs)
	if err != nil {
		return err
	}
	notes, cats := d.renderNotes(mrs, ms, r)

	if d.Env.AutoTag {
		if err := d.Client.CreateTag(ctx, r.tag, d.ref()); err != nil {
			return err
		}
	}
	if err := d.Client.UpsertRelease(ctx, model.Release{
		TagName:     r.tag,
		Name:        r.name,
		Description: notes,
		Milestone:   ms.Title,
	}); err != nil {
		return err
	}
	if err := d.updateChangelog(cats, r); err != nil {
		return err
	}
	if err := d.writePreviewArtifacts(notes, ms, r); err != nil {
		return err
	}
	output.Summary(d.Stdout, "release", fmt.Sprintf("%s released as %s", ms.Title, r.tag))
	return nil
}

// Changelog updates CHANGELOG.md for the relevant milestone without cutting a release.
func Changelog(ctx context.Context, d *Deps) error {
	ms, err := d.selectMilestone(ctx, false)
	if err != nil {
		return err
	}
	if ms == nil {
		output.Summary(d.Stdout, "changelog", "no milestone; nothing to do")
		return nil
	}
	mrs, err := d.Client.MergedMRs(ctx, *ms)
	if err != nil {
		return err
	}
	r, err := d.resolveVersion(ctx, mrs)
	if err != nil {
		return err
	}
	_, cats := d.renderNotes(mrs, ms, r)
	if err := d.updateChangelog(cats, r); err != nil {
		return err
	}
	output.Summary(d.Stdout, "changelog", fmt.Sprintf("updated %s for %s", d.Config.Changelog.File, r.tag))
	return nil
}

func (d *Deps) updateChangelog(cats []categorize.CategoryResult, r resolved) error {
	path := d.outPath(d.Config.Changelog.File)
	existing := readFileOrEmpty(path)
	sections := d.buildSections(cats)

	params := changelog.PromoteParams{
		Version:     r.next.String(),
		Date:        d.date(),
		TagName:     r.tag,
		PreviousTag: r.prevTag,
		Sections:    sections,
	}
	if d.Env.ProjectURL != "" {
		params.CompareURL = d.Env.ProjectURL + "/-/compare/{from}...{to}"
	}
	out, err := changelog.Promote(existing, params)
	if err != nil {
		return err
	}
	return output.WriteFile(path, out)
}

// buildSections maps categorized MRs to ordered Keep a Changelog sections,
// merging categories that map to the same section title.
func (d *Deps) buildSections(cats []categorize.CategoryResult) []changelog.Section {
	cfg := d.Config
	var order []string
	byTitle := map[string]*changelog.Section{}
	for _, c := range cats {
		title := c.Title
		if mapped, ok := cfg.Changelog.SectionMap[c.Title]; ok && mapped != "" {
			title = mapped
		}
		sec, ok := byTitle[title]
		if !ok {
			sec = &changelog.Section{Title: title}
			byTitle[title] = sec
			order = append(order, title)
		}
		for _, mr := range c.MRs {
			sec.Items = append(sec.Items, changelogItem(cfg.ChangeTemplate, mr))
		}
	}
	out := make([]changelog.Section, 0, len(order))
	for _, t := range order {
		out = append(out, *byTitle[t])
	}
	return out
}

func (d *Deps) selectMilestone(ctx context.Context, preview bool) (*model.Milestone, error) {
	if d.Env.Milestone != "" {
		return d.Client.MilestoneByRef(ctx, d.Env.Milestone)
	}
	if preview && d.Env.CommitBranch != "" && d.Env.CommitBranch != d.Env.DefaultBranch {
		return d.Client.OpenMRMilestone(ctx, d.Env.CommitBranch)
	}
	return d.Client.NearestOpenMilestone(ctx)
}

func (d *Deps) releaseTargets(ctx context.Context) ([]model.Milestone, error) {
	if d.Env.Milestone != "" {
		ms, err := d.Client.MilestoneByRef(ctx, d.Env.Milestone)
		if err != nil || ms == nil {
			return nil, err
		}
		return []model.Milestone{*ms}, nil
	}
	closed, err := d.Client.ClosedUnreleasedMilestones(ctx, d.Config.Version.TagTemplate)
	if err != nil {
		return nil, err
	}
	if len(closed) > 0 {
		return closed, nil
	}
	ms, err := d.Client.NearestOpenMilestone(ctx)
	if err != nil || ms == nil {
		return nil, err
	}
	return []model.Milestone{*ms}, nil
}

func readFileOrEmpty(path string) string {
	b, err := osReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
