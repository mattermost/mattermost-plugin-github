package model

type PullRequestReview struct {
	ID     int64  `json:"id,omitempty"`
	NodeID string `json:"node_id,omitempty"`
	User   *User  `json:"user,omitempty"`
	Body   string `json:"body,omitempty"`
	State  string `json:"state,omitempty"`
	URL    string `json:"html_url,omitempty"`
}
