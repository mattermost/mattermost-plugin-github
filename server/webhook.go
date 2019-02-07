package main

import (
	"context"
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
	config := p.getConfiguration()

	signature := r.Header.Get("X-Hub-Signature")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if !verifyWebhookSignature([]byte(config.WebhookSecret), signature, body) {
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
		p.handlePullRequestNotification(event)
	case *github.IssuesEvent:
		p.postIssueEvent(event)
		p.handleIssueNotification(event)
	case *github.IssueCommentEvent:
		p.postIssueCommentEvent(event)
		p.handleCommentMentionNotification(event)
		p.handleCommentAuthorNotification(event)
	case *github.PullRequestReviewEvent:
		p.postPullRequestReviewEvent(event)
		p.handlePullRequestReviewNotification(event)
	case *github.PullRequestReviewCommentEvent:
		p.postPullRequestReviewCommentEvent(event)
	case *github.PushEvent:
		p.postPushEvent(event)
	case *github.CreateEvent:
		p.postCreateEvent(event)
	case *github.DeleteEvent:
		p.postDeleteEvent(event)
	}
}

func (p *Plugin) permissionToRepo(userID string, ownerAndRepo string) bool {
	if userID == "" {
		return false
	}

	config := p.getConfiguration()
	ctx := context.Background()
	_, owner, repo := parseOwnerAndRepo(ownerAndRepo, config.EnterpriseBaseURL)

	if owner == "" {
		return false
	}
	if err := p.checkOrg(owner); err != nil {
		return false
	}

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		return false
	}
	var githubClient *github.Client
	githubClient = p.githubConnect(*info.Token)

	if result, _, err := githubClient.Repositories.Get(ctx, owner, repo); result == nil || err != nil {
		if err != nil {
			mlog.Error(err.Error())
		}
		return false
	}
	return true
}

func (p *Plugin) postPullRequestEvent(event *github.PullRequestEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)
	if subs == nil || len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "opened" && action != "labeled" && action != "closed" {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	pr := event.GetPullRequest()
	prUser := pr.GetUser()
	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(pr.Labels))
	for i, v := range pr.Labels {
		labels[i] = v.GetName()
	}

	newPRMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-pull-request by [%s](%s) on [%s](%s)

%s
`, pr.GetTitle(), repo.GetFullName(), pr.GetNumber(), pr.GetHTMLURL(), prUser.GetLogin(), prUser.GetHTMLURL(), pr.GetCreatedAt().String(), pr.GetHTMLURL(), pr.GetBody())

	fmtCloseMessage := ""
	if pr.GetMerged() {
		fmtCloseMessage = "[%s] Pull request [#%v %s](%s) was merged by [%s](%s)"
	} else {
		fmtCloseMessage = "[%s] Pull request [#%v %s](%s) was closed by [%s](%s)"
	}
	closedPRMessage := fmt.Sprintf(fmtCloseMessage, repo.GetFullName(), pr.GetNumber(), pr.GetTitle(), pr.GetHTMLURL(), event.GetSender().GetLogin(), event.GetSender().GetHTMLURL())

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

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		if action == "labeled" {
			if label != "" && label == eventLabel {
				post.Message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n#pull-request-labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", pr.GetTitle(), repo.GetFullName(), pr.GetNumber(), pr.GetHTMLURL(), eventLabel, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), pr.GetUpdatedAt().String(), pr.GetHTMLURL(), pr.GetBody())
			} else {
				continue
			}
		}

		if action == "opened" {
			post.Message = newPRMessage
		}

		if action == "closed" {
			post.Message = closedPRMessage
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postIssueEvent(event *github.IssuesEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)
	if subs == nil || len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "opened" && action != "labeled" && action != "closed" {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	issue := event.GetIssue()
	issueUser := issue.GetUser()
	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(issue.Labels))
	for i, v := range issue.Labels {
		labels[i] = v.GetName()
	}

	newIssueMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
#new-issue by [%s](%s) on [%s](%s)

%s
`, issue.GetTitle(), repo.GetFullName(), issue.GetNumber(), issue.GetHTMLURL(), issueUser.GetLogin(), issueUser.GetHTMLURL(), issue.GetCreatedAt().String(), issue.GetHTMLURL(), issue.GetBody())

	closedIssueMessage := fmt.Sprintf("\\[%s] Issue [%s](%s) closed by [%s](%s)",
		repo.GetFullName(), issue.GetTitle(), issue.GetHTMLURL(), event.GetSender().GetLogin(), event.GetSender().GetHTMLURL())

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

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		if action == "labeled" {
			if label != "" && label == eventLabel {
				post.Message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n#issue-labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", issue.GetTitle(), repo.GetFullName(), issue.GetNumber(), issue.GetHTMLURL(), eventLabel, event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), issue.GetUpdatedAt().String(), issue.GetHTMLURL(), issue.GetBody())
			} else {
				continue
			}
		}

		if action == "opened" {
			post.Message = newIssueMessage
		}

		if action == "closed" {
			post.Message = closedIssueMessage
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postPushEvent(event *github.PushEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)

	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
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
		fmtMessage = "[%s](%s) force-pushed [%d new commits](%s) to [\\[%s:%s\\]](%s/tree/%s):\n"
	} else {
		fmtMessage = "[%s](%s) pushed [%d new commits](%s) to [\\[%s:%s\\]](%s/tree/%s):\n"
	}
	newPushMessage := fmt.Sprintf(fmtMessage, pusher.GetLogin(), pusher.GetHTMLURL(), len(commits), compare_url, repo.GetName(), branch, repo.GetHTMLURL(), branch)
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

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postCreateEvent(event *github.CreateEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)

	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	typ := event.GetRefType()
	sender := event.GetSender()
	name := event.GetRef()

	if typ != "tag" && typ != "branch" {
		return
	}

	newCreateMessage := fmt.Sprintf("[%s](%s) just created %s [\\[%s:%s\\]](%s/tree/%s)",
		sender.GetLogin(), sender.GetHTMLURL(), typ, repo.GetName(), name, repo.GetHTMLURL(), name)

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_create",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
		Message: newCreateMessage,
	}

	for _, sub := range subs {
		if !sub.Creates() {
			continue
		}

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postDeleteEvent(event *github.DeleteEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)

	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	typ := event.GetRefType()
	sender := event.GetSender()
	name := event.GetRef()

	if typ != "tag" && typ != "branch" {
		return
	}

	newDeleteMessage := fmt.Sprintf("[%s](%s) just deleted %s \\[%s:%s]",
		sender.GetLogin(), sender.GetHTMLURL(), typ, repo.GetName(), name)

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_delete",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
		Message: newDeleteMessage,
	}

	for _, sub := range subs {
		if !sub.Deletes() {
			continue
		}

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postIssueCommentEvent(event *github.IssueCommentEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)

	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	if event.GetAction() != "created" {
		return
	}

	body := event.GetComment().GetBody()

	// Try to parse out email footer junk
	if strings.Contains(body, "notifications@github.com") {
		body = strings.Split(body, "\n\nOn")[0]
	}

	message := fmt.Sprintf("[\\[%s\\]](%s) New comment by [%s](%s) on [#%v %s]:\n\n%s",
		repo.GetFullName(), repo.GetHTMLURL(), event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetIssue().GetNumber(), event.GetIssue().GetTitle(), body)

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_comment",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	labels := make([]string, len(event.GetIssue().Labels))
	for i, v := range event.GetIssue().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.IssueComments() {
			continue
		}

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		if event.GetAction() == "created" {
			post.Message = message
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postPullRequestReviewEvent(event *github.PullRequestReviewEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)
	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	action := event.GetAction()
	if action != "submitted" {
		return
	}

	state := event.GetReview().GetState()
	fmtReviewMessage := ""
	switch state {
	case "APPROVED":
		fmtReviewMessage = "[\\[%s\\]](%s) [%s](%s) approved [#%v %s](%s):\n\n%s"
	case "COMMENTED":
		fmtReviewMessage = "[\\[%s\\]](%s) [%s](%s) commented on [#%v %s](%s):\n\n%s"
	case "CHANGES_REQUESTED":
		fmtReviewMessage = "[\\[%s\\]](%s) [%s](%s) requested changes on [#%v %s](%s):\n\n%s"
	default:
		return
	}

	newReviewMessage := fmt.Sprintf(fmtReviewMessage, repo.GetFullName(), repo.GetHTMLURL(), event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetPullRequest().GetNumber(), event.GetPullRequest().GetTitle(), event.GetPullRequest().GetHTMLURL(), event.GetReview().GetBody())

	post := &model.Post{
		UserId:  userID,
		Type:    "custom_git_pull_review",
		Message: newReviewMessage,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for i, v := range event.GetPullRequest().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.PullReviews() {
			continue
		}

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postPullRequestReviewCommentEvent(event *github.PullRequestReviewCommentEvent) {
	config := p.getConfiguration()
	repo := event.GetRepo()

	isPrivate := repo.GetPrivate()
	if isPrivate && !config.EnablePrivateRepo {
		return
	}

	repoName := repo.GetFullName()
	subs := p.GetSubscribedChannelsForRepository(repoName)
	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	newReviewMessage := fmt.Sprintf("[\\[%s\\]](%s) New review comment by [%s](%s) on [#%v %s](%s):\n\n%s\n%s",
		repo.GetFullName(), repo.GetHTMLURL(), event.GetSender().GetLogin(), event.GetSender().GetHTMLURL(), event.GetPullRequest().GetNumber(), event.GetPullRequest().GetTitle(), event.GetPullRequest().GetHTMLURL(), event.GetComment().GetDiffHunk(), event.GetComment().GetBody())

	post := &model.Post{
		UserId:  userID,
		Type:    "custom_git_pull_review_comment",
		Message: newReviewMessage,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for i, v := range event.GetPullRequest().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.PullReviews() {
			continue
		}

		if isPrivate && !p.permissionToRepo(sub.CreatorID, repoName) {
			return
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) handleCommentMentionNotification(event *github.IssueCommentEvent) {
	config := p.getConfiguration()
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

		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(userID, event.GetRepo().GetFullName())) {
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
	config := p.getConfiguration()
	author := event.GetIssue().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}

	authorUserID := p.getGitHubToUserIDMapping(author)
	if authorUserID == "" {
		return
	}

	if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
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
	config := p.getConfiguration()
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
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(requestedUserID, event.GetRepo().GetFullName())) {
			return
		}
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
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
			return
		}
	case "reopened":
		if author == sender {
			return
		}
		message = "[%s](%s) reopened your pull request [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
			return
		}
	case "assigned":
		assignee := event.GetPullRequest().GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		message = "[%s](%s) assigned you to pull request [%s#%v](%s)"
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(assigneeUserID, event.GetRepo().GetFullName())) {
			return
		}
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
	config := p.getConfiguration()
	author := event.GetIssue().GetUser().GetLogin()
	sender := event.GetSender().GetLogin()
	if author == sender {
		return
	}
	message := ""
	authorUserID := ""
	assigneeUserID := ""

	switch event.GetAction() {
	case "closed":
		message = "[%s](%s) closed your issue [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
			return
		}
	case "reopened":
		message = "[%s](%s) reopened your issue [%s#%v](%s)"
		authorUserID = p.getGitHubToUserIDMapping(author)
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
			return
		}
	case "assigned":
		assignee := event.GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		message = "[%s](%s) assigned you to issue [%s#%v](%s)"
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(assigneeUserID, event.GetRepo().GetFullName())) {
			return
		}
	}

	if len(message) > 0 {
		message = fmt.Sprintf(message, sender, event.GetSender().GetHTMLURL(), event.GetRepo().GetFullName(), event.GetIssue().GetNumber(), event.GetIssue().GetHTMLURL())
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
	config := p.getConfiguration()
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

	if event.GetRepo().GetPrivate() && (!config.EnablePrivateRepo || !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName())) {
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
