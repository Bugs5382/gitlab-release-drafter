package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// HTTPClient talks to the GitLab REST API v4.
type HTTPClient struct {
	apiBase   string // e.g. https://gitlab.example.com/api/v4
	projectID string
	token     string
	mrIID     int // CI_MERGE_REQUEST_IID (0 if not in an MR pipeline)
	http      *http.Client
}

var _ Client = (*HTTPClient)(nil)

// Options configures an HTTPClient from CI environment values.
type Options struct {
	APIBaseURL string // CI_API_V4_URL
	ProjectID  string // CI_PROJECT_ID
	Token      string // GLRD_TOKEN
	MRIID      int    // CI_MERGE_REQUEST_IID
}

// NewHTTPClient constructs a GitLab REST client.
func NewHTTPClient(o Options) *HTTPClient {
	return &HTTPClient{
		apiBase:   strings.TrimRight(o.APIBaseURL, "/"),
		projectID: o.ProjectID,
		token:     o.Token,
		mrIID:     o.MRIID,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPClient) projectPath(suffix string) string {
	return fmt.Sprintf("%s/projects/%s%s", c.apiBase, url.PathEscape(c.projectID), suffix)
}

func (c *HTTPClient) do(ctx context.Context, method, rawURL string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// APIError is a non-2xx GitLab response.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gitlab api: status %d: %s", e.Status, e.Body)
}

// --- wire types ---

type wireMilestone struct {
	IID     int    `json:"iid"`
	Title   string `json:"title"`
	DueDate string `json:"due_date"`
	State   string `json:"state"`
}

func (m wireMilestone) toModel() model.Milestone {
	due, _ := time.Parse("2006-01-02", m.DueDate)
	return model.Milestone{IID: m.IID, Title: m.Title, DueOn: due, Closed: m.State == "closed"}
}

type wireMR struct {
	IID          int      `json:"iid"`
	Title        string   `json:"title"`
	WebURL       string   `json:"web_url"`
	Labels       []string `json:"labels"`
	SourceBranch string   `json:"source_branch"`
	MergedAt     string   `json:"merged_at"`
	Author       struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"author"`
	Milestone *wireMilestone `json:"milestone"`
}

func (w wireMR) toModel() model.MergeRequest {
	merged, _ := time.Parse(time.RFC3339, w.MergedAt)
	return model.MergeRequest{
		IID:          w.IID,
		Title:        w.Title,
		WebURL:       w.WebURL,
		Labels:       w.Labels,
		SourceBranch: w.SourceBranch,
		MergedAt:     merged,
		Author:       model.Author{Username: w.Author.Username, Name: w.Author.Name},
	}
}

// --- Client implementation ---

func (c *HTTPClient) milestones(ctx context.Context, state string) ([]wireMilestone, error) {
	q := url.Values{"per_page": {"100"}}
	if state != "" {
		q.Set("state", state)
	}
	var out []wireMilestone
	if err := c.do(ctx, http.MethodGet, c.projectPath("/milestones?"+q.Encode()), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) NearestOpenMilestone(ctx context.Context) (*model.Milestone, error) {
	ms, err := c.milestones(ctx, "active")
	if err != nil {
		return nil, err
	}
	var best *wireMilestone
	for i := range ms {
		if ms[i].DueDate == "" {
			continue
		}
		if best == nil || ms[i].DueDate < best.DueDate {
			best = &ms[i]
		}
	}
	if best == nil {
		if len(ms) == 0 {
			return nil, nil
		}
		best = &ms[0] // no due dates set; fall back to first active
	}
	m := best.toModel()
	return &m, nil
}

func (c *HTTPClient) MilestoneByRef(ctx context.Context, ref string) (*model.Milestone, error) {
	ms, err := c.milestones(ctx, "")
	if err != nil {
		return nil, err
	}
	iid, numErr := strconv.Atoi(ref)
	for i := range ms {
		if ms[i].Title == ref || (numErr == nil && ms[i].IID == iid) {
			m := ms[i].toModel()
			return &m, nil
		}
	}
	return nil, nil
}

func (c *HTTPClient) OpenMRMilestone(ctx context.Context, branch string) (*model.Milestone, error) {
	q := url.Values{"source_branch": {branch}, "state": {"opened"}, "per_page": {"1"}}
	var out []wireMR
	if err := c.do(ctx, http.MethodGet, c.projectPath("/merge_requests?"+q.Encode()), nil, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 || out[0].Milestone == nil {
		return nil, nil
	}
	m := out[0].Milestone.toModel()
	return &m, nil
}

func (c *HTTPClient) ClosedUnreleasedMilestones(ctx context.Context, tagTemplate string) ([]model.Milestone, error) {
	closed, err := c.milestones(ctx, "closed")
	if err != nil {
		return nil, err
	}
	released, err := c.releasedMilestoneTitles(ctx)
	if err != nil {
		return nil, err
	}
	var out []model.Milestone
	for _, m := range closed {
		if _, done := released[m.Title]; done {
			continue
		}
		out = append(out, m.toModel())
	}
	return out, nil
}

func (c *HTTPClient) releasedMilestoneTitles(ctx context.Context) (map[string]struct{}, error) {
	var rels []struct {
		Milestones []struct {
			Title string `json:"title"`
		} `json:"milestones"`
	}
	if err := c.do(ctx, http.MethodGet, c.projectPath("/releases?per_page=100"), nil, &rels); err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, r := range rels {
		for _, m := range r.Milestones {
			set[m.Title] = struct{}{}
		}
	}
	return set, nil
}

func (c *HTTPClient) MergedMRs(ctx context.Context, milestone model.Milestone) ([]model.MergeRequest, error) {
	q := url.Values{
		"milestone": {milestone.Title},
		"state":     {"merged"},
		"per_page":  {"100"},
	}
	var out []wireMR
	if err := c.do(ctx, http.MethodGet, c.projectPath("/merge_requests?"+q.Encode()), nil, &out); err != nil {
		return nil, err
	}
	mrs := make([]model.MergeRequest, 0, len(out))
	for _, w := range out {
		mrs = append(mrs, w.toModel())
	}
	return mrs, nil
}

func (c *HTTPClient) MRsForPipeline(ctx context.Context) ([]model.MergeRequest, error) {
	if c.mrIID == 0 {
		return nil, nil
	}
	var w wireMR
	path := c.projectPath(fmt.Sprintf("/merge_requests/%d", c.mrIID))
	if err := c.do(ctx, http.MethodGet, path, nil, &w); err != nil {
		return nil, err
	}
	mr := w.toModel()
	files, err := c.mrChangedFiles(ctx, c.mrIID)
	if err != nil {
		return nil, err
	}
	mr.ChangedFiles = files
	return []model.MergeRequest{mr}, nil
}

func (c *HTTPClient) mrChangedFiles(ctx context.Context, iid int) ([]string, error) {
	var resp struct {
		Changes []struct {
			NewPath string `json:"new_path"`
		} `json:"changes"`
	}
	path := c.projectPath(fmt.Sprintf("/merge_requests/%d/changes", iid))
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	files := make([]string, 0, len(resp.Changes))
	for _, ch := range resp.Changes {
		files = append(files, ch.NewPath)
	}
	return files, nil
}

func (c *HTTPClient) Tags(ctx context.Context) ([]string, error) {
	var out []struct {
		Name string `json:"name"`
	}
	if err := c.do(ctx, http.MethodGet, c.projectPath("/repository/tags?per_page=100"), nil, &out); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(out))
	for _, t := range out {
		tags = append(tags, t.Name)
	}
	return tags, nil
}

func (c *HTTPClient) AddMRLabels(ctx context.Context, mrIID int, labels []string) error {
	q := url.Values{"add_labels": {strings.Join(labels, ",")}}
	path := c.projectPath(fmt.Sprintf("/merge_requests/%d?%s", mrIID, q.Encode()))
	return c.do(ctx, http.MethodPut, path, nil, nil)
}

func (c *HTTPClient) CreateTag(ctx context.Context, tag, ref string) error {
	q := url.Values{"tag_name": {tag}, "ref": {ref}}
	path := c.projectPath("/repository/tags?" + q.Encode())
	err := c.do(ctx, http.MethodPost, path, nil, nil)
	// A pre-existing tag is not an error for our purposes.
	if apiErr, ok := err.(*APIError); ok && apiErr.Status == http.StatusBadRequest &&
		strings.Contains(apiErr.Body, "already exists") {
		return nil
	}
	return err
}

func (c *HTTPClient) UpsertRelease(ctx context.Context, r model.Release) error {
	payload := map[string]any{
		"tag_name":    r.TagName,
		"name":        r.Name,
		"description": r.Description,
	}
	if r.Milestone != "" {
		payload["milestones"] = []string{r.Milestone}
	}
	buf, _ := json.Marshal(payload)

	err := c.do(ctx, http.MethodPost, c.projectPath("/releases"), bytes.NewReader(buf), nil)
	if apiErr, ok := err.(*APIError); ok && (apiErr.Status == http.StatusConflict ||
		strings.Contains(apiErr.Body, "already exists")) {
		// Update the existing release.
		delete(payload, "tag_name")
		ubuf, _ := json.Marshal(payload)
		updatePath := c.projectPath("/releases/" + url.PathEscape(r.TagName))
		return c.do(ctx, http.MethodPut, updatePath, bytes.NewReader(ubuf), nil)
	}
	return err
}
