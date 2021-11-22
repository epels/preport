package main

import (
	"context"
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

var config struct {
	ReportTemplate string `required:"true" split_words:"true"`
	Gitlab         struct {
		BaseURL   string `required:"true" split_words:"true"`
		Bearer    string `required:"true" split_words:"true"`
		ProjectID string `required:"true" split_words:"true"`
	} `required:"true" split_words:"true"`
	Slack struct {
		BaseURL string `required:"true" split_words:"true"`
		Bearer  string `required:"true" split_words:"true"`
		Channel string `required:"true" split_words:"true"`
	} `required:"true" split_words:"true"`
}

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("envconfig: Process: %s", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sc, err := notifier.NewSlack(config.Slack.BaseURL, config.Slack.Bearer)
	if err != nil {
		log.Fatalf("notifier; NewSlack: %s", err)
	}
	gc, err := vcs.NewGitlab(config.Gitlab.BaseURL, config.Gitlab.Bearer)
	if err != nil {
		log.Fatalf("vcs; NewGitlab: %s", err)
	}

	f := false
	prs, err := gc.ListPullRequests(ctx, config.Gitlab.ProjectID, vcs.GitlabOptions{
		Scope:           vcs.ScopeAll,
		State:           vcs.StateOpened,
		IsDraft:         &f,
		HasAssignee:     &f,
		HasBeenApproved: &f,
		HasReviewer:     &f,
		Sort:            vcs.SortAsc,
	})
	if err != nil {
		log.Fatalf("vcs: Gitlab.ListPullRequests: %s", err)
	}

	text := format(prs)
	if err := sc.Notify(ctx, config.Slack.Channel, text); err != nil {
		log.Printf("notifier: Slack.Notify: %s", err)
	}
}

func format(prs []preport.PullRequest) string {
	sort.Sort(preport.PullRequestsByCreatedAt(prs))

	tmpl, err := template.New("pullrequests").Parse(config.ReportTemplate)
	if err != nil {
		panic(err)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, prs); err != nil {
		panic(err)
	}
	return b.String()
}
