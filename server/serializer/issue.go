package serializer

type CreateIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Repo      string   `json:"repo"`
	PostID    string   `json:"post_id"`
	ChannelID string   `json:"channel_id"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
	Milestone int      `json:"milestone"`
}

type CreateIssueCommentRequest struct {
	PostID              string `json:"post_id"`
	Owner               string `json:"owner"`
	Repo                string `json:"repo"`
	Number              int    `json:"number"`
	Comment             string `json:"comment"`
	ShowAttachedMessage bool   `json:"show_attached_message"`
}

type UpdateIssueRequest struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Repo        string   `json:"repo"`
	PostID      string   `json:"post_id"`
	ChannelID   string   `json:"channel_id"`
	Labels      []string `json:"labels"`
	Assignees   []string `json:"assignees"`
	Milestone   int      `json:"milestone"`
	IssueNumber int      `json:"issue_number"`
}

type CommentAndCloseRequest struct {
	ChannelID    string `json:"channel_id"`
	IssueComment string `json:"issue_comment"`
	StatusReason string `json:"status_reason"`
	Number       int    `json:"number"`
	Owner        string `json:"owner"`
	Repository   string `json:"repo"`
	Status       string `json:"status"`
	PostID       string `json:"postId"`
}
