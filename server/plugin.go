package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"golang.org/x/oauth2"
)

type Plugin struct {
	api           plugin.API
	configuration atomic.Value
	githubClient  *github.Client
	context       context
}

func githubConnect(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client, ctx
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
	p.githubClient, p.context = githubConnect(config.GithubToken)

	// Register commands
	p.api.RegisterCommand(&model.Command{
		Trigger:     "github",
		DisplayName: "Github",
		Description: "Integration with Github.",
	})

	return nil
}

func (p *Plugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

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
			return nil, nil
		}
	case "register":
	case "todo":
		err := HandleTodo(args.UserId)
		if err != nil {
			return nil, nil
		}
	}

	resp := &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Text:         "You have subscribed to the repository.",
		Username:     "github",
		IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
		Type:         model.POST_DEFAULT,
	}

	return resp, nil
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

func (p *Plugin) HandleTodo(userId, gitHubOrg string) error {

	// Get the user Direct Channel to post the github todos
	dmChannel, err := p.api.GetDirectChannel(userId, userId)
	if err != nil {
		return err.Error()
	}

	// TODO: Get the user token
	gitHubUserToken, err := GetGitHubUserToken(userid)
	if err != nil {
		return fmt.Errorf("Error retrieving the GitHub User token")
	}

	githubClient, ctx := githubConnect(gitHubUserToken)

	// Get the user information. We need to know the username
	me, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("Error retrieving the GitHub User information")
	}

	// Get all repositories for one specific Organization and after that get an PRs for
	// each repository that are waiting review from the user.
	var repos []string
	githubRepos, _, err := githubClient.Repositories.ListByOrg(ctx, gitHubOrg, nil)
	if err != nil {
		return fmt.Errorf("Error retrieving the GitHub repository")
	}
	for _, repo := range githubRepos {
		repos = append(repos, repo.GetName())
	}

	var prWaitingReviews PullRequestWaitingReviews
	for _, repo := range repos {
		prs, _, err := githubClient.PullRequests.List(ctx, gitHubOrg, repo, opt)
		if err != nil {
			return fmt.Errorf("Error retrieving the GitHub PRs List")
		}
		for _, pull := range pulls {
			prReviewers, _, err := githubClient.PullRequests.ListReviewers(ctx, gitHubOrg, githubRepo, pull.GetNumber(), nil)
			if err != nil {
				return fmt.Errorf("Error retrieving the GitHub PRs Reviewers")
			}
			for _, reviewer := range prReviewers.Users {
				if reviewer.GetLogin() == me.GetLogin() {
					prWaitingReviews = append(prWaitingReviews, PullRequestWaitingReview{repo, reviewer.GetLogin(), pull.GetNumber(), pull.GetHTMLURL()})
				}
			}
		}
	}

	var buffer bytes.Buffer
	for _, toReview := range prWaitingReviews {
		for _, tt := range b {
			buffer.WriteString(fmt.Sprintf("[%v] PRs waiting %v review: PR-%v url: %v\n", toReview.GitHubRepo, toReview.GitHubUserName, toReview.PullRequestNumber, toReview.PullRequestURL))
		}
	}

	post := &model.Post{
		UserId:    userId,
		ChannelId: dmChannel.Id,
		Message:   buffer.String(),
		Type:      "github_todo",
	}

	if post, err := p.api.CreatePost(post); err != nil {
		return fmt.Errorf("Error creating the post")
	}

	return nil
}
