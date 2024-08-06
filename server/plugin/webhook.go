package plugin

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // GitHub webhooks are signed using sha1 https://developer.github.com/webhooks/.
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v54/github"
	"github.com/microcosm-cc/bluemonday"

	"github.com/mattermost/mattermost/server/public/model"
)

// RenderConfig holds various configuration options to be used in a template
// for redering an event.
type RenderConfig struct {
	Style string
}

// EventWithRenderConfig holds an event along with configuration options for
// rendering.
type EventWithRenderConfig struct {
	Event  interface{}
	Config RenderConfig
}

func verifyWebhookSignature(secret []byte, signature string, body []byte) (bool, error) {
	const signaturePrefix = "sha1="
	const signatureLength = 45

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false, nil
	}

	actual := make([]byte, 20)
	_, err := hex.Decode(actual, []byte(signature[5:]))
	if err != nil {
		return false, err
	}

	sb, err := signBody(secret, body)
	if err != nil {
		return false, err
	}

	return hmac.Equal(sb, actual), nil
}

func signBody(secret, body []byte) ([]byte, error) {
	computed := hmac.New(sha1.New, secret)
	_, err := computed.Write(body)
	if err != nil {
		return nil, err
	}

	return computed.Sum(nil), nil
}

// GetEventWithRenderConfig wraps any github Event into an EventWithRenderConfig
// which also contains per-subscription configuration options.
func GetEventWithRenderConfig(event interface{}, sub *Subscription) *EventWithRenderConfig {
	style := ""
	if sub != nil {
		style = sub.RenderStyle()
	}

	return &EventWithRenderConfig{
		Event: event,
		Config: RenderConfig{
			Style: style,
		},
	}
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

// WebhookBroker is a message broker for webhook events.
type WebhookBroker struct {
	sendGitHubPingEvent func(event *github.PingEvent)

	lock     sync.RWMutex // Protects closed and pingSubs
	closed   bool
	pingSubs []chan *github.PingEvent
}

func NewWebhookBroker(sendGitHubPingEvent func(event *github.PingEvent)) *WebhookBroker {
	return &WebhookBroker{
		sendGitHubPingEvent: sendGitHubPingEvent,
	}
}

func (wb *WebhookBroker) SubscribePings() <-chan *github.PingEvent {
	wb.lock.Lock()
	defer wb.lock.Unlock()

	ch := make(chan *github.PingEvent, 1)
	wb.pingSubs = append(wb.pingSubs, ch)

	return ch
}

func (wb *WebhookBroker) UnsubscribePings(ch <-chan *github.PingEvent) {
	wb.lock.Lock()
	defer wb.lock.Unlock()

	for i, sub := range wb.pingSubs {
		if sub == ch {
			wb.pingSubs = append(wb.pingSubs[:i], wb.pingSubs[i+1:]...)
			break
		}
	}
}

func (wb *WebhookBroker) publishPing(event *github.PingEvent, fromCluster bool) {
	wb.lock.Lock()
	defer wb.lock.Unlock()

	if wb.closed {
		return
	}

	for _, sub := range wb.pingSubs {
		// non-blocking send
		select {
		case sub <- event:
		default:
		}
	}

	if !fromCluster {
		wb.sendGitHubPingEvent(event)
	}
}

func (wb *WebhookBroker) Close() {
	wb.lock.Lock()
	defer wb.lock.Unlock()

	if !wb.closed {
		wb.closed = true

		for _, sub := range wb.pingSubs {
			close(sub)
		}
	}
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Hub-Signature")
	valid, err := verifyWebhookSignature([]byte(config.WebhookSecret), signature, body)
	if err != nil {
		p.client.Log.Warn("Failed to verify webhook signature", "error", err.Error())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), body)
	if err != nil {
		p.client.Log.Debug("GitHub webhook content type should be set to \"application/json\"", "error", err.Error())
		http.Error(w, "wrong mime-type. should be \"application/json\"", http.StatusBadRequest)
		return
	}

	if config.EnableWebhookEventLogging {
		bodyByte, err := json.Marshal(event)
		if err != nil {
			p.client.Log.Warn("Error while Marshal Webhook Request", "error", err.Error())
			http.Error(w, "Error while Marshal Webhook Request", http.StatusBadRequest)
			return
		}
		p.client.Log.Debug("Webhook Event Log", "event", string(bodyByte))
	}

	var repo *github.Repository
	var handler func()

	switch event := event.(type) {
	case *github.PingEvent:
		handler = func() {
			p.webhookBroker.publishPing(event, false)
		}
	case *github.PullRequestEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postPullRequestEvent(event)
			p.handlePullRequestNotification(event)
			p.handlePRDescriptionMentionNotification(event)
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
			p.handleCommentAssigneeNotification(event)
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
	case *github.StarEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postStarEvent(event)
		}
	case *github.ReleaseEvent:
		repo = event.GetRepo()
		handler = func() {
			p.postReleaseEvent(event)
		}
	}

	if handler == nil {
		return
	}

	if repo != nil && repo.GetPrivate() && !config.EnablePrivateRepo {
		return
	}

	handler()
}

func (p *Plugin) permissionToRepo(userID string, ownerAndRepo string) bool {
	if userID == "" {
		return false
	}

	config := p.getConfiguration()

	owner, repo := parseOwnerAndRepo(ownerAndRepo, config.getBaseURL())

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
	ctx := context.Background()
	githubClient := p.githubConnectUser(ctx, info)

	if result, _, err := githubClient.Repositories.Get(ctx, owner, repo); result == nil || err != nil {
		if err != nil {
			p.client.Log.Warn("Failed fetch repository to check permission", "error", err.Error())
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
		p.client.Log.Warn("Failed to exclude org member", "error", err.Message)
		return false
	}

	githubClient := p.githubConnectUser(context.Background(), info)
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
	switch action {
	case actionOpened,
		actionReopened,
		actionMarkedReadyForReview,
		actionLabeled,
		actionClosed:
	default:
		return
	}

	pr := event.GetPullRequest()
	isPRInDraftState := pr.GetDraft()
	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(pr.Labels))
	for index, label := range pr.Labels {
		labels[index] = label.GetName()
	}

	closedPRMessage, err := renderTemplate("closedPR", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Pulls() && !sub.PullsMerged() && !sub.PullsCreated() {
			continue
		}

		if sub.PullsMerged() && action != actionClosed {
			continue
		}

		if sub.PullsCreated() && action != actionOpened {
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

		repoName := strings.ToLower(repo.GetFullName())
		prNumber := event.GetPullRequest().Number

		post := p.makeBotPost("", "custom_git_pr")

		post.AddProp(postPropGithubRepo, repoName)
		post.AddProp(postPropGithubObjectID, prNumber)
		post.AddProp(postPropGithubObjectType, githubObjectTypeIssue)

		if action == actionLabeled {
			if label != "" && label == eventLabel {
				pullRequestLabelledMessage, err := renderTemplate("pullRequestLabelled", event)
				if err != nil {
					p.client.Log.Warn("Failed to render template", "error", err.Error())
					return
				}

				post.Message = pullRequestLabelledMessage
			} else {
				continue
			}
		}

		if action == actionOpened {
			prNotificationType := "newPR"
			if isPRInDraftState {
				prNotificationType = "newDraftPR"
			}
			newPRMessage, err := renderTemplate(prNotificationType, GetEventWithRenderConfig(event, sub))
			if err != nil {
				p.client.Log.Warn("Failed to render template", "error", err.Error())
				return
			}

			post.Message = p.sanitizeDescription(newPRMessage)
		}

		if action == actionReopened {
			reopenedPRMessage, err := renderTemplate("reopenedPR", event)
			if err != nil {
				p.client.Log.Warn("Failed to render template", "error", err.Error())
				return
			}

			post.Message = p.sanitizeDescription(reopenedPRMessage)
		}

		if action == actionMarkedReadyForReview {
			markedReadyToReviewPRMessage, err := renderTemplate("markedReadyToReviewPR", GetEventWithRenderConfig(event, sub))
			if err != nil {
				p.client.Log.Warn("Failed to render template", "error", err.Error())
				return
			}

			post.Message = p.sanitizeDescription(markedReadyToReviewPRMessage)
		}

		if action == actionClosed {
			post.Message = closedPRMessage
		}

		post.ChannelId = sub.ChannelID
		if err := p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}
	}
}

func (p *Plugin) sanitizeDescription(description string) string {
	if strings.Contains(description, "<details>") {
		var policy = bluemonday.StrictPolicy()
		policy.SkipElementsContent("details")
		description = html.UnescapeString(policy.Sanitize(description))
	}
	return strings.TrimSpace(description)
}

func (p *Plugin) handlePRDescriptionMentionNotification(event *github.PullRequestEvent) {
	action := event.GetAction()
	if action != actionOpened {
		return
	}

	body := event.GetPullRequest().GetBody()

	mentionedUsernames := parseGitHubUsernamesFromText(body)

	message, err := renderTemplate("pullRequestMentionNotification", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, username := range mentionedUsernames {
		// Don't notify user of their own comment
		if username == event.GetSender().GetLogin() {
			continue
		}

		// Notifications for pull request authors are handled separately
		if username == event.GetPullRequest().GetUser().GetLogin() {
			continue
		}

		userID := p.getGitHubToUserIDMapping(username)
		if userID == "" {
			continue
		}

		if event.GetRepo().GetPrivate() && !p.permissionToRepo(userID, event.GetRepo().GetFullName()) {
			continue
		}

		channel, err := p.client.Channel.GetDirect(userID, p.BotUserID)
		if err != nil {
			continue
		}

		post := p.makeBotPost(message, "custom_git_mention")
		post.ChannelId = channel.Id

		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}

		p.sendRefreshEvent(userID)
	}
}

func (p *Plugin) postIssueEvent(event *github.IssuesEvent) {
	repo := event.GetRepo()
	issue := event.GetIssue()
	action := event.GetAction()

	// This condition is made to check if the message doesn't get automatically labeled to prevent duplicated issue messages
	timeDiff := time.Until(issue.GetCreatedAt().Time) * -1
	if action == actionLabeled && timeDiff.Seconds() < 4.00 {
		return
	}

	subscribedChannels := p.GetSubscribedChannelsForRepository(repo)
	if len(subscribedChannels) == 0 {
		return
	}

	issueTemplate := ""
	switch action {
	case actionOpened:
		issueTemplate = "newIssue"

	case actionClosed:
		issueTemplate = "closedIssue"

	case actionReopened:
		issueTemplate = "reopenedIssue"

	case actionLabeled:
		issueTemplate = "issueLabelled"

	default:
		return
	}

	eventLabel := event.GetLabel().GetName()
	labels := make([]string, len(issue.Labels))
	for index, label := range issue.Labels {
		labels[index] = label.GetName()
	}

	for _, sub := range subscribedChannels {
		if !sub.Issues() && !sub.IssueCreations() {
			continue
		}

		if sub.IssueCreations() && action != actionOpened && action != actionReopened && action != actionLabeled {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		renderedMessage, err := renderTemplate(issueTemplate, GetEventWithRenderConfig(event, sub))
		if err != nil {
			p.client.Log.Warn("Failed to render template", "error", err.Error())
			return
		}
		renderedMessage = p.sanitizeDescription(renderedMessage)

		assignees := make([]string, len(issue.Assignees))
		for index, user := range issue.Assignees {
			assignees[index] = user.GetLogin()
		}
		description := ""
		if issue.Body != nil {
			description = *issue.Body
		}

		post := &model.Post{
			UserId: p.BotUserID,
			Type:   "custom_git_release",
		}

		if action == actionOpened {
			post.Props = model.StringInterface{
				"attachments": []*model.SlackAttachment{
					{
						Pretext:   renderedMessage,
						Title:     fmt.Sprintf("%s #%d", *issue.Title, *issue.Number),
						TitleLink: *issue.HTMLURL,
						Text:      description,
						Actions: []*model.PostAction{
							{
								Name: "Comment",
								Integration: &model.PostActionIntegration{
									Context: map[string]interface{}{
										KeyRepoOwner:   repo.GetOwner().GetLogin(),
										KeyRepoName:    repo.GetName(),
										KeyIssueNumber: issue.GetNumber(),
										KeyIssueID:     issue.GetID(),
										KeyStatus:      *issue.State,
									},
									URL: fmt.Sprintf("%s%s", p.GetPluginAPIPath(), PathOpenIssueCommentModal),
								},
								Style: "primary",
							},
							{
								Name: "Edit",
								Integration: &model.PostActionIntegration{
									Context: map[string]interface{}{
										KeyRepoOwner:   repo.GetOwner().GetLogin(),
										KeyRepoName:    repo.GetName(),
										KeyIssueNumber: issue.GetNumber(),
										KeyIssueID:     issue.GetID(),
										KeyStatus:      *issue.State,
									},
									URL: fmt.Sprintf("%s%s", p.GetPluginAPIPath(), PathOpenIssueEditModal),
								},
							},
							{
								Name: "Close",
								Integration: &model.PostActionIntegration{
									Context: map[string]interface{}{
										KeyRepoOwner:   repo.GetOwner().GetLogin(),
										KeyRepoName:    repo.GetName(),
										KeyIssueNumber: issue.GetNumber(),
										KeyIssueID:     issue.GetID(),
										KeyStatus:      *issue.State,
									},
									URL: fmt.Sprintf("%s%s", p.GetPluginAPIPath(), PathOpenIssueStatusModal),
								},
							},
						},
						Fields: p.CreateFieldsForIssuePost(assignees, labels),
					},
				},
			}
		}
		repoName := strings.ToLower(repo.GetFullName())
		issueNumber := issue.Number

		post.AddProp(postPropGithubRepo, repoName)
		post.AddProp(postPropGithubObjectID, issueNumber)
		post.AddProp(postPropGithubObjectType, githubObjectTypeIssue)

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

		if action == actionLabeled {
			if label == "" || label != eventLabel {
				continue
			}
		}

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
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

	setShowAuthorInCommitNotification(p.configuration.ShowAuthorInCommitNotification)
	pushedCommitsMessage, err := renderTemplate("pushedCommits", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Pushes() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post := p.makeBotPost(pushedCommitsMessage, "custom_git_push")

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
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
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Creates() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post := p.makeBotPost(newCreateMessage, "custom_git_create")

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
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
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Deletes() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post := p.makeBotPost(newDeleteMessage, "custom_git_delete")
		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}
	}
}

func (p *Plugin) postIssueCommentEvent(event *github.IssueCommentEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	if event.GetAction() != actionCreated {
		return
	}

	message, err := renderTemplate("issueComment", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	labels := make([]string, len(event.GetIssue().Labels))
	for index, label := range event.GetIssue().Labels {
		labels[index] = label.GetName()
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

		post := p.makeBotPost("", "custom_git_comment")

		repoName := strings.ToLower(repo.GetFullName())
		commentID := event.GetComment().GetID()

		post.AddProp(postPropGithubRepo, repoName)
		post.AddProp(postPropGithubObjectID, commentID)
		post.AddProp(postPropGithubObjectType, githubObjectTypeIssueComment)

		if event.GetAction() == actionCreated {
			post.Message = message
		}

		post.ChannelId = sub.ChannelID

		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}
	}
}

func (p *Plugin) senderMutedByReceiver(userID string, sender string) bool {
	var mutedUsernameBytes []byte
	if err := p.store.Get(fmt.Sprintf("%s-muted-users", userID), &mutedUsernameBytes); err != nil {
		p.client.Log.Warn("Failed to get muted users", "userID", userID)
		return false
	}

	mutedUsernames := string(mutedUsernameBytes)
	return strings.Contains(mutedUsernames, sender)
}

func (p *Plugin) postPullRequestReviewEvent(event *github.PullRequestReviewEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)
	if len(subs) == 0 {
		return
	}

	action := event.GetAction()
	if action != actionSubmitted {
		return
	}

	switch event.GetReview().GetState() {
	case "approved":
	case "commented":
	case "changes_requested":
	default:
		p.client.Log.Debug("Unhandled review state", "state", event.GetReview().GetState())
		return
	}

	newReviewMessage, err := renderTemplate("pullRequestReviewEvent", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for index, label := range event.GetPullRequest().Labels {
		labels[index] = label.GetName()
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

		post := p.makeBotPost(newReviewMessage, "custom_git_pull_review")

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
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
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	labels := make([]string, len(event.GetPullRequest().Labels))
	for index, label := range event.GetPullRequest().Labels {
		labels[index] = label.GetName()
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

		post := p.makeBotPost(newReviewMessage, "custom_git_pr_comment")

		repoName := strings.ToLower(repo.GetFullName())
		commentID := event.GetComment().GetID()

		post.AddProp(postPropGithubRepo, repoName)
		post.AddProp(postPropGithubObjectID, commentID)
		post.AddProp(postPropGithubObjectType, githubObjectTypePRReviewComment)

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}
	}
}

func (p *Plugin) handleCommentMentionNotification(event *github.IssueCommentEvent) {
	action := event.GetAction()
	if action == actionEdited || action == actionDeleted {
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
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	assignees := event.GetIssue().Assignees

	for _, username := range mentionedUsernames {
		assigneeMentioned := false
		for _, assignee := range assignees {
			if username == *assignee.Login {
				assigneeMentioned = true
				break
			}
		}

		// This has been handled in "handleCommentAssigneeNotification" function
		if assigneeMentioned {
			continue
		}

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

		if event.GetRepo().GetPrivate() && !p.permissionToRepo(userID, event.GetRepo().GetFullName()) {
			continue
		}

		channel, err := p.client.Channel.GetDirect(userID, p.BotUserID)
		if err != nil {
			continue
		}

		post := p.makeBotPost(message, "custom_git_mention")

		post.ChannelId = channel.Id
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error creating mention post", "error", err.Error())
		}

		p.sendRefreshEvent(userID)
	}
}

func (p *Plugin) handleCommentAuthorNotification(event *github.IssueCommentEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	if author == event.GetSender().GetLogin() {
		return
	}

	action := event.GetAction()
	if action == actionEdited || action == actionDeleted {
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
		p.client.Log.Debug("Unhandled issue type", "type", splitURL[len(splitURL)-2])
		return
	}

	if p.senderMutedByReceiver(authorUserID, event.GetSender().GetLogin()) {
		p.client.Log.Debug("Commenter is muted, skipping notification")
		return
	}

	message, err := renderTemplate(templateName, event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	p.CreateBotDMPost(authorUserID, message, "custom_git_author")
	p.sendRefreshEvent(authorUserID)
}

func (p *Plugin) handleCommentAssigneeNotification(event *github.IssueCommentEvent) {
	author := event.GetIssue().GetUser().GetLogin()
	assignees := event.GetIssue().Assignees
	repoName := event.GetRepo().GetFullName()

	splitURL := strings.Split(event.GetIssue().GetHTMLURL(), "/")
	if len(splitURL) < 2 {
		return
	}

	eventType := splitURL[len(splitURL)-2]
	var templateName string
	switch eventType {
	case "pull":
		templateName = "commentAssigneePullRequestNotification"
	case "issues":
		templateName = "commentAssigneeIssueNotification"
	default:
		p.client.Log.Debug("Unhandled issue type", "Type", eventType)
		return
	}

	mentionedUsernames := parseGitHubUsernamesFromText(event.GetComment().GetBody())

	for _, assignee := range assignees {
		usernameMentioned := false
		template := templateName
		for _, username := range mentionedUsernames {
			if username == *assignee.Login {
				usernameMentioned = true
				break
			}
		}

		if usernameMentioned {
			switch eventType {
			case "pull":
				template = "commentAssigneeSelfMentionPullRequestNotification"
			case "issues":
				template = "commentAssigneeSelfMentionIssueNotification"
			}
		}

		userID := p.getGitHubToUserIDMapping(assignee.GetLogin())
		if userID == "" {
			continue
		}

		if author == assignee.GetLogin() {
			continue
		}
		if event.Sender.GetLogin() == assignee.GetLogin() {
			continue
		}

		if !p.permissionToRepo(userID, repoName) {
			continue
		}

		assigneeID := p.getGitHubToUserIDMapping(assignee.GetLogin())
		if assigneeID == "" {
			continue
		}

		if p.senderMutedByReceiver(assigneeID, event.GetSender().GetLogin()) {
			p.client.Log.Debug("Commenter is muted, skipping notification")
			continue
		}

		message, err := renderTemplate(template, event)
		if err != nil {
			p.client.Log.Warn("Failed to render template", "error", err.Error())
			continue
		}
		p.CreateBotDMPost(assigneeID, message, "custom_git_assignee")
		p.sendRefreshEvent(assigneeID)
	}
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
	case actionClosed:
		if author == sender {
			return
		}
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case actionReopened:
		if author == sender {
			return
		}
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case actionAssigned:
		assignee := event.GetPullRequest().GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if isPrivate && !p.permissionToRepo(assigneeUserID, repoName) {
			assigneeUserID = ""
		}
	default:
		p.client.Log.Debug("Unhandled event action", "action", event.GetAction())
		return
	}

	message, err := renderTemplate("pullRequestNotification", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
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
	case actionClosed:
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case actionReopened:
		authorUserID = p.getGitHubToUserIDMapping(author)
		if isPrivate && !p.permissionToRepo(authorUserID, repoName) {
			authorUserID = ""
		}
	case actionAssigned:
		assignee := event.GetAssignee().GetLogin()
		if assignee == sender {
			return
		}
		assigneeUserID = p.getGitHubToUserIDMapping(assignee)
		if isPrivate && !p.permissionToRepo(assigneeUserID, repoName) {
			assigneeUserID = ""
		}
	default:
		p.client.Log.Debug("Unhandled event action", "action", event.GetAction())
		return
	}

	message, err := renderTemplate("issueNotification", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
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

	if event.GetAction() != actionSubmitted {
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
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	p.CreateBotDMPost(authorUserID, message, "custom_git_review")
	p.sendRefreshEvent(authorUserID)
}

func (p *Plugin) postStarEvent(event *github.StarEvent) {
	repo := event.GetRepo()

	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	newStarMessage, err := renderTemplate("newRepoStar", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Stars() {
			continue
		}

		if p.excludeConfigOrgMember(event.GetSender(), sub) {
			continue
		}

		post := p.makeBotPost(newStarMessage, "custom_git_star")

		post.ChannelId = sub.ChannelID
		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "post", post, "error", err.Error())
		}
	}
}

func (p *Plugin) makeBotPost(message, postType string) *model.Post {
	return &model.Post{
		UserId:  p.BotUserID,
		Type:    postType,
		Message: message,
	}
}

func (p *Plugin) postReleaseEvent(event *github.ReleaseEvent) {
	if event.GetAction() != actionCreated && event.GetAction() != actionDeleted {
		return
	}

	repo := event.GetRepo()
	subs := p.GetSubscribedChannelsForRepository(repo)

	if len(subs) == 0 {
		return
	}

	newReleaseMessage, err := renderTemplate("newReleaseEvent", event)
	if err != nil {
		p.client.Log.Warn("Failed to render template", "Error", err.Error())
		return
	}

	for _, sub := range subs {
		if !sub.Release() {
			continue
		}

		post := &model.Post{
			UserId:    p.BotUserID,
			Type:      "custom_git_release",
			Message:   newReleaseMessage,
			ChannelId: sub.ChannelID,
		}

		if err = p.client.Post.CreatePost(post); err != nil {
			p.client.Log.Warn("Error webhook post", "Post", post, "Error", err.Error())
		}
	}
}
