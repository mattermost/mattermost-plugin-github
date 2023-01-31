package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v48/github"
	"github.com/pkg/errors"
)

const (
	SubscriptionsKey     = "subscriptions"
	flagExcludeOrgMember = "exclude-org-member"
	flagRenderStyle      = "render-style"
	flagFeatures         = "features"
)

type SubscriptionFlags struct {
	ExcludeOrgMembers bool
	RenderStyle       string
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

	return strings.Join(flags, ",")
}

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
	Flags      SubscriptionFlags
	Repository string
}

type Subscriptions struct {
	Repositories map[string][]*Subscription
}

func (s *Subscription) Pulls() bool {
	return strings.Contains(s.Features, featurePulls)
}

func (s *Subscription) PullsMerged() bool {
	return strings.Contains(s.Features, "pulls_merged")
}

func (s *Subscription) IssueCreations() bool {
	return strings.Contains(s.Features, "issue_creations")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features, featureIssues)
}

func (s *Subscription) Pushes() bool {
	return strings.Contains(s.Features, "pushes")
}

func (s *Subscription) Creates() bool {
	return strings.Contains(s.Features, "creates")
}

func (s *Subscription) Deletes() bool {
	return strings.Contains(s.Features, "deletes")
}

func (s *Subscription) IssueComments() bool {
	return strings.Contains(s.Features, "issue_comments")
}

func (s *Subscription) PullReviews() bool {
	return strings.Contains(s.Features, "pull_reviews")
}

func (s *Subscription) Stars() bool {
	return strings.Contains(s.Features, featureStars)
}

func (s *Subscription) Label() string {
	if !strings.Contains(s.Features, "label:") {
		return ""
	}

	labelSplit := strings.Split(s.Features, "\"")
	if len(labelSplit) < 3 {
		return ""
	}

	return labelSplit[1]
}

func (s *Subscription) ExcludeOrgMembers() bool {
	return s.Flags.ExcludeOrgMembers
}

func (s *Subscription) RenderStyle() string {
	return s.Flags.RenderStyle
}

func (p *Plugin) Subscribe(ctx context.Context, githubClient *github.Client, userID, owner, repo, channelID, features string, flags SubscriptionFlags) error {
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
		p.API.LogWarn("Failed to get repository or org for subscribe action", "error", err.Error())
		return errors.Errorf("Encountered an error subscribing to %s", fullNameFromOwnerAndRepo(owner, repo))
	}

	sub := &Subscription{
		ChannelID:  channelID,
		CreatorID:  userID,
		Features:   features,
		Repository: fullNameFromOwnerAndRepo(owner, repo),
		Flags:      flags,
	}

	if err := p.AddSubscription(fullNameFromOwnerAndRepo(owner, repo), sub); err != nil {
		return errors.Wrap(err, "could not add subscription")
	}

	return nil
}

func (p *Plugin) SubscribeOrg(ctx context.Context, githubClient *github.Client, userID, org, channelID, features string, flags SubscriptionFlags) error {
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

	value, appErr := p.API.KVGet(SubscriptionsKey)
	if appErr != nil {
		return nil, errors.Wrap(appErr, "could not get subscriptions from KVStore")
	}

	if value == nil {
		return &Subscriptions{Repositories: map[string][]*Subscription{}}, nil
	}

	err := json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions)
	if err != nil {
		return nil, errors.Wrap(err, "could not properly decode subscriptions key")
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	b, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "error while converting subscriptions map to json")
	}

	if appErr := p.API.KVSet(SubscriptionsKey, b); appErr != nil {
		return errors.Wrap(appErr, "could not store subscriptions in KV store")
	}

	return nil
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
		subsToReturn = append(subsToReturn, sub)
	}

	return subsToReturn
}

func (p *Plugin) Unsubscribe(channelID string, repo string) error {
	config := p.getConfiguration()

	owner, repo := parseOwnerAndRepo(repo, config.getBaseURL())
	if owner == "" && repo == "" {
		return errors.New("invalid repository")
	}

	owner = strings.ToLower(owner)
	repo = strings.ToLower(repo)

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
