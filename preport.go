package preport

import (
	"time"
)

type PullRequest struct {
	Title, URL string
	Author     Author
	CreatedAt  time.Time
}

type Author struct {
	Username string
}
