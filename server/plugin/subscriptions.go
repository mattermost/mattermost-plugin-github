package plugin

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
)

const (
	SubscriptionsKey      = "subscriptions"
	flagExcludeOrgMember  = "exclude-org-member"
	flagRenderStyle       = "render-style"
	flagFeatures          = "features"
	flagExcludeRepository = "exclude"
)

type SubscriptionFlags struct {
	ExcludeOrgMembers bool
	RenderStyle       string
	ExcludeRepository []string
}

func (s *SubscriptionFlags) AddFlag(flag string, value string) error {
	switch flag {
	case flagExcludeOrgMember:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		s.ExcludeOrgMembers = parsed
	case flagRenderStyle:
		s.RenderStyle = value
	case flagExcludeRepository:
		repos := strings.Split(value, ",")
		for i := range repos {
			repos[i] = strings.TrimSpace(repos[i])
		}
		s.ExcludeRepository = repos
	}

	return nil
}

func (s SubscriptionFlags) String() string {
	flags := []string{}

	if s.ExcludeOrgMembers {
		flag := "--" + flagExcludeOrgMember + " true"
		flags = append(flags, flag)
	}

	if s.RenderStyle != "" {
		flag := "--" + flagRenderStyle + " " + s.RenderStyle
		flags = append(flags, flag)
	}

	if len(s.ExcludeRepository) > 0 {
		flag := "--" + flagExcludeRepository + " " + strings.Join(s.ExcludeRepository, ",")
		flags = append(flags, flag)
	}

	return strings.Join(flags, ",")
}

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   Features
	Flags      SubscriptionFlags
	Repository string
	Projects   []Project
}

type Subscriptions struct {
	Repositories map[string][]*Subscription
}

func (s *Subscription) Pulls() bool {
	return strings.Contains(s.Features.String(), featurePulls)
}

func (s *Subscription) PullsCreated() bool {
	return strings.Contains(s.Features.String(), featurePullsCreated)
}

func (s *Subscription) PullsMerged() bool {
	return strings.Contains(s.Features.String(), "pulls_merged")
}

func (s *Subscription) IssueCreations() bool {
	return strings.Contains(s.Features.String(), "issue_creations")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features.String(), featureIssues)
}

func (s *Subscription) Pushes() bool {
	return strings.Contains(s.Features.String(), "pushes")
}

func (s *Subscription) Creates() bool {
	return strings.Contains(s.Features.String(), "creates")
}

func (s *Subscription) Deletes() bool {
	return strings.Contains(s.Features.String(), "deletes")
}

func (s *Subscription) IssueComments() bool {
	return strings.Contains(s.Features.String(), "issue_comments")
}

func (s *Subscription) PullReviews() bool {
	return strings.Contains(s.Features.String(), "pull_reviews")
}

func (s *Subscription) Stars() bool {
	return strings.Contains(s.Features.String(), featureStars)
}

func (s *Subscription) ProjectIssues() bool {
	return strings.Contains(s.Features.String(), featureProjectIssues)
}

func (s *Subscription) ProjectPulls() bool {
	return strings.Contains(s.Features.String(), featureProjectPulls)
}

func (s *Subscription) Label() string {
	var label string
	featuresSplit := strings.Split(s.Features.String(), ",")

	for _, feature := range featuresSplit {
		if strings.Contains(feature, "label:") {
			labelSplit := strings.Split(feature, "\"")
			label = labelSplit[1]
			break
		}
	}

	return label
}

func (s *Subscription) ProjectTitles() []string {
	var projects []string
	featuresSplit := strings.Split(s.Features.String(), ",")

	for _, feature := range featuresSplit {
		if strings.Contains(feature, "project:") {
			projectSplit := strings.Split(feature, "\"")
			if len(projectSplit[1]) == 0 {
				continue
			}
			projects = append(projects, projectSplit[1])
		}
	}

	return projects
}

func (s *Subscription) ExcludeOrgMembers() bool {
	return s.Flags.ExcludeOrgMembers
}

func (s *Subscription) RenderStyle() string {
	return s.Flags.RenderStyle
}

func (s *Subscription) excludedRepoForSub(repo *github.Repository) bool {
	for _, repository := range s.Flags.ExcludeRepository {
		if repository == repo.GetFullName() {
			return true
		}
	}
	return false
}

func (p *Plugin) Subscribe(ctx context.Context, githubClient *github.Client, userID, owner, repo, channelID string, features Features, flags SubscriptionFlags) error {
	if owner == "" {
		return errors.Errorf("invalid repository")
	}

	owner = strings.ToLower(owner)
	repo = strings.ToLower(repo)

	if err := p.checkOrg(owner); err != nil {
		return errors.Wrap(err, "organization not supported")
	}

	if flags.ExcludeOrgMembers && !p.isOrganizationLocked() {
		return errors.New("Unable to set --exclude-org-member flag. The GitHub plugin is not locked to a single organization.")
	}

	var err error

	if repo == "" {
		var ghOrg *github.Organization
		ghOrg, _, err = githubClient.Organizations.Get(ctx, owner)
		if ghOrg == nil {
			var ghUser *github.User
			ghUser, _, err = githubClient.Users.Get(ctx, owner)
			if ghUser == nil {
				return errors.Errorf("Unknown organization %s", owner)
			}
		}
	} else {
		var ghRepo *github.Repository
		ghRepo, _, err = githubClient.Repositories.Get(ctx, owner, repo)

		if ghRepo == nil {
			return errors.Errorf("unknown repository %s", fullNameFromOwnerAndRepo(owner, repo))
		}
	}

	if err != nil {
		p.client.Log.Warn("Failed to get repository or org for subscribe action", "error", err.Error())
		return errors.Errorf("Encountered an error subscribing to %s", fullNameFromOwnerAndRepo(owner, repo))
	}

	sub := &Subscription{
		ChannelID:  channelID,
		CreatorID:  userID,
		Features:   features,
		Repository: fullNameFromOwnerAndRepo(owner, repo),
		Flags:      flags,
	}

	subscribedProjects := sub.ProjectTitles()
	if len(subscribedProjects) > 0 {
		userInfo, apiErr := p.getGitHubUserInfo(userID)
		if apiErr != nil {
			return errors.Errorf("Failed to get GitHub user info for userID:%s", userID)
		}

		graphqlClient := p.graphQLConnect(userInfo)
		if graphqlClient == nil {
			return errors.Errorf("Failed to get graphql client")
		}

		// currently only queries for organizational projects not user repo projects
		// org level projects and repo level projects are both included, both can be used in issue/pr assignment
		projectData, err := graphqlClient.GetProjectsV2Data(ctx, owner)
		if err != nil {
			p.client.Log.Warn("graphql organization projects query failed", "error", err.Error())
			return errors.Errorf("Failed to get project data")
		}

		if projectData != nil {
			var matchedProjects []Project
			for _, project := range projectData {
				for _, subscribedProject := range subscribedProjects {
					if subscribedProject == *project.Title {
						matchedProjects = append(matchedProjects, Project{NodeID: *project.NodeID, Title: *project.Title})
					}
				}
			}
			if len(matchedProjects) == 0 {
				return errors.Errorf("No project(s) were found matching %s", strings.Join(subscribedProjects, ", "))
			}
			sub.Projects = matchedProjects
		}
	}

	if err := p.AddSubscription(fullNameFromOwnerAndRepo(owner, repo), sub); err != nil {
		return errors.Wrap(err, "could not add subscription")
	}

	return nil
}

func (p *Plugin) SubscribeOrg(ctx context.Context, githubClient *github.Client, userID, org, channelID string, features Features, flags SubscriptionFlags) error {
	if org == "" {
		return errors.New("invalid organization")
	}

	return p.Subscribe(ctx, githubClient, userID, org, "", channelID, features, flags)
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*Subscription, error) {
	var filteredSubs []*Subscription
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, errors.Wrap(err, "could not get subscriptions")
	}

	for repo, v := range subs.Repositories {
		for _, s := range v {
			if s.ChannelID == channelID {
				// this is needed to be backwards compatible
				if len(s.Repository) == 0 {
					s.Repository = repo
				}
				filteredSubs = append(filteredSubs, s)
			}
		}
	}

	sort.Slice(filteredSubs, func(i, j int) bool {
		return filteredSubs[i].Repository < filteredSubs[j].Repository
	})

	return filteredSubs, nil
}

func (p *Plugin) AddSubscription(repo string, sub *Subscription) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}

	repoSubs := subs.Repositories[repo]
	if repoSubs == nil {
		repoSubs = []*Subscription{sub}
	} else {
		exists := false
		for index, s := range repoSubs {
			if s.ChannelID == sub.ChannelID {
				repoSubs[index] = sub
				exists = true
				break
			}
		}

		if !exists {
			repoSubs = append(repoSubs, sub)
		}
	}

	subs.Repositories[repo] = repoSubs

	err = p.StoreSubscriptions(subs)
	if err != nil {
		return errors.Wrap(err, "could not store subscriptions")
	}

	return nil
}

func (p *Plugin) GetSubscriptions() (*Subscriptions, error) {
	var subscriptions *Subscriptions

	err := p.store.Get(SubscriptionsKey, &subscriptions)
	if err != nil {
		return nil, errors.Wrap(err, "could not get subscriptions from KVStore")
	}

	// No subscriptions stored.
	if subscriptions == nil {
		return &Subscriptions{Repositories: map[string][]*Subscription{}}, nil
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	if _, err := p.store.Set(SubscriptionsKey, s); err != nil {
		return errors.Wrap(err, "could not store subscriptions in KV store")
	}

	return nil
}

func (p *Plugin) GetSubscribedChannelsForOrg(org string) []*Subscription {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil
	}

	subsForOrg := []*Subscription{}
	orgKey := org + "/"

	if subs.Repositories[orgKey] != nil {
		subsForOrg = append(subsForOrg, subs.Repositories[orgKey]...)
	}

	if len(subsForOrg) == 0 {
		return nil
	}

	return subsForOrg
}

func (p *Plugin) GetSubscribedChannelsForRepository(repo *github.Repository) []*Subscription {
	name := repo.GetFullName()
	name = strings.ToLower(name)
	org := strings.Split(name, "/")[0]
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil
	}

	// Add subscriptions for the specific repo
	subsForRepo := []*Subscription{}
	if subs.Repositories[name] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[name]...)
	}

	// Add subscriptions for the organization
	orgKey := fullNameFromOwnerAndRepo(org, "")
	if subs.Repositories[orgKey] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[orgKey]...)
	}

	if len(subsForRepo) == 0 {
		return nil
	}

	subsToReturn := []*Subscription{}

	for _, sub := range subsForRepo {
		if repo.GetPrivate() && !p.permissionToRepo(sub.CreatorID, name) {
			continue
		}
		if sub.excludedRepoForSub(repo) {
			continue
		}
		subsToReturn = append(subsToReturn, sub)
	}

	return subsToReturn
}

func (p *Plugin) Unsubscribe(channelID, repo, owner string) error {
	repoWithOwner := fmt.Sprintf("%s/%s", owner, repo)

	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}

	repoSubs := subs.Repositories[repoWithOwner]
	if repoSubs == nil {
		return nil
	}

	removed := false
	for index, sub := range repoSubs {
		if sub.ChannelID == channelID {
			repoSubs = append(repoSubs[:index], repoSubs[index+1:]...)
			removed = true
			break
		}
	}

	if removed {
		subs.Repositories[repoWithOwner] = repoSubs
		if err := p.StoreSubscriptions(subs); err != nil {
			return errors.Wrap(err, "could not store subscriptions")
		}
	}

	return nil
}
