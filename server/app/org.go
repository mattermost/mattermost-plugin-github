package app

import (
	"context"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

func (a *App) CheckOrg(org string) error {
	config := a.config.GetConfiguration()

	configOrg := strings.TrimSpace(config.GitHubOrg)
	if configOrg != "" && configOrg != org && strings.ToLower(configOrg) != org {
		return errors.Errorf("only repositories in the %v organization are supported", configOrg)
	}

	return nil
}

func (a *App) isUserOrganizationMember(githubClient *github.Client, user *github.User, organization string) bool {
	if organization == "" {
		return false
	}

	isMember, _, err := githubClient.Organizations.IsMember(context.Background(), organization, *user.Login)
	if err != nil {
		a.client.Log.Warn("Failled to check if user is org member", "GitHub username", *user.Login, "error", err.Error())
		return false
	}

	return isMember
}

func (a *App) IsOrganizationLocked() bool {
	config := a.config.GetConfiguration()
	configOrg := strings.TrimSpace(config.GitHubOrg)

	return configOrg != ""
}
