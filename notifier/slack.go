package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"go.opencensus.io/plugin/ochttp"
)

type Slack struct {
	httpc           *http.Client
	baseURL, bearer string
}

func NewSlack(baseURL, bearer string) (*Slack, error) {
	switch "" {
	case baseURL:
		return nil, errors.New("baseURL must not be empty")
	case bearer:
		return nil, errors.New("bearer must not be empty")
	}
	if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("baseURL must be a valid http(s) URL")
	}

	return &Slack{
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

func (s *Slack) Notify(ctx context.Context, channel, content string) error {
	type textBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type block struct {
		Type string    `json:"type"`
		Text textBlock `json:"text"`
	}
	reqData := struct {
		Channel string  `json:"channel"`
		Blocks  []block `json:"blocks"`
	}{
		Channel: channel,
		Blocks: []block{
			{
				Type: "section",
				Text: textBlock{
					Type: "mrkdwn",
					Text: content,
				},
			},
		},
	}

	b, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("encoding/json: Marshal: %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/chat.postMessage", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("net/http: NewRequestWithContext: %s", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.bearer)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	res, err := s.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("net/http: Client.Do: %s", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("%T: Close: %s", res.Body, err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	b, err = ioutil.ReadAll(io.LimitReader(res.Body, 1024))
	if err != nil {
		return fmt.Errorf("io/ioutil: ReadAll: %s", err)
	}

	var resData struct{ OK bool }
	if err := json.Unmarshal(b, &resData); err != nil {
		return fmt.Errorf("encoding/json: Unmarshal: %s", err)
	}
	if !resData.OK {
		return fmt.Errorf("request was not successful with body: %q", b)
	}
	return nil
}
