package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/google/go-github/v25/github"
)

const (
	SUBSCRIPTIONS_KEY       = "subscriptions"
	EXCLUDE_ORG_MEMBER_FLAG = "exclude-org-member"
)

type SubscriptionFlags struct {
	ExcludeOrgMembers bool
}

func (s *SubscriptionFlags) AddFlag(flag string) {
	switch flag {
	case EXCLUDE_ORG_MEMBER_FLAG:
		s.ExcludeOrgMembers = true
	}
}

func (s SubscriptionFlags) String() string {
	flags := []string{}

	if s.ExcludeOrgMembers {
		flag := "--" + EXCLUDE_ORG_MEMBER_FLAG
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
	return strings.Contains(s.Features, "pulls")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features, "issues")
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

func (p *Plugin) Subscribe(ctx context.Context, githubClient *github.Client, userId, owner, repo, channelID, features string, flags SubscriptionFlags) error {
	if owner == "" {
		return fmt.Errorf("Invalid repository")
	}

	if err := p.checkOrg(owner); err != nil {
		return err
	}

	var err error

	if repo == "" {
		var ghOrg *github.Organization
		ghOrg, _, err = githubClient.Organizations.Get(ctx, owner)
		if ghOrg == nil {
			var ghUser *github.User
			ghUser, _, err = githubClient.Users.Get(ctx, owner)
			if ghUser == nil {
				return fmt.Errorf("Unknown organization %s", owner)
			}
		}
	} else {
		var ghRepo *github.Repository
		ghRepo, _, err = githubClient.Repositories.Get(ctx, owner, repo)

		if ghRepo == nil {
			return fmt.Errorf("Unknown repository %s", fullNameFromOwnerAndRepo(owner, repo))
		}
	}

	if err != nil {
		mlog.Error(err.Error())
		return fmt.Errorf("Encountered an error subscribing to %s", fullNameFromOwnerAndRepo(owner, repo))
	}

	sub := &Subscription{
		ChannelID:  channelID,
		CreatorID:  userId,
		Features:   features,
		Repository: fullNameFromOwnerAndRepo(owner, repo),
		Flags:      flags,
	}

	if err := p.AddSubscription(fullNameFromOwnerAndRepo(owner, repo), sub); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) SubscribeOrg(ctx context.Context, githubClient *github.Client, userId, org, channelID, features string, flags SubscriptionFlags) error {
	if org == "" {
		return fmt.Errorf("Invalid organization")
	}

	return p.Subscribe(ctx, githubClient, userId, org, "", channelID, features, flags)
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*Subscription, error) {
	var filteredSubs []*Subscription
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, err
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
		return err
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
		return err
	}

	return nil
}

func (p *Plugin) GetSubscriptions() (*Subscriptions, error) {
	var subscriptions *Subscriptions

	value, err := p.API.KVGet(SUBSCRIPTIONS_KEY)
	if err != nil {
		return nil, err
	}

	if value == nil {
		subscriptions = &Subscriptions{Repositories: map[string][]*Subscription{}}
	} else {
		json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions)
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p.API.KVSet(SUBSCRIPTIONS_KEY, b)
	return nil
}

func (p *Plugin) GetSubscribedChannelsForRepository(repo *github.Repository) []*Subscription {
	name := repo.GetFullName()
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

	repo, _, _ = parseOwnerAndRepo(repo, config.EnterpriseBaseURL)

	if repo == "" {
		return fmt.Errorf("Invalid repository")
	}

	subs, err := p.GetSubscriptions()
	if err != nil {
		return err
	}

	repoSubs := subs.Repositories[repo]
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
		subs.Repositories[repo] = repoSubs
		if err := p.StoreSubscriptions(subs); err != nil {
			return err
		}
	}

	return nil
}
