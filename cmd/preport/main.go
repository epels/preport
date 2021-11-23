package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
)

type generalConfig struct {
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var gc generalConfig
	if err := envconfig.Process("", &gc); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "envconfig: Process: %s\n", err)
		os.Exit(1)
	}
	if err := run(ctx, gc, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run: %s\n", err)
		os.Exit(1)
	}
}
