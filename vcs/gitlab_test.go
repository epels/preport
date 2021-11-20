package vcs_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epels/preport"
	"github.com/epels/preport/internal/testutil"
	"github.com/epels/preport/vcs"
)

func TestNewGitlab(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		gc, err := vcs.NewGitlab("https://example.com", "bearer")
		require.NoError(t, err)
		assert.NotNil(t, gc)
	})
	t.Run("Invalid baseURL", func(t *testing.T) {
		_, err := vcs.NewGitlab("ftp://example.com", "bearer")
		require.Error(t, err)
	})
	t.Run("Empty baseURL", func(t *testing.T) {
		_, err := vcs.NewGitlab("", "bearer")
		require.Error(t, err)
	})
	t.Run("Empty bearer", func(t *testing.T) {
		_, err := vcs.NewGitlab("https://example.com", "")
		require.Error(t, err)
	})
}

func TestGitlab_ListPullRequests(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ts := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v4/projects/1234/merge_requests", r.URL.Path)
			assert.Equal(t, url.Values{
				"scope":           []string{"all"},
				"state":           []string{"opened"},
				"wip":             []string{"no"},
				"approved_by_ids": []string{"None"},
				"assignee_id":     []string{"None"},
				"reviewer_id":     []string{"None"},
				"sort":            []string{"desc"},
			}, r.URL.Query())
			assert.Equal(t, "Bearer super-secret", r.Header.Get("Authorization"))

			testutil.WriteTestdata(t, "testdata/ok_response.json", w)
		})

		gc, err := vcs.NewGitlab(ts.URL, "super-secret")
		require.NoError(t, err)

		prs, err := gc.ListPullRequests(context.Background(), "1234", vcs.GitlabOptions{
			Scope:           vcs.ScopeAll,
			State:           vcs.StateOpened,
			Sort:            vcs.SortDesc,
			IsDraft:         boolPointer(t, false),
			HasAssignee:     boolPointer(t, false),
			HasReviewer:     boolPointer(t, false),
			HasBeenApproved: boolPointer(t, false),
		})
		require.NoError(t, err)
		assert.Equal(t, []preport.PullRequest{
			{
				Title: "Add upload",
				URL:   "https://gitlab.com/group/repo/-/merge_requests/14",
				Author: preport.Author{
					Username: "epels",
				},
				CreatedAt: mustParseRFC3339(t, "2019-03-06T14:00:56.380Z"),
			},
			{
				Title: "Strip trailing newlines from log statements.",
				URL:   "https://gitlab.com/group/repo/-/merge_requests/13",
				Author: preport.Author{
					Username: "epels",
				},
				CreatedAt: mustParseRFC3339(t, "2019-03-02T14:54:51.051Z"),
			},
		}, prs)
	})

	t.Run("Round trip failed", func(t *testing.T) {
		invalidBaseURL := "https://DF977BEA-4295-4758-AFF9-0EBCB1F509E2.fail"
		gc, err := vcs.NewGitlab(invalidBaseURL, "super-secret")
		require.NoError(t, err)

		_, err = gc.ListPullRequests(context.Background(), "1234", vcs.GitlabOptions{})
		require.Error(t, err)
	})

	t.Run("Internal error", func(t *testing.T) {
		ts := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		gc, err := vcs.NewGitlab(ts.URL, "super-secret")
		require.NoError(t, err)

		_, err = gc.ListPullRequests(context.Background(), "1234", vcs.GitlabOptions{})
		require.Error(t, err)
	})
}

func boolPointer(t *testing.T, b bool) *bool {
	t.Helper()

	return &b
}

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()

	res, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("time: Parse: %s", err)
	}
	return res
}
