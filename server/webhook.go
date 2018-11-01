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
		if !event.GetRepo().GetPrivate() {
			p.postPullRequestEvent(event)
			p.handlePullRequestNotification(event)
		}
	case *github.IssuesEvent:
		if !event.GetRepo().GetPrivate() {
			p.postIssueEvent(event)
			p.handleIssueNotification(event)
		}
	case *github.IssueCommentEvent:
		if !event.GetRepo().GetPrivate() {
			p.handleCommentMentionNotification(event)
			p.handleCommentAuthorNotification(event)
		}
	case *github.PullRequestReviewEvent:
		if !event.GetRepo().GetPrivate() {
			p.handlePullRequestReviewNotification(event)
		}
	case *github.PushEvent:
		if !event.GetRepo().GetPrivate() {
			p.postPushEvent(event)
		}
	}
}

func (p *Plugin) postPullRequestEvent(event *github.PullRequestEvent) {
	repo := event.GetRepo()
	subs := p.GetSubscribedChannelsForRepository(repo.GetFullName())
	if subs == nil || len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "opened" && action != "labeled" {
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
	eventLabel := event.GetLabel().GetName()

	newPRMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-pull-request by [%s](%s) on [%s](%s)

%s
`, pr.GetTitle(), repo.GetFullName(), pr.GetNumber(), pr.GetHTMLURL(), prUser.GetLogin(), prUser.GetHTMLURL(), pr.GetCreatedAt().String(), pr.GetHTMLURL(), pr.GetBody())

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_pr",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	for _, sub := range subs {
		if !sub.Pulls() {
			continue
		}

		label := sub.Label()
		if action == "labeled" {
			if label != "" && label == eventLabel {
				post.Message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n#pull-request-labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", pr.GetTitle(), repo.GetFullName(), pr.GetNumber(), pr.GetHTMLURL(), eventLabel, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), pr.GetUpdatedAt().String(), pr.GetHTMLURL(), pr.GetBody())
			} else {
				continue
			}
		}

		if action == "opened" {
			if label == "" {
				post.Message = newPRMessage
			} else {
				continue
			}
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

	action := event.GetAction()
	if action != "opened" && action != "labeled" {
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
	eventLabel := event.GetLabel().GetName()

	newIssueMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-issue by [%s](%s) on [%s](%s)

%s
`, issue.GetTitle(), repo.GetFullName(), issue.GetNumber(), issue.GetHTMLURL(), issueUser.GetLogin(), issueUser.GetHTMLURL(), issue.GetCreatedAt().String(), issue.GetHTMLURL(), issue.GetBody())

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_issue",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	for _, sub := range subs {
		if !sub.Issues() {
			continue
		}

		label := sub.Label()
		if action == "labeled" {
			if label != "" && label == eventLabel {
				post.Message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n#issue-labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", issue.GetTitle(), repo.GetFullName(), issue.GetNumber(), issue.GetHTMLURL(), eventLabel, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), issue.GetUpdatedAt().String(), issue.GetHTMLURL(), issue.GetBody())
			} else {
				continue
			}
		}

		if action == "opened" {
			if label == "" {
				post.Message = newIssueMessage
			} else {
				continue
			}
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postPushEvent(event *github.PushEvent) {
	repo := event.GetRepo()
	subs := p.GetSubscribedChannelsForRepository(repo.GetFullName())

	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(p.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	forced := event.GetForced()
	branch := strings.Replace(event.GetRef(), "refs/heads/", "", 1)
	commits := event.Commits
	compare_url := event.GetCompare()
	pusher := event.GetSender()

	if len(commits) == 0 {
		return
	}

	fmtMessage := ``
	if forced {
		fmtMessage = "[%s](%s) force-pushed [%d new commits](%s) to [\\[%s:%s\\]](%s):\n"
	} else {
		fmtMessage = "[%s](%s) pushed [%d new commits](%s) to [\\[%s:%s\\]](%s):\n"
	}
	newPushMessage := fmt.Sprintf(fmtMessage, pusher.GetLogin(), pusher.GetHTMLURL(), len(commits), compare_url, repo.GetName(), branch, event.GetHeadCommit().GetURL())
	for _, commit := range commits {
		newPushMessage += fmt.Sprintf("[`%s`](%s) %s - %s\n",
			commit.GetID()[:6], commit.GetURL(), commit.GetMessage(), commit.GetCommitter().GetName())
	}

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_push",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
		Message: newPushMessage,
	}

	for _, sub := range subs {
		if !sub.Pushes() {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}

}

func (p *Plugin) handleCommentMentionNotification(event *github.IssueCommentEvent) {
	body := event.GetComment().GetBody()

	// Try to parse out email footer junk
	if strings.Contains(body, "notifications@github.com") {
		body = strings.Split(body, "\n\nOn")[0]
	}

	mentionedUsernames := parseGitHubUsernamesFromText(body)

	message := fmt.Sprintf("[%s](%s) mentioned you on [%s#%v](%s):\n>%s", event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetIssue().GetNumber(), event.GetComment().GetHTMLURL(), body)

	post := &model.Post{
		UserId:  p.BotUserID,
		Message: message,
		Type:    "custom_git_mention",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	for _, username := range mentionedUsernames {
		// Don't notify user of their own comment
		if username == event.GetSender().GetLogin() {
			continue
		}

		// Notifications for issue authors are handled separately
		if username == event.GetIssue().GetUser().GetLogin() {
			continue
		}

		userID := p.getGitHubToUserIDMapping(username)
		if userID == "" {
			continue
		}

		channel, err := p.API.GetDirectChannel(userID, p.BotUserID)
		if err != nil {
			continue
		}

		post.ChannelId = channel.Id
		_, err = p.API.CreatePost(post)
		if err != nil {
			mlog.Error("Error creating mention post: " + err.Error())
		}

		p.sendRefreshEvent(userID)
	}
}

func (p *Plugin) handleCommentAuthorNotification(event *github.IssueCommentEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}

	authorUserID := p.getGitHubToUserIDMapping(author)
	if authorUserID == "" {
		return
	}

	splitURL := strings.Split(event.GetIssue().GetHTMLURL(), "/")
	if len(splitURL) < 2 {
		return
	}

	message := ""
	switch splitURL[len(splitURL)-2] {
	case "pull":
		message = "[%s](%s) commented on your pull request [%s#%v](%s)"
	case "issues":
		message = "[%s](%s) commented on your issue [%s#%v](%s)"
	}

	message = fmt.Sprintf(message, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetIssue().GetNumber(), event.GetIssue().GetHTMLURL())

	p.CreateBotDMPost(authorUserID, message, "custom_git_author")
	p.sendRefreshEvent(authorUserID)
}

func (p *Plugin) handlePullRequestNotification(event *github.PullRequestEvent) {
	author := event.GetPullRequest().GetUser().GetLogin()
	sender := event.GetSender().GetLogin()

	requestedReviewer := ""
	requestedUserID := ""
	message := ""
	authorUserID := ""
	assigneeUserID := ""

	switch event.GetAction() {
	case "review_requested":
		requestedReviewer = event.GetRequestedReviewer().GetLogin()
		if requestedReviewer == sender {
			return
		}
		requestedUserID = p.getGitHubToUserIDMapping(requestedReviewer)
		message = "[%s](%s) requested your review on [%s#%v](%s)"
	case "closed":
		if author == sender {
			return
		}
		if event.GetPullRequest().GetMerged() {
			message = "[%s](%s) merged your pull request [%s#%v](%s)"
		} else {
			message = "[%s](%s) closed your pull request [%s#%v](%s)"
		}
		authorUserID = p.getGitHubToUserIDMapping(author)
	case "reopened":
		if author == sender {
			return
		}
		message = "[%s](%s) reopened your pull request [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
	case "assigned":
		assignee := event.GetPullRequest().GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		message = "[%s](%s) assigned you to pull request [%s#%v](%s)"
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
	}

	if len(message) > 0 {
		message = fmt.Sprintf(message, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetNumber(), event.GetPullRequest().GetHTMLURL())
	}

	if len(requestedUserID) > 0 {
		p.CreateBotDMPost(requestedUserID, message, "custom_git_review_request")
		p.sendRefreshEvent(requestedUserID)
	}

	p.postIssueNotification(message, authorUserID, assigneeUserID)
}

func (p *Plugin) handleIssueNotification(event *github.IssuesEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}
	message := ""
	authorUserID := ""
	assigneeUserID := ""

	switch event.GetAction() {
	case "closed":
		message = "[%s](%s) closed your issue [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
	case "reopened":
		message = "[%s](%s) reopened your issue [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
	case "assigned":
		message = "[%s](%s) assigned you to issue [%s#%v](%s)"
		assigneeUserID = p.getGitHubToUserIDMapping(event.GetAssignee().GetLogin())
	}

	if len(message) > 0 {
		message = fmt.Sprintf(message, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetIssue().GetNumber(), event.GetIssue().GetHTMLURL())
	}

	p.postIssueNotification(message, authorUserID, assigneeUserID)
}

func (p *Plugin) postIssueNotification(message, authorUserID, assigneeUserID string) {
	if len(authorUserID) > 0 {
		p.CreateBotDMPost(authorUserID, message, "custom_git_author")
		p.sendRefreshEvent(authorUserID)
	}

	if len(assigneeUserID) > 0 {
		p.CreateBotDMPost(assigneeUserID, message, "custom_git_assigned")
		p.sendRefreshEvent(assigneeUserID)
	}
}

func (p *Plugin) handlePullRequestReviewNotification(event *github.PullRequestReviewEvent) {
	author := event.GetPullRequest().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}

	if event.GetAction() != "submitted" {
		return
	}

	authorUserID := p.getGitHubToUserIDMapping(author)
	if authorUserID == "" {
		return
	}

	message := ""
	switch event.GetReview().GetState() {
	case "approved":
		message = "[%s](%s) approved your pull request [%s#%v](%s)"
	case "changes_requested":
		message = "[%s](%s) requested changes on your pull request [%s#%v](%s)"
	case "commented":
		message = "[%s](%s) commented on your pull request [%s#%v](%s)"
	}

	message = fmt.Sprintf(message, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetPullRequest().GetNumber(), event.GetPullRequest().GetHTMLURL())

	p.CreateBotDMPost(authorUserID, message, "custom_git_review")
	p.sendRefreshEvent(authorUserID)
}
