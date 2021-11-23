package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/epels/preport"
	"github.com/epels/preport/notifier"
	"github.com/epels/preport/vcs"
)

type notifierConfig struct {
	Notifiers []struct {
		Channel  string
		Projects []string
	}
}

func run(ctx context.Context, genConf generalConfig, stderr io.Writer) error {
	errLog := log.New(stderr, "", log.LstdFlags|log.Lshortfile)

	var notConf notifierConfig
	if err := json.Unmarshal([]byte(genConf.NotifierConfig), &notConf); err != nil {
		return fmt.Errorf("encoding/json: Unmarshal: %s", err)
	}
	tmpl, err := template.New("pullrequests").Parse(genConf.ReportTemplate)
	if err != nil {
		return fmt.Errorf("text/template: Template.Parse: %s", err)
	}

	sc, err := notifier.NewSlack(genConf.Slack.BaseURL, genConf.Slack.Bearer)
	if err != nil {
		return fmt.Errorf("notifier: NewSlack: %s", err)
	}
	gc, err := vcs.NewGitlab(genConf.Gitlab.BaseURL, genConf.Gitlab.Bearer)
	if err != nil {
		return fmt.Errorf("vcs: NewGitlab: %s", err)
	}

	// First, create a flat map of projects and fetch every project's pull
	// requests just once.
	projectsToPullRequests := make(map[string][]preport.PullRequest)
	for _, n := range notConf.Notifiers {
		for _, p := range n.Projects {
			if _, ok := projectsToPullRequests[p]; ok {
				continue
			}
			prs, err := gc.ListPullRequests(ctx, p, vcs.GitlabOptions{
				Scope:           vcs.ScopeAll,
				State:           vcs.StateOpened,
				IsDraft:         &vcs.False,
				HasAssignee:     &vcs.False,
				HasBeenApproved: &vcs.False,
				HasReviewer:     &vcs.False,
				Sort:            vcs.SortAsc,
			})
			if err != nil {
				errLog.Printf("vcs: Gitlab.ListPullRequests: %s", err)
				continue
			}
			projectsToPullRequests[p] = prs
		}
	}

	// Now send out a formatted Slack message to each channel, utilizing the
	// projects we fetched earlier.
	for _, n := range notConf.Notifiers {
		prs := make([]preport.PullRequest, 0, len(n.Projects))
		for _, p := range n.Projects {
			pr, ok := projectsToPullRequests[p]
			if !ok {
				errLog.Printf("Missing PullRequest entry for %s; skipping", p)
				continue
			}
			prs = append(prs, pr...)
		}

		text, err := renderTemplate(tmpl, prs)
		if err != nil {
			errLog.Printf("renderTemplate: %s", err)
			continue
		}
		if err := sc.Notify(ctx, n.Channel, text); err != nil {
			errLog.Printf("notifier: Slack.Notify: %s", err)
		}
	}
	return nil
}

func renderTemplate(tmpl *template.Template, prs []preport.PullRequest) (string, error) {
	sort.Sort(preport.PullRequestsByCreatedAt(prs))

	var b strings.Builder
	if err := tmpl.Execute(&b, prs); err != nil {
		return "", fmt.Errorf("text/template: Template.Execute: %s", err)
	}
	return b.String(), nil
}
