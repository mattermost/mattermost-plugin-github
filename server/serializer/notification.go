package serializer

import "github.com/google/go-github/v48/github"

type FilteredNotification struct {
	github.Notification
	HTMLUrl string `json:"html_url"`
}
