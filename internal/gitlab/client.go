package gitlab

import (
	"context"

	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// Client abstracts the GitLab operations the tool needs. The HTTP
// implementation lives in http.go; tests use the in-memory Fake.
type Client interface {
	// NearestOpenMilestone returns the project's nearest-due open milestone,
	// or (nil, nil) if there is none.
	NearestOpenMilestone(ctx context.Context) (*model.Milestone, error)
	// MilestoneByRef resolves a milestone by title or IID.
	MilestoneByRef(ctx context.Context, ref string) (*model.Milestone, error)
	// OpenMRMilestone returns the milestone of the open MR for branch, or nil.
	OpenMRMilestone(ctx context.Context, branch string) (*model.Milestone, error)
	// ClosedUnreleasedMilestones returns closed milestones that have no Release
	// for their resolved tag yet.
	ClosedUnreleasedMilestones(ctx context.Context, tagTemplate string) ([]model.Milestone, error)
	// MergedMRs returns the merged MRs assigned to a milestone.
	MergedMRs(ctx context.Context, milestone model.Milestone) ([]model.MergeRequest, error)
	// MRsForPipeline returns the MR(s) associated with the current pipeline.
	MRsForPipeline(ctx context.Context) ([]model.MergeRequest, error)
	// Tags returns existing tag names.
	Tags(ctx context.Context) ([]string, error)
	// AddMRLabels adds labels to an MR.
	AddMRLabels(ctx context.Context, mrIID int, labels []string) error
	// CreateTag creates a tag at ref.
	CreateTag(ctx context.Context, tag, ref string) error
	// UpsertRelease creates or updates a GitLab Release.
	UpsertRelease(ctx context.Context, r model.Release) error
}
