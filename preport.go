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

// PullRequestsBySortedAt providers a sorter based on CreatedAt timestamp, from
// oldest to newest.
type PullRequestsByCreatedAt []PullRequest

func (ps PullRequestsByCreatedAt) Len() int           { return len(ps) }
func (ps PullRequestsByCreatedAt) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps PullRequestsByCreatedAt) Less(i, j int) bool { return ps[i].CreatedAt.Before(ps[j].CreatedAt) }
