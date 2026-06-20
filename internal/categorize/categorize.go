package categorize

import (
	"sort"

	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// CategoryResult is a category with the MRs that landed in it (render order).
type CategoryResult struct {
	Title string
	MRs   []model.MergeRequest
}

// Categorize assigns MRs to the configured categories. Each MR lands in the
// first category it matches (config order). Excluded MRs are dropped entirely.
// Unmatched MRs go to an uncategorized bucket when uncategorized-title is set,
// otherwise they are dropped.
func Categorize(mrs []model.MergeRequest, cfg *config.Config) []CategoryResult {
	excluded := toSet(cfg.ExcludeLabels)

	results := make([]CategoryResult, len(cfg.Categories))
	for i, c := range cfg.Categories {
		results[i].Title = c.Title
	}
	var uncategorized []model.MergeRequest

	for _, mr := range mrs {
		if hasAny(mr.Labels, excluded) {
			continue
		}
		idx := -1
		for i, c := range cfg.Categories {
			if matches(mr.Labels, c.Labels, cfg.LabelsMode) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			results[idx].MRs = append(results[idx].MRs, mr)
		} else if cfg.UncategorizedTitle != "" {
			uncategorized = append(uncategorized, mr)
		}
	}

	// Drop empty configured categories.
	out := make([]CategoryResult, 0, len(results)+1)
	for _, r := range results {
		if len(r.MRs) > 0 {
			sortMRs(r.MRs, cfg.Sort, cfg.SortDirection)
			out = append(out, r)
		}
	}
	if cfg.UncategorizedTitle != "" && len(uncategorized) > 0 {
		sortMRs(uncategorized, cfg.Sort, cfg.SortDirection)
		out = append(out, CategoryResult{Title: cfg.UncategorizedTitle, MRs: uncategorized})
	}
	return out
}

func matches(mrLabels, catLabels []string, mode string) bool {
	if len(catLabels) == 0 {
		return false
	}
	set := toSet(mrLabels)
	if mode == "all" {
		for _, l := range catLabels {
			if _, ok := set[l]; !ok {
				return false
			}
		}
		return true
	}
	// any
	for _, l := range catLabels {
		if _, ok := set[l]; ok {
			return true
		}
	}
	return false
}

func sortMRs(mrs []model.MergeRequest, by, dir string) {
	less := func(a, b model.MergeRequest) bool {
		if by == "title" {
			return a.Title < b.Title
		}
		return a.MergedAt.Before(b.MergedAt)
	}
	sort.SliceStable(mrs, func(i, j int) bool {
		if dir == "desc" {
			return less(mrs[j], mrs[i])
		}
		return less(mrs[i], mrs[j])
	})
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func hasAny(labels []string, set map[string]struct{}) bool {
	for _, l := range labels {
		if _, ok := set[l]; ok {
			return true
		}
	}
	return false
}
