package model

type PullRequest struct {
	URL                string              `json:"url"`
	Number             int64               `json:"number"`
	Status             string              `json:"status"`
	Mergeable          bool                `json:"mergeable"`
	MergeableState     string              `json:"mergeable_state"`
	RequestedReviewers []string            `json:"requestedReviewers"`
	Reviews            []PullRequestReview `json:"reviews"`
}
