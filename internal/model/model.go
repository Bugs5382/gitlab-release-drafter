package model

import "time"

// Author is the creator of a merge request.
type Author struct {
	Username string
	Name     string
}

// MergeRequest is a merged (or open) MR relevant to a release.
type MergeRequest struct {
	IID          int
	Title        string
	WebURL       string
	Labels       []string
	Author       Author
	MergedAt     time.Time
	SourceBranch string
	ChangedFiles []string
}

// Milestone is the release unit; its merged MRs feed a release.
type Milestone struct {
	IID    int
	Title  string
	DueOn  time.Time
	Closed bool
}

// Release is a GitLab Release to create or upsert.
type Release struct {
	TagName     string
	Name        string
	Description string
	Milestone   string
}
