package vcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.opencensus.io/plugin/ochttp"

	"github.com/epels/preport"
)

type Gitlab struct {
	httpc           *http.Client
	baseURL, bearer string
}

type mergeRequestsResponse []mergeRequestResponse

type mergeRequestResponse struct {
	Title  string
	WebURL string `json:"web_url"`
	Author struct {
		Username string
	}
	CreatedAt time.Time `json:"created_at"`
}

type (
	Scope string
	Sort  string
	State string
)

// GitlabOptions are parameters used to filter the pull requests; values with
// the respective type's zero values are discarded.
type GitlabOptions struct {
	Scope           Scope
	State           State
	IsDraft         *bool
	HasAssignee     *bool
	HasBeenApproved *bool
	HasReviewer     *bool
	Sort            Sort
	PerPage         int
}

const (
	ScopeCreatedByMe  Scope = "created_by_me"
	ScopeAssignedToMe Scope = "assigned_to_me"
	ScopeAll          Scope = "all"

	SortAsc  Sort = "asc"
	SortDesc Sort = "desc"

	StateClosed State = "closed"
	StateLocked State = "locked"
	StateMerged State = "merged"
	StateOpened State = "opened"
)

func NewGitlab(baseURL, bearer string) (*Gitlab, error) {
	switch "" {
	case baseURL:
		return nil, errors.New("baseURL must not be empty")
	case bearer:
		return nil, errors.New("bearer must not be empty")
	}
	if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("baseURL must be a valid http(s) URL")
	}

	return &Gitlab{
		httpc: &http.Client{
			Transport: &ochttp.Transport{},
			// Timeout is a generous duration intended as a fallback for when
			// the caller does not provide a context with a sensible deadline.
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		bearer:  bearer,
	}, nil
}

func (g *Gitlab) ListPullRequests(ctx context.Context, projectID string, opts GitlabOptions) ([]preport.PullRequest, error) {
	vals, err := opts.toValues()
	if err != nil {
		return nil, fmt.Errorf("unable to validate GitlabOptions: %s", err)
	}

	u := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests?%s", g.baseURL, projectID, vals.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("net/http: NewRequestWithContext: %s", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.bearer)

	res, err := g.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("net/http: Client.Do: %s", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("%T: Close: %s", res.Body, err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var rs mergeRequestsResponse
	if err := json.NewDecoder(res.Body).Decode(&rs); err != nil {
		return nil, fmt.Errorf("encoding/json: Decoder.Decode: %s", err)
	}
	return rs.toPullRequests(), nil
}

func (rs mergeRequestsResponse) toPullRequests() []preport.PullRequest {
	prs := make([]preport.PullRequest, 0, len(rs))
	for _, r := range rs {
		prs = append(prs, r.toPullRequest())
	}
	return prs
}

func (r mergeRequestResponse) toPullRequest() preport.PullRequest {
	return preport.PullRequest{
		Title: r.Title,
		URL:   r.WebURL,
		Author: preport.Author{
			Username: r.Author.Username,
		},
		CreatedAt: r.CreatedAt,
	}
}

func (o GitlabOptions) validate() error {
	switch o.Scope {
	case "", ScopeCreatedByMe, ScopeAssignedToMe, ScopeAll:
	default:
		return fmt.Errorf("unexpected scope: %q", o.Scope)
	}

	switch o.Sort {
	case "", SortAsc, SortDesc:
	default:
		return fmt.Errorf("unexpected sort: %q", o.Sort)
	}

	switch o.State {
	case "", StateClosed, StateLocked, StateMerged, StateOpened:
	default:
		return fmt.Errorf("unexpected state: %q", o.State)
	}

	return nil
}

func (o GitlabOptions) toValues() (url.Values, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}

	v := url.Values{}
	if o.Scope != "" {
		v.Set("scope", string(o.Scope))
	}
	if o.State != "" {
		v.Set("state", string(o.State))
	}
	if o.Sort != "" {
		v.Set("sort", string(o.Sort))
	}
	if o.IsDraft != nil {
		wip := "no"
		if *o.IsDraft {
			wip = "yes"
		}
		v.Set("wip", wip)
	}
	if o.HasAssignee != nil {
		assignee := "None"
		if *o.HasAssignee {
			assignee = "Any"
		}
		v.Set("assignee_id", assignee)
	}
	if o.HasBeenApproved != nil {
		approvedBy := "None"
		if *o.HasBeenApproved {
			approvedBy = "Any"
		}
		v.Set("approved_by_ids", approvedBy)
	}
	if o.HasReviewer != nil {
		reviewer := "None"
		if *o.HasReviewer {
			reviewer = "Any"
		}
		v.Set("reviewer_id", reviewer)
	}
	perPage := 100
	if o.PerPage != 0 {
		perPage = o.PerPage
	}
	v.Set("per_page", strconv.Itoa(perPage))
	return v, nil
}
