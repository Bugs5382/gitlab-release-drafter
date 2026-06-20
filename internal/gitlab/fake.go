package gitlab

import (
	"context"

	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// Fake is an in-memory Client for tests.
type Fake struct {
	NearestMilestone *model.Milestone
	ByRef            map[string]*model.Milestone
	BranchMilestone  map[string]*model.Milestone
	ClosedUnreleased []model.Milestone
	MergedByMS       map[int][]model.MergeRequest
	PipelineMRs      []model.MergeRequest
	ExistingTags     []string

	// Recorded writes.
	AddedLabels    map[int][]string
	CreatedTags    map[string]string
	UpsertReleases []model.Release
}

// NewFake returns a Fake with initialized maps.
func NewFake() *Fake {
	return &Fake{
		ByRef:           map[string]*model.Milestone{},
		BranchMilestone: map[string]*model.Milestone{},
		MergedByMS:      map[int][]model.MergeRequest{},
		AddedLabels:     map[int][]string{},
		CreatedTags:     map[string]string{},
	}
}

func (f *Fake) NearestOpenMilestone(ctx context.Context) (*model.Milestone, error) {
	return f.NearestMilestone, nil
}

func (f *Fake) MilestoneByRef(ctx context.Context, ref string) (*model.Milestone, error) {
	return f.ByRef[ref], nil
}

func (f *Fake) OpenMRMilestone(ctx context.Context, branch string) (*model.Milestone, error) {
	return f.BranchMilestone[branch], nil
}

func (f *Fake) ClosedUnreleasedMilestones(ctx context.Context, tagTemplate string) ([]model.Milestone, error) {
	return f.ClosedUnreleased, nil
}

func (f *Fake) MergedMRs(ctx context.Context, milestone model.Milestone) ([]model.MergeRequest, error) {
	return f.MergedByMS[milestone.IID], nil
}

func (f *Fake) MRsForPipeline(ctx context.Context) ([]model.MergeRequest, error) {
	return f.PipelineMRs, nil
}

func (f *Fake) Tags(ctx context.Context) ([]string, error) {
	return f.ExistingTags, nil
}

func (f *Fake) AddMRLabels(ctx context.Context, mrIID int, labels []string) error {
	f.AddedLabels[mrIID] = append(f.AddedLabels[mrIID], labels...)
	return nil
}

func (f *Fake) CreateTag(ctx context.Context, tag, ref string) error {
	f.CreatedTags[tag] = ref
	return nil
}

func (f *Fake) UpsertRelease(ctx context.Context, r model.Release) error {
	f.UpsertReleases = append(f.UpsertReleases, r)
	return nil
}
