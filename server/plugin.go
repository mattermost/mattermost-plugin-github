package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"golang.org/x/oauth2"
)

const (
	GITHUB_TOKEN_KEY = "_githubtoken"
)

type Plugin struct {
	api           plugin.API
	configuration atomic.Value
	githubClient  *github.Client
	userId        string
}

func githubConnect(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func (p *Plugin) OnActivate(api plugin.API) error {
	p.api = api
	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	config := p.config()
	if err := config.IsValid(); err != nil {
		return err
	}

	// Connect to github
	p.githubClient = githubConnect(config.GithubToken)

	// Register commands
	p.api.RegisterCommand(&model.Command{
		Trigger:     "github",
		DisplayName: "Github",
		Description: "Integration with Github.",
	})

	// Get our userId
	user, err := p.api.GetUserByUsername(config.Username)
	if err != nil {
		return err
	}

	p.userId = user.Id

	return nil
}

func (p *Plugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	config := p.config()
	split := strings.Split(args.Command, " ")
	command := split[0]
	parameters := []string{}
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/github" {
		return nil, nil
	}

	switch action {
	case "subscribe":
		if len(parameters) != 1 {
			return &model.CommandResponse{Text: "Wrong number of parameters.", ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}, nil
		}
		subscriptions, _ := NewSubscriptionsFromKVStore(p.api.KeyValueStore())

		subscriptions.Add(args.ChannelId, parameters[0])

		subscriptions.StoreInKVStore(p.api.KeyValueStore())

		resp := &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			Text:         "You have subscribed to the repository.",
			Username:     "github",
			IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
			Type:         model.POST_DEFAULT,
		}
		return resp, nil
	case "register":
		if len(parameters) != 1 {
			return &model.CommandResponse{Text: "Wrong number of parameters.", ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}, nil
		}
		p.api.KeyValueStore().Set(args.UserId+GITHUB_TOKEN_KEY, []byte(parameters[0]))
		resp := &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Registered github token.",
			Username:     "github",
			IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
			Type:         model.POST_DEFAULT,
		}
		return resp, nil
	case "deregister":
		p.api.KeyValueStore().Delete(args.UserId + GITHUB_TOKEN_KEY)
		resp := &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Deregistered github token.",
			Username:     "github",
			IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
			Type:         model.POST_DEFAULT,
		}
		return resp, nil
	case "todo":
		prsToReview, err := p.HandleTodo(args.UserId, config.GithubOrg)
		if err != nil {
			return &model.CommandResponse{Text: err.Error(), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}, nil
		}
		resp := &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         prsToReview,
			Username:     "github",
			IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
			Type:         model.POST_DEFAULT,
		}
		return resp, nil
	}

	return nil, nil
}

func (p *Plugin) config() *Configuration {
	return p.configuration.Load().(*Configuration)
}

func (p *Plugin) OnConfigurationChange() error {
	var configuration Configuration
	err := p.api.LoadPluginConfiguration(&configuration)
	p.configuration.Store(&configuration)
	return err
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	config := p.config()
	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

	switch path := r.URL.Path; path {
	case "/webhook":
		p.handleWebhook(w, r)
	default:
		http.NotFound(w, r)
	}
}

type PullRequestWaitingReview struct {
	GitHubRepo        string `url:"github_repo"`
	GitHubUserName    string `url:"github_username"`
	PullRequestNumber int    `url:"pullrequest_number"`
	PullRequestURL    string `url:"pullrequest_url"`
}

type PullRequestWaitingReviews []PullRequestWaitingReview

func (p *Plugin) HandleTodo(userId, gitHubOrg string) (prToReview string, bad error) {
	ctx := context.Background()

	// TODO: not using now, need to remove the comment when we start post to the DM channel
	// Get the user Direct Channel to post the github todos
	// dmChannel, err := p.api.GetDirectChannel(userId, userId)
	// if err != nil {
	// 	return "", err
	// }

	b, err := p.api.KeyValueStore().Get(userId + GITHUB_TOKEN_KEY)
	if err != nil {
		return "", fmt.Errorf("Error retrieving the GitHub User token")
	}
	gitHubUserToken := string(b)

	githubClient := githubConnect(gitHubUserToken)

	// Get the user information. We need to know the username
	me, _, err2 := githubClient.Users.Get(ctx, "")
	if err2 != nil {
		return "", fmt.Errorf("Error retrieving the GitHub User information")
	}

	// Get all repositories for one specific Organization and after that get an PRs for
	// each repository that are waiting review from the user.
	var repos []string
	githubRepos, _, err2 := githubClient.Repositories.ListByOrg(ctx, gitHubOrg, nil)
	if err2 != nil {
		return "", fmt.Errorf("Error retrieving the GitHub repository")
	}
	for _, repo := range githubRepos {
		repos = append(repos, repo.GetName())
	}

	var prWaitingReviews PullRequestWaitingReviews
	for _, repo := range repos {
		prs, _, err := githubClient.PullRequests.List(ctx, gitHubOrg, repo, nil)
		if err != nil {
			return "", fmt.Errorf("Error retrieving the GitHub PRs List")
		}
		for _, pull := range prs {
			prReviewers, _, err := githubClient.PullRequests.ListReviewers(ctx, gitHubOrg, repo, pull.GetNumber(), nil)
			if err != nil {
				return "", fmt.Errorf("Error retrieving the GitHub PRs Reviewers")
			}
			for _, reviewer := range prReviewers.Users {
				if reviewer.GetLogin() == me.GetLogin() {
					prWaitingReviews = append(prWaitingReviews, PullRequestWaitingReview{repo, reviewer.GetLogin(), pull.GetNumber(), pull.GetHTMLURL()})
				}
			}
		}
	}

	if len(prWaitingReviews) == 0 {
		return "No pending PRs to review. Go and grab a coffee :smile:", nil
	}

	var buffer bytes.Buffer
	for _, toReview := range prWaitingReviews {
		buffer.WriteString(fmt.Sprintf("[**%v**] PRs waiting %v's review: **PR-%v** url: %v\n", toReview.GitHubRepo, toReview.GitHubUserName, toReview.PullRequestNumber, toReview.PullRequestURL))
	}

	// TODO: post to the direct channel
	// post := &model.Post{
	// 	UserId:    userId,
	// 	ChannelId: dmChannel.Id,
	// 	Message:   buffer.String(),
	// 	Type:      "github_todo",
	// }

	// if _, err := p.api.CreatePost(post); err != nil {
	// 	return fmt.Errorf("Error creating the post")
	// }

	return buffer.String(), nil
}

func NewString(st string) *string {
	return &st
}

func (p *Plugin) postFromPullRequest(pullRequest *github.PullRequest) *model.Post {
	props := map[string]interface{}{}
	//props["number"] =

	return &model.Post{
		UserId:  p.userId,
		Message: "Joram screwed up",
		Type:    model.POST_DEFAULT,
		Props:   props,
	}
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	config := p.config()

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.WebhookSecret)) != 1 {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	payload, _ := github.ValidatePayload(r, []byte(config.WebhookSecret))
	event, _ := github.ParseWebHook(github.WebHookType(r), payload)
	switch event := event.(type) {
	case *github.PullRequestEvent:
		p.pullRequestOpened(event.GetRepo().GetFullName(), event.PullRequest)
	}
}

func (p *Plugin) pullRequestOpened(repo string, pullRequest *github.PullRequest) {
	subscriptions, _ := NewSubscriptionsFromKVStore(p.api.KeyValueStore())

	channels := subscriptions.GetChannelsForRepository(repo)
	post := p.postFromPullRequest(pullRequest)
	for _, channel := range channels {
		post.ChannelId = channel
		p.api.CreatePost(post)
	}
}
