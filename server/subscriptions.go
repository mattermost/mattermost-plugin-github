package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/mlog"

	"github.com/google/go-github/github"
)

const (
	SUBSCRIPTIONS_KEY = "subscriptions"
)

type Subscription struct {
	ChannelId string
	Features  string
}

type Subscriptions struct {
	Repositories map[string][]*Subscription
}

func parseOwnerAndRepo(full string) (string, string, string) {
	full = strings.TrimSuffix(strings.TrimSpace(strings.Replace(full, "https://github.com/", "", 1)), "/")
	splitStr := strings.Split(full, "/")
	if len(splitStr) != 2 {
		return "", "", ""
	}
	owner := splitStr[0]
	repo := splitStr[1]

	return fmt.Sprintf("%s/%s", owner, repo), owner, repo
}

func (p *Plugin) Subscribe(ctx context.Context, githubClient *github.Client, userId, ownerAndRepo, channelId, features string) error {
	_, owner, repo := parseOwnerAndRepo(ownerAndRepo)

	if owner == "" {
		return fmt.Errorf("Invalid repository")
	}

	if result, _, err := githubClient.Repositories.Get(ctx, owner, repo); result == nil || err != nil {
		if err != nil {
			mlog.Error(err.Error())
		}
		return fmt.Errorf("Unknown repository %s/%s", owner, repo)
	}

	sub := &Subscription{
		ChannelId: channelId,
		Features:  features,
	}

	if err := p.AddSubscription(fmt.Sprintf("%s/%s", owner, repo), sub); err != nil {
		return err
	}

	return nil
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
			if s.ChannelId == sub.ChannelId {
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

func (p *Plugin) GetSubscribedChannelsForRepository(repository string) []*Subscription {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil
	}
	return subs.Repositories[repository]
}

func (p *Plugin) Unsubscribe(channelId string, repo string) error {
	repo, _, _ = parseOwnerAndRepo(repo)

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
		if sub.ChannelId == channelId {
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
