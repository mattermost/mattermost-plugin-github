package model

type User struct {
	ID        int64  `json:"id,omitempty"`
	Login     string `json:"login,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	Name      string `json:"name,omitempty"`
}
