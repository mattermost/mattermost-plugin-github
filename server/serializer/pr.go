package serializer

import "github.com/google/go-github/v48/github"

type PRDetails struct {
	URL                string                      `json:"url"`
	Number             int                         `json:"number"`
	Status             string                      `json:"status"`
	Mergeable          bool                        `json:"mergeable"`
	RequestedReviewers []*string                   `json:"requestedReviewers"`
	Reviews            []*github.PullRequestReview `json:"reviews"`
}
