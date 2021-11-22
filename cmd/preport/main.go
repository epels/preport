package main

import (
	"context"
	"encoding/json"
	"log"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/template"

	"github.com/kelseyhightower/envconfig"

	"github.com/epels/preport"
	"github.com/epels/preport/notifier"
	"github.com/epels/preport/vcs"
)

var generalConfig struct {
	NotifierConfig string `required:"true" split_words:"true"`
	ReportTemplate string `required:"true" split_words:"true"`
	Gitlab         struct {
		BaseURL string `required:"true" split_words:"true"`
		Bearer  string `required:"true" split_words:"true"`
	} `required:"true" split_words:"true"`
	Slack struct {
		BaseURL string `required:"true" split_words:"true"`
		Bearer  string `required:"true" split_words:"true"`
	} `required:"true" split_words:"true"`
}

var notifierConfig struct {
	Notifiers []struct {
		Channel  string
		Projects []string
	}
}

func main() {
	if err := envconfig.Process("", &generalConfig); err != nil {
		log.Fatalf("envconfig: Process: %s", err)
	}
	if err := json.Unmarshal([]byte(generalConfig.NotifierConfig), &notifierConfig); err != nil {
		log.Fatalf("encoding/json: Unmarshal: %s", err)
	}
	tmpl, err := template.New("pullrequests").Parse(generalConfig.ReportTemplate)
	if err != nil {
		log.Fatalf("text/template: Template.Parse: %s", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sc, err := notifier.NewSlack(generalConfig.Slack.BaseURL, generalConfig.Slack.Bearer)
	if err != nil {
		log.Fatalf("notifier; NewSlack: %s", err)
	}
	gc, err := vcs.NewGitlab(generalConfig.Gitlab.BaseURL, generalConfig.Gitlab.Bearer)
	if err != nil {
		log.Fatalf("vcs; NewGitlab: %s", err)
	}

	// First, create a flat map of projects and fetch every project's pull
	// requests just once.
	projectsToPullRequests := make(map[string][]preport.PullRequest)
	for _, n := range notifierConfig.Notifiers {
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
				log.Printf("vcs: Gitlab.ListPullRequests: %s", err)
				continue
			}
			projectsToPullRequests[p] = prs
		}
	}

	// Now send out a formatted Slack message to each channel, utilizing the
	// projects we fetched earlier.
	for _, n := range notifierConfig.Notifiers {
		prs := make([]preport.PullRequest, 0, len(n.Projects))
		for _, p := range n.Projects {
			pr, ok := projectsToPullRequests[p]
			if !ok {
				log.Printf("Missing PullRequest entry for %s; skipping", p)
				continue
			}
			prs = append(prs, pr...)
		}

		text := renderTemplate(tmpl, prs)
		if err := sc.Notify(ctx, n.Channel, text); err != nil {
			log.Printf("notifier: Slack.Notify: %s", err)
		}
	}
}

func renderTemplate(tmpl *template.Template, prs []preport.PullRequest) string {
	sort.Sort(preport.PullRequestsByCreatedAt(prs))

	var b strings.Builder
	if err := tmpl.Execute(&b, prs); err != nil {
		log.Fatalf("text/template: Template.Execute: %s", err)
	}
	return b.String()
}
