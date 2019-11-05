package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/utils"

	"github.com/google/go-github/v25/github"
)

const (
	SUBSCRIPTIONS_KEY = "subscriptions"
)

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
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

func (p *Plugin) Subscribe(ctx context.Context, githubClient *github.Client, userId, owner, repo, channelID, features string) error {
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
	}

	if err := p.AddSubscription(fullNameFromOwnerAndRepo(owner, repo), sub); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) SubscribeOrg(ctx context.Context, githubClient *github.Client, userId, org, channelID, features string) error {
	if org == "" {
		return fmt.Errorf("Invalid organization")
	}

	return p.Subscribe(ctx, githubClient, userId, org, "", channelID, features)
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

	return filteredSubs, nil
}

// AddSubscription ...
func (p *Plugin) AddSubscription(repository string, subscription *Subscription) error {
	return p.Helpers.KVAtomicModify(context.Background(), SUBSCRIPTIONS_KEY, utils.NewTokenBucket(20*time.Millisecond, 3), func(initial []byte) ([]byte, error) {
		var subs *Subscriptions
		if initial == nil {
			subs = &Subscriptions{Repositories: map[string][]*Subscription{}}
		} else {
			json.NewDecoder(bytes.NewReader(initial)).Decode(&subs)
		}
		repoSubs := subs.Repositories[repository]
		if repoSubs == nil {
			repoSubs = []*Subscription{subscription}
		} else {
			exists := false
			for index, s := range repoSubs {
				if s.ChannelID == subscription.ChannelID {
					repoSubs[index] = subscription
					exists = true
					break
				}
			}
			if !exists {
				repoSubs = append(repoSubs, subscription)
			}
		}
		subs.Repositories[repository] = repoSubs
		return json.Marshal(repoSubs)
	})
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

func (p *Plugin) GetSubscribedChannelsForRepository(repo *github.Repository) []*Subscription {
	name := repo.GetFullName()
	org := strings.Split(name, "/")[0]
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil
	}

	// Add subcriptions for the specific repo
	subsForRepo := []*Subscription{}
	if subs.Repositories[name] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[name]...)
	}

	// Add subcriptions for the organization
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

// Unsubscribe ...
func (p *Plugin) Unsubscribe(channelID string, repository string) error {
	config := p.getConfiguration()
	repository, _, _ = parseOwnerAndRepo(repository, config.EnterpriseBaseURL)
	if repository == "" {
		return errors.New("invalid repository")
	}
	bucket := utils.NewTokenBucket(20*time.Millisecond, 3)
	ctx := context.Background()
	err := p.Helpers.KVAtomicModify(ctx, SUBSCRIPTIONS_KEY, bucket, func(initial []byte) ([]byte, error) {
		if initial == nil {
			return nil, errors.New("gh.plugin.err: nothing to be done")
		}
		var subs *Subscriptions
		json.NewDecoder(bytes.NewReader(initial)).Decode(&subs)
		repoSubs := subs.Repositories[repository]
		if repoSubs == nil {
			return nil, errors.New("gh.plugin.err: nothing to be done")
		}
		for index, sub := range repoSubs {
			if sub.ChannelID == channelID {
				repoSubs = append(repoSubs[:index], repoSubs[index+1:]...)
				subs.Repositories[repository] = repoSubs
				return json.Marshal(subs)
			}
		}
		return nil, errors.New("gh.plugin.err: nothing to be done")
	})

	if err.Error() != "gh.plugin.err: nothing to be done" {
		return err
	}

	return nil
}
