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

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/google/go-github/v25/github"
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
	return computed.Sum(nil)
}

// Hack to convert from github.PushEventRepository to github.Repository
func ConvertPushEventRepositoryToRepository(pushRepo *github.PushEventRepository) *github.Repository {
	repoName := pushRepo.GetFullName()
	private := pushRepo.GetPrivate()
	return &github.Repository{
		FullName: &repoName,
		Private:  &private,
	}
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
		mlog.Error("GitHub webhook content type should be set to \"application/json\"", mlog.Err(err))
		http.Error(w, "wrong mime-type. should be \"application/json\"", http.StatusBadRequest)
		return
	}

	var repo *github.Repository
	var handler func()

	switch event := event.(type) {
	case *github.PullRequestEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postPullRequestEvent(event)
			p.handlePullRequestNotification(event)
		}
	case *github.IssuesEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postIssueEvent(event)
			p.handleIssueNotification(event)
		}
	case *github.IssueCommentEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postIssueCommentEvent(event)
			p.handleCommentMentionNotification(event)
			p.handleCommentAuthorNotification(event)
		}
	case *github.PullRequestReviewEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postPullRequestReviewEvent(event)
			p.handlePullRequestReviewNotification(event)
		}
	case *github.PullRequestReviewCommentEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postPullRequestReviewCommentEvent(event)
		}
	case *github.PushEvent:
		repo = ConvertPushEventRepositoryToRepository(event.GetRepo())
		handler = func() {
			p.postPushEvent(event)
		}
	case *github.CreateEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postCreateEvent(event)
		}
	case *github.DeleteEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postDeleteEvent(event)
		}
	}

	if repo == nil || handler == nil {
		return
	}

	if repo.GetPrivate() && !config.EnablePrivateRepo {
		return
	}

	handler()
}

func (p *Plugin) permissionToRepo(userID string, ownerAndRepo string) bool {
	if userID == "" {
		return false
	}

	config := p.getConfiguration()
	ctx := context.Background()
	owner, repo := parseOwnerAndRepo(ownerAndRepo, config.EnterpriseBaseURL)

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
	githubClient := p.githubConnect(*info.Token)

	if result, _, err := githubClient.Repositories.Get(ctx, owner, repo); result == nil || err != nil {
		if err != nil {
			mlog.Error(err.Error())
		}
		return false
	}
	return true
}

func (p *Plugin) excludeConfigOrgMember(user *github.User, subscription *Subscription) bool {
	if !subscription.ExcludeOrgMembers() {
		return false
	}

	info, err := p.getGitHubUserInfo(subscription.CreatorID)
	if err != nil {
		mlog.Warn(err.Message)
		return false
	}

	githubClient := p.githubConnect(*info.Token)
	organization := p.getConfiguration().GitHubOrg

	return p.isUserOrganizationMember(githubClient, user, organization)
}

func (p *Plugin) postPullRequestEvent(event *github.PullRequestEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)
	if len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "opened" && action != "labeled" && action != "closed" {
		return
	}

	pr := event.GetPullRequest()
	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(pr.Labels))
	for i, v := range pr.Labels {
		labels[i] = v.GetName()
	}

	newPRMessage, err := renderTemplate("newPR", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	closedPRMessage, err := renderTemplate("closedPR", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId: p.BotUserID,
		Type:   "custom_git_pr",
	}

	for _, sub := range subs {
		if !sub.Pulls() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
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
				pullRequestLabelledMessage, err := renderTemplate("pullRequestLabelled", event)
				if err != nil {
					mlog.Error("failed to render template", mlog.Err(err))
					return
				}

				post.Message = pullRequestLabelledMessage
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
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)
	if len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "opened" && action != "labeled" && action != "closed" {
		return
	}

	issue := event.GetIssue()
	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(issue.Labels))
	for i, v := range issue.Labels {
		labels[i] = v.GetName()
	}

	newIssueMessage, err := renderTemplate("newIssue", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	closedIssueMessage, err := renderTemplate("closedIssue", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId: p.BotUserID,
		Type:   "custom_git_issue",
	}

	for _, sub := range subs {
		if !sub.Issues() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
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
				issueLabelledMessage, err := renderTemplate("issueLabelled", event)
				if err != nil {
					mlog.Error("failed to render template", mlog.Err(err))
					return
				}

				post.Message = issueLabelledMessage
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
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(ConvertPushEventRepositoryToRepository(repo))

	if len(subs) == 0 {
		return
	}

	commits := event.Commits
	if len(commits) == 0 {
		return
	}

	pushedCommitsMessage, err := renderTemplate("pushedCommits", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Type:    "custom_git_push",
		Message: pushedCommitsMessage,
	}

	for _, sub := range subs {
		if !sub.Pushes() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postCreateEvent(event *github.CreateEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	typ := event.GetRefType()

	if typ != "tag" && typ != "branch" {
		return
	}

	newCreateMessage, err := renderTemplate("newCreateMessage", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Type:    "custom_git_create",
		Message: newCreateMessage,
	}

	for _, sub := range subs {
		if !sub.Creates() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postDeleteEvent(event *github.DeleteEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	typ := event.GetRefType()

	if typ != "tag" && typ != "branch" {
		return
	}

	newDeleteMessage, err := renderTemplate("newDeleteMessage", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Type:    "custom_git_delete",
		Message: newDeleteMessage,
	}

	for _, sub := range subs {
		if !sub.Deletes() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postIssueCommentEvent(event *github.IssueCommentEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	if event.GetAction() != "created" {
		return
	}

	message, err := renderTemplate("issueComment", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId: p.BotUserID,
		Type:   "custom_git_comment",
	}

	labels := make([]string, len(event.GetIssue().Labels))
	for i, v := range event.GetIssue().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.IssueComments() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
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
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)
	if len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != "submitted" {
		return
	}

	switch event.GetReview().GetState() {
	case "APPROVED":
	case "COMMENTED":
	case "CHANGES_REQUESTED":
	default:
		mlog.Warn(fmt.Sprintf("unhandled review state %s", event.GetReview().GetState()))
		return
	}

	newReviewMessage, err := renderTemplate("pullRequestReviewEvent", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Type:    "custom_git_pull_review",
		Message: newReviewMessage,
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for i, v := range event.GetPullRequest().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.PullReviews() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
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
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)
	if len(subs) == 0 {
		return
	}

	newReviewMessage, err := renderTemplate("newReviewComment", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Type:    "custom_git_pull_review_comment",
		Message: newReviewMessage,
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for i, v := range event.GetPullRequest().Labels {
		labels[i] = v.GetName()
	}

	for _, sub := range subs {
		if !sub.PullReviews() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
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
	action := event.GetAction()
	if action == "edited" || action == "deleted" {
		return
	}

	body := event.GetComment().GetBody()

	// Try to parse out email footer junk
	if strings.Contains(body, "notifications@github.com") {
		body = strings.Split(body, "\n\nOn")[0]
	}

	mentionedUsernames := parseGitHubUsernamesFromText(body)

	message, err := renderTemplate("commentMentionNotification", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	post := &model.Post{
		UserId:  p.BotUserID,
		Message: message,
		Type:    "custom_git_mention",
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

		userId := p.getGitHubToUserIDMapping(username)
		if userId == "" {
			continue
		}

		if event.GetRepo().GetPrivate() && !p.permissionToRepo(userId, event.GetRepo().GetFullName()) {
			continue
		}

		channel, err := p.API.GetDirectChannel(userId, p.BotUserID)
		if err != nil {
			continue
		}

		post.ChannelId = channel.Id
		_, err = p.API.CreatePost(post)
		if err != nil {
			mlog.Error("Error creating mention post: " + err.Error())
		}

		p.sendRefreshEvent(p.BotUserID)
	}
}

func (p *Plugin) handleCommentAuthorNotification(event *github.IssueCommentEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}

	action := event.GetAction()
	if action == "edited" || action == "deleted" {
		return
	}

	authorUserID := p.getGitHubToUserIDMapping(author)
	if authorUserID == "" {
		return
	}

	if event.GetRepo().GetPrivate() && !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName()) {
		return
	}

	splitURL := strings.Split(event.GetIssue().GetHTMLURL(), "/")
	if len(splitURL) < 2 {
		return
	}

	var templateName string
	switch splitURL[len(splitURL)-2] {
	case "pull":
		templateName = "commentAuthorPullRequestNotification"
	case "issues":
		templateName = "commentAuthorIssueNotification"
	default:
		mlog.Warn(fmt.Sprintf("unhandled issue type %s", splitURL[len(splitURL)-2]))
		return
	}

	message, err := renderTemplate(templateName, event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	p.CreateBotDMPost(authorUserID, message, "custom_git_author")
	p.sendRefreshEvent(authorUserID)
}

func (p *Plugin) handlePullRequestNotification(event *github.PullRequestEvent) {
	author := event.GetPullRequest().GetUser().GetLogin()
	sender := event.GetSender().GetLogin()
	repoName := event.GetRepo().GetFullName()
	isPrivate := event.GetRepo().GetPrivate()

	requestedReviewer := ""
	requestedUserID := ""
	authorUserID := ""
	assigneeUserID := ""

	switch event.GetAction() {
	case "review_requested":
		requestedReviewer = event.GetRequestedReviewer().GetLogin()
		if requestedReviewer == sender {
			return
		}
		requestedUserID = p.getGitHubToUserIDMapping(requestedReviewer)
		if isPrivate && !p.permissionToRepo(requestedUserID, repoName) {
			requestedUserID = ""
		}
	case "closed":
		if author == sender {
			return
		}
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case "reopened":
		if author == sender {
			return
		}
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case "assigned":
		assignee := event.GetPullRequest().GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if isPrivate && !p.permissionToRepo(assigneeUserID, repoName) {
			assigneeUserID = ""
		}
	default:
		mlog.Warn(fmt.Sprintf("unhandled event action %s", event.GetAction()))
		return
	}

	message, err := renderTemplate("pullRequestNotification", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	if len(requestedUserID) > 0 {
		p.CreateBotDMPost(requestedUserID, message, "custom_git_review_request")
		p.sendRefreshEvent(requestedUserID)
	}

	p.postIssueNotification(message, authorUserID, assigneeUserID)
}

func (p *Plugin) handleIssueNotification(event *github.IssuesEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	sender := event.GetSender().GetLogin()
	if author == sender {
		return
	}
	repoName := event.GetRepo().GetFullName()
	isPrivate := event.GetRepo().GetPrivate()

	message := ""
	authorUserID := ""
	assigneeUserID := ""

	switch event.GetAction() {
	case "closed":
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case "reopened":
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case "assigned":
		assignee := event.GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if isPrivate && !p.permissionToRepo(assigneeUserID, repoName) {
			assigneeUserID = ""
		}
	default:
		mlog.Warn(fmt.Sprintf("unhandled event action %s", event.GetAction()))
		return
	}

	message, err := renderTemplate("issueNotification", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
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

	if event.GetRepo().GetPrivate() && !p.permissionToRepo(authorUserID, event.GetRepo().GetFullName()) {
		return
	}

	message, err := renderTemplate("pullRequestReviewNotification", event)
	if err != nil {
		mlog.Error("failed to render template", mlog.Err(err))
		return
	}

	p.CreateBotDMPost(authorUserID, message, "custom_git_review")
	p.sendRefreshEvent(authorUserID)
}
