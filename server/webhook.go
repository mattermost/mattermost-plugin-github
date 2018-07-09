package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func verifyWebhookSignature(secret []byte, signature string, body []byte) bool {

	const signaturePrefix = "sha1="
	const signatureLength = 45

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))

	return hmac.Equal(signBody(secret, body), actual)
}

func signBody(secret, body []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if !verifyWebhookSignature([]byte(p.WebhookSecret), signature, body) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), body)
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	switch event := event.(type) {
	case *github.PullRequestEvent:
		p.postPullRequestEvent(event)
	case *github.IssuesEvent:
		p.postIssueEvent(event)
	}

}

func (p *Plugin) postPullRequestEvent(event *github.PullRequestEvent) {
	repo := event.GetRepo()
	subs := p.GetSubscribedChannelsForRepository(repo.GetFullName())
	if subs == nil || len(subs) == 0 {
		return
	}

	if event.GetAction() != "opened" {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(p.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	pr := event.GetPullRequest()
	prUser := pr.GetUser()

	message := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-pull-request by [%s](%s) on [%s](%s)

%s
`, pr.GetTitle(), repo.GetFullName(), pr.GetNumber(), pr.GetHTMLURL(), prUser.GetLogin(), prUser.GetHTMLURL(), pr.GetCreatedAt().String(), pr.GetHTMLURL(), pr.GetBody())

	post := &model.Post{
		UserId:  userID,
		Message: message,
		Type:    "custom_git_pr",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	for _, sub := range subs {
		if !strings.Contains(sub.Features, "pulls") {
			continue
		}
		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postIssueEvent(event *github.IssuesEvent) {
	repo := event.GetRepo()
	subs := p.GetSubscribedChannelsForRepository(repo.GetFullName())
	if subs == nil || len(subs) == 0 {
		return
	}

	if event.GetAction() != "opened" {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(p.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	issue := event.GetIssue()
	issueUser := issue.GetUser()

	message := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-issue by [%s](%s) on [%s](%s)

%s
`, issue.GetTitle(), repo.GetFullName(), issue.GetNumber(), issue.GetHTMLURL(), issueUser.GetLogin(), issueUser.GetHTMLURL(), issue.GetCreatedAt().String(), issue.GetHTMLURL(), issue.GetBody())

	post := &model.Post{
		UserId:  userID,
		Message: message,
		Type:    "custom_git_issue",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	for _, sub := range subs {
		if !strings.Contains(sub.Features, "issues") {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}
