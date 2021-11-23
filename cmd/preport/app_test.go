package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epels/preport/internal/testutil"
)

func TestRun(t *testing.T) {
	var callsFirst, callsSecond int
	gitlabServer := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/foo/merge_requests":
			testutil.WriteTestdata(t, "testdata/gitlab_project_response_foo.json", w)
		case "/api/v4/projects/bar/merge_requests":
			testutil.WriteTestdata(t, "testdata/gitlab_project_response_bar.json", w)
		default:
			t.Errorf("Unexpected call to %q", r.URL.Path)
		}
	})
	slackServer := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Channel string
		}
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		err = json.Unmarshal(b, &req)
		require.NoError(t, err)

		switch req.Channel {
		case "first":
			callsFirst++
			testutil.AssertTestdataJSONEquals(t, "testdata/slack_request_first.json", bytes.NewReader(b))
		case "second":
			callsSecond++
			testutil.AssertTestdataJSONEquals(t, "testdata/slack_request_second.json", bytes.NewReader(b))
		default:
			t.Errorf("Unexpected call for channel %q", req.Channel)
		}
		testutil.WriteTestdata(t, "testdata/slack_response_ok.json", w)
	})

	genConf := generalConfig{
		NotifierConfig: `
{
  "notifiers": [
    {
      "channel": "first",
      "projects": [
        "foo"
      ]
    },
    {
      "channel": "second",
      "projects": [
        "foo",
        "bar"
      ]
    }
  ]
}
`,
		ReportTemplate: `{{range $pr := .}}{{$pr.URL}},{{$pr.Title}},{{$pr.Author.Username}},{{end}}`,
		Gitlab: struct {
			BaseURL string `required:"true" split_words:"true"`
			Bearer  string `required:"true" split_words:"true"`
		}{
			BaseURL: gitlabServer.URL,
			Bearer:  "gitlab-secret",
		},
		Slack: struct {
			BaseURL string `required:"true" split_words:"true"`
			Bearer  string `required:"true" split_words:"true"`
		}{
			BaseURL: slackServer.URL,
			Bearer:  "slack-secret",
		},
	}

	err := run(context.Background(), genConf, os.Stderr)
	require.NoError(t, err)
	assert.Equal(t, 1, callsFirst)
	assert.Equal(t, 1, callsSecond)
}
