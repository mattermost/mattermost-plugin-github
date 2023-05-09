package app

import (
	"context"
	"net/http"
	"net/url"
	"path"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	"golang.org/x/oauth2"
)

func GetGitHubClient(token oauth2.Token, config *config.Configuration) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(context.Background(), ts)

	return getGitHubClient(tc, config)
}

func getGitHubClient(authenticatedClient *http.Client, config *config.Configuration) (*github.Client, error) {
	if config.EnterpriseBaseURL == "" || config.EnterpriseUploadURL == "" {
		return github.NewClient(authenticatedClient), nil
	}

	baseURL, _ := url.Parse(config.EnterpriseBaseURL)
	baseURL.Path = path.Join(baseURL.Path, "api", "v3")

	uploadURL, _ := url.Parse(config.EnterpriseUploadURL)
	uploadURL.Path = path.Join(uploadURL.Path, "api", "v3")

	client, err := github.NewEnterpriseClient(baseURL.String(), uploadURL.String(), authenticatedClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (a *App) GithubConnectUser(ctx context.Context, info *GitHubUserInfo) *github.Client {
	tok := *info.Token
	return a.GithubConnectToken(tok)
}

func (a *App) GithubConnectToken(token oauth2.Token) *github.Client {
	config := a.config.GetConfiguration()

	client, err := GetGitHubClient(token, config)
	if err != nil {
		a.client.Log.Warn("Failed to create GitHub client", "error", err.Error())
		return nil
	}

	return client
}

func (a *App) GetGitHubUserInfo(userID string) (*GitHubUserInfo, *APIErrorResponse) {
	config := a.config.GetConfiguration()

	var userInfo *GitHubUserInfo
	err := a.client.KV.Get(userID+GithubTokenKey, &userInfo)
	if err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to get user info.", StatusCode: http.StatusInternalServerError}
	}
	if userInfo == nil {
		return nil, &APIErrorResponse{ID: ApiErrorIDNotConnected, Message: "Must connect user account to GitHub first.", StatusCode: http.StatusBadRequest}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		a.client.Log.Warn("Failed to decrypt access token", "error", err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	return userInfo, nil
}

func (a *App) getGitHubToUserIDMapping(githubUsername string) string {
	var data []byte
	_ = a.client.KV.Get(githubUsername+GithubUsernameKey, data)

	return string(data)
}

// GetGitHubToUsernameMapping maps a GitHub username to the corresponding Mattermost username, if any.
func (a *App) GetGitHubToUsernameMapping(githubUsername string) string {
	user, _ := a.client.User.Get(a.getGitHubToUserIDMapping(githubUsername))
	if user == nil {
		return ""
	}

	return user.Username
}
