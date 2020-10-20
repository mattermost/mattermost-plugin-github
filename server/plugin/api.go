package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v31/github"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	apiErrorIDNotConnected = "not_connected"
	// TokenTTL is the OAuth token expiry duration in seconds
	TokenTTL = 10 * 60
)

type OAuthState struct {
	UserID         string `json:"user_id"`
	Token          string `json:"token"`
	PrivateAllowed bool   `json:"private_allowed"`
}

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func (e *APIErrorResponse) Error() string {
	return e.Message
}

type PRDetails struct {
	URL                string                      `json:"url"`
	Number             int                         `json:"number"`
	Status             string                      `json:"status"`
	Mergeable          bool                        `json:"mergeable"`
	RequestedReviewers []*string                   `json:"requestedReviewers"`
	Reviews            []*github.PullRequestReview `json:"reviews"`
}

// HTTPHandlerFuncWithUser is http.HandleFunc but userID is already exported
type HTTPHandlerFuncWithUser func(w http.ResponseWriter, r *http.Request, userID string)

// ResponseType indicates type of response returned by api
type ResponseType string

const (
	// ResponseTypeJSON indicates that response type is json
	ResponseTypeJSON ResponseType = "JSON_RESPONSE"
	// ResponseTypePlain indicates that response type is text plain
	ResponseTypePlain ResponseType = "TEXT_RESPONSE"
)

func (p *Plugin) writeJSON(w http.ResponseWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		p.API.LogWarn("Failed to marshal JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		p.API.LogWarn("Failed to write JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) writeAPIError(w http.ResponseWriter, apiErr *APIErrorResponse) {
	b, err := json.Marshal(apiErr)
	if err != nil {
		p.API.LogWarn("Failed to marshal API error", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(apiErr.StatusCode)

	_, err = w.Write(b)
	if err != nil {
		p.API.LogWarn("Failed to write JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) initializeAPI() {
	p.router = mux.NewRouter()
	p.router.Use(p.withRecovery)

	oauthRouter := p.router.PathPrefix("/oauth").Subrouter()
	apiRouter := p.router.PathPrefix("/api/v1").Subrouter()

	p.router.HandleFunc("/webhook", p.handleWebhook).Methods(http.MethodPost)

	oauthRouter.HandleFunc("/connect", p.extractUserMiddleWare(p.connectUserToGitHub, ResponseTypePlain)).Methods(http.MethodGet)
	oauthRouter.HandleFunc("/complete", p.extractUserMiddleWare(p.completeConnectUserToGitHub, ResponseTypePlain)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/connected", p.getConnected).Methods(http.MethodGet)
	apiRouter.HandleFunc("/todo", p.extractUserMiddleWare(p.postToDo, ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/reviews", p.extractUserMiddleWare(p.getReviews, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/yourprs", p.extractUserMiddleWare(p.getYourPrs, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/prsdetails", p.extractUserMiddleWare(p.getPrsDetails, ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/searchissues", p.extractUserMiddleWare(p.searchIssues, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/yourassignments", p.extractUserMiddleWare(p.getYourAssignments, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/createissue", p.extractUserMiddleWare(p.createIssue, ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/createissuecomment", p.extractUserMiddleWare(p.createIssueComment, ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/mentions", p.extractUserMiddleWare(p.getMentions, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/unreads", p.extractUserMiddleWare(p.getUnreads, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/labels", p.extractUserMiddleWare(p.getLabels, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/milestones", p.extractUserMiddleWare(p.getMilestones, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/assignees", p.extractUserMiddleWare(p.getAssignees, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/repositories", p.extractUserMiddleWare(p.getRepositories, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/settings", p.extractUserMiddleWare(p.updateSettings, ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/user", p.extractUserMiddleWare(p.getGitHubUser, ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/issue", p.extractUserMiddleWare(p.getIssueByNumber, ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/pr", p.extractUserMiddleWare(p.getPrByNumber, ResponseTypePlain)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/config", checkPluginRequest(p.getConfig)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/token", checkPluginRequest(p.getToken)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/settings", p.getSettings).Methods(http.MethodGet)
}

func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.API.LogError("Recovered from a panic",
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) extractUserMiddleWare(handler HTTPHandlerFuncWithUser, responseType ResponseType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			switch responseType {
			case ResponseTypeJSON:
				p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
			case ResponseTypePlain:
				http.Error(w, "Not authorized", http.StatusUnauthorized)
			default:
				p.API.LogError("Unknown ResponseType detected")
			}
			return
		}

		handler(w, r, userID)
	}
}

func checkPluginRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// All other plugins are allowed
		pluginID := r.Header.Get("Mattermost-Plugin-ID")
		if pluginID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

	r.Header.Add("Mattermost-Plugin-ID", c.SourcePluginId)
	w.Header().Set("Content-Type", "application/json")

	p.router.ServeHTTP(w, r)
}

func (p *Plugin) connectUserToGitHub(w http.ResponseWriter, r *http.Request, userID string) {
	privateAllowed := false
	pValBool, _ := strconv.ParseBool(r.URL.Query().Get("private"))
	if pValBool {
		privateAllowed = true
	}

	conf := p.getOAuthConfig(privateAllowed)

	state := OAuthState{
		UserID:         userID,
		Token:          model.NewId()[:15],
		PrivateAllowed: privateAllowed,
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		http.Error(w, "json marshal failed", http.StatusInternalServerError)
		return
	}

	appErr := p.API.KVSetWithExpiry(state.Token, stateBytes, TokenTTL)
	if appErr != nil {
		http.Error(w, "error setting stored state", http.StatusBadRequest)
		return
	}

	url := conf.AuthCodeURL(state.Token, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitHub(w http.ResponseWriter, r *http.Request, authedUserID string) {
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	stateToken := r.URL.Query().Get("state")

	storedState, appErr := p.API.KVGet(stateToken)
	if appErr != nil {
		p.API.LogWarn("Failed to get state token", "error", appErr.Error())
		http.Error(w, "missing stored state", http.StatusBadRequest)
		return
	}

	appErr = p.API.KVDelete(stateToken)
	if appErr != nil {
		p.API.LogWarn("Failed to delete state token", "error", appErr.Error())
		http.Error(w, "error deleting stored state", http.StatusBadRequest)
		return
	}

	var state OAuthState
	if err := json.Unmarshal(storedState, &state); err != nil {
		http.Error(w, "json unmarshal failed", http.StatusBadRequest)
		return
	}

	if state.Token != stateToken {
		http.Error(w, "invalid state token", http.StatusBadRequest)
		return
	}

	if state.UserID != authedUserID {
		http.Error(w, "Not authorized, incorrect user", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	conf := p.getOAuthConfig(state.PrivateAllowed)

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		p.API.LogWarn("Failed to exchange oauth code into token", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubClient := p.githubConnect(*tok)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		p.API.LogWarn("Failed to get authenticated GitHub user", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo := &GitHubUserInfo{
		UserID:         state.UserID,
		Token:          tok,
		GitHubUsername: gitUser.GetLogin(),
		LastToDoPostAt: model.GetMillis(),
		Settings: &UserSettings{
			SidebarButtons: settingButtonsTeam,
			DailyReminder:  true,
			Notifications:  true,
		},
		AllowedPrivateRepos: state.PrivateAllowed,
	}

	if err = p.storeGitHubUserInfo(userInfo); err != nil {
		p.API.LogWarn("Failed to store GitHub user info", "error", err.Error())
		http.Error(w, "Unable to connect user to GitHub", http.StatusInternalServerError)
		return
	}

	if err = p.storeGitHubToUserIDMapping(gitUser.GetLogin(), state.UserID); err != nil {
		p.API.LogWarn("Failed to store GitHub user info mapping", "error", err.Error())
	}

	commandHelp, err := renderTemplate("helpText", p.getConfiguration())
	if err != nil {
		p.API.LogWarn("Failed to render help template", "error", err.Error())
	}

	// Post intro post
	message := fmt.Sprintf("#### Welcome to the Mattermost GitHub Plugin!\n"+
		"You've connected your Mattermost account to [%s](%s) on GitHub. Read about the features of this plugin below:\n\n"+
		"##### Daily Reminders\n"+
		"The first time you log in each day, you will get a post right here letting you know what messages you need to read and what pull requests are awaiting your review.\n"+
		"Turn off reminders with `/github settings reminders off`.\n\n"+
		"##### Notifications\n"+
		"When someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you, you'll get a post here about it.\n"+
		"Turn off notifications with `/github settings notifications off`.\n\n"+
		"##### Sidebar Buttons\n"+
		"Check out the buttons in the left-hand sidebar of Mattermost.\n"+
		"* The first button tells you how many pull requests you have submitted.\n"+
		"* The second shows the number of PR that are awaiting your review.\n"+
		"* The third shows the number of PR and issues your are assiged to.\n"+
		"* The fourth tracks the number of unread messages you have.\n"+
		"* The fifth will refresh the numbers.\n\n"+
		"Click on them!\n\n"+
		"##### Slash Commands\n"+
		commandHelp, gitUser.GetLogin(), gitUser.GetHTMLURL())

	p.CreateBotDMPost(state.UserID, message, "custom_git_welcome")

	config := p.getConfiguration()

	p.API.PublishWebSocketEvent(
		wsEventConnect,
		map[string]interface{}{
			"connected":           true,
			"github_username":     userInfo.GitHubUsername,
			"github_client_id":    config.GitHubOAuthClientID,
			"enterprise_base_url": config.EnterpriseBaseURL,
			"organization":        config.GitHubOrg,
		},
		&model.WebsocketBroadcast{UserId: state.UserID},
	)

	html := `
			<!DOCTYPE html>
			<html>
			<head>
			<script>
			window.close();
			</script>
			</head>
			<body>
			<p>Completed connecting to GitHub. Please close this window.</p>
			</body>
			</html>
			`

	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write([]byte(html))
	if err != nil {
		p.API.LogWarn("Failed to write HTML response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) getGitHubUser(w http.ResponseWriter, r *http.Request, _ string) {
	type GitHubUserRequest struct {
		UserID string `json:"user_id"`
	}

	req := &GitHubUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.API.LogWarn("Error decoding GitHubUserRequest from JSON body", "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.UserID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object with a non-blank user_id field.", StatusCode: http.StatusBadRequest})
		return
	}

	userInfo, apiErr := p.getGitHubUserInfo(req.UserID)
	if apiErr != nil {
		if apiErr.ID == apiErrorIDNotConnected {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		} else {
			p.writeAPIError(w, apiErr)
		}
		return
	}

	if userInfo == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		return
	}

	type GitHubUserResponse struct {
		Username string `json:"username"`
	}

	resp := &GitHubUserResponse{Username: userInfo.GitHubUsername}
	p.writeJSON(w, resp)
}

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	type ConnectedResponse struct {
		Connected         bool          `json:"connected"`
		GitHubUsername    string        `json:"github_username"`
		GitHubClientID    string        `json:"github_client_id"`
		EnterpriseBaseURL string        `json:"enterprise_base_url,omitempty"`
		Organization      string        `json:"organization"`
		Settings          *UserSettings `json:"settings"`
	}

	resp := &ConnectedResponse{
		Connected:         false,
		EnterpriseBaseURL: config.EnterpriseBaseURL,
		Organization:      config.GitHubOrg,
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		p.writeJSON(w, resp)
		return
	}

	info, _ := p.getGitHubUserInfo(userID)
	if info == nil || info.Token == nil {
		p.writeJSON(w, resp)
		return
	}

	resp.Connected = true
	resp.GitHubUsername = info.GitHubUsername
	resp.GitHubClientID = config.GitHubOAuthClientID
	resp.Settings = info.Settings

	if info.Settings.DailyReminder && r.URL.Query().Get("reminder") == "true" {
		lastPostAt := info.LastToDoPostAt

		var timezone *time.Location
		offset, _ := strconv.Atoi(r.Header.Get("X-Timezone-Offset"))
		timezone = time.FixedZone("local", -60*offset)

		// Post to do message if it's the next day and been more than an hour since the last post
		now := model.GetMillis()
		nt := time.Unix(now/1000, 0).In(timezone)
		lt := time.Unix(lastPostAt/1000, 0).In(timezone)
		if nt.Sub(lt).Hours() >= 1 && (nt.Day() != lt.Day() || nt.Month() != lt.Month() || nt.Year() != lt.Year()) {
			if p.HasUnreads(info) {
				p.PostToDo(info)
				info.LastToDoPostAt = now
				if err := p.storeGitHubUserInfo(info); err != nil {
					p.API.LogWarn("Failed to store github info for new user", "userID", userID, "error", err.Error())
				}
			}
		}
	}

	privateRepoStoreKey := info.UserID + githubPrivateRepoKey
	if config.EnablePrivateRepo && !info.AllowedPrivateRepos {
		val, err := p.API.KVGet(privateRepoStoreKey)
		if err != nil {
			p.API.LogWarn("Unable to get private repo key value", "error", err.Error())
			return
		}

		// Inform the user once that private repositories enabled
		if val == nil {
			p.CreateBotDMPost(info.UserID, "Private repositories have been enabled for this plugin. To be able to use them you must disconnect and reconnect your GitHub account. To reconnect your account, use the following slash commands: `/github disconnect` followed by `/github connect private`.", "")

			err := p.API.KVSet(privateRepoStoreKey, []byte("1"))
			if err != nil {
				p.API.LogWarn("Unable to set private repo key value", "error", err.Error())
			}
		}
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) getMentions(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)
	username := info.GitHubUsername
	query := getMentionSearchQuery(username, config.GitHubOrg)

	result, _, err := githubClient.Search.Issues(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for issues", "query", query, "error", err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getUnreads(w http.ResponseWriter, r *http.Request, userID string) {
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)

	notifications, _, err := githubClient.Activity.ListNotifications(context.Background(), &github.NotificationListOptions{})
	if err != nil {
		p.API.LogWarn("Failed to list notifications", "error", err.Error())
		return
	}

	type filteredNotification struct {
		github.Notification

		HTMLUrl string `json:"html_url"`
	}

	filteredNotifications := []*filteredNotification{}
	for _, n := range notifications {
		if n.GetReason() == notificationReasonSubscribed {
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		issueURL := n.GetSubject().GetURL()
		issueNumIndex := strings.LastIndex(issueURL, "/")
		issueNum := issueURL[issueNumIndex+1:]
		subjectURL := n.GetSubject().GetURL()
		if n.GetSubject().GetLatestCommentURL() != "" {
			subjectURL = n.GetSubject().GetLatestCommentURL()
		}

		filteredNotifications = append(filteredNotifications, &filteredNotification{
			Notification: *n,
			HTMLUrl:      fixGithubNotificationSubjectURL(subjectURL, issueNum),
		})
	}

	p.writeJSON(w, filteredNotifications)
}

func (p *Plugin)  getSettings(w http.ResponseWriter, _ *http.Request) {
    resp := struct {
		LeftSidebarEnabled bool `json:"left_sidebar_enabled"`
	}{
		LeftSidebarEnabled: p.getConfiguration().EnableLeftSidebar,
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) getReviews(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)
	username := info.GitHubUsername

	query := getReviewSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for review", "query", query, "error", err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getYourPrs(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)
	username := info.GitHubUsername

	query := getYourPrsSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for PRs", "query", query, "error", err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getPrsDetails(w http.ResponseWriter, r *http.Request, userID string) {
	info, err := p.getGitHubUserInfo(userID)
	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	githubClient := p.githubConnect(*info.Token)

	var prList []*PRDetails
	if err := json.NewDecoder(r.Body).Decode(&prList); err != nil {
		p.API.LogWarn("Error decoding PRDetails JSON body", "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	prDetails := make([]*PRDetails, len(prList))
	ctx := context.Background()
	var wg sync.WaitGroup
	for i, pr := range prList {
		i := i
		pr := pr
		wg.Add(1)
		go func() {
			defer wg.Done()
			prDetail := p.fetchPRDetails(ctx, githubClient, pr.URL, pr.Number)
			prDetails[i] = prDetail
		}()
	}

	wg.Wait()

	p.writeJSON(w, prDetails)
}

func (p *Plugin) fetchPRDetails(ctx context.Context, client *github.Client, prURL string, prNumber int) *PRDetails {
	var status string
	var mergeable bool
	// Initialize to a non-nil slice to simplify JSON handling semantics
	requestedReviewers := []*string{}
	var reviewsList []*github.PullRequestReview = []*github.PullRequestReview{}

	repoOwner, repoName := getRepoOwnerAndNameFromURL(prURL)

	var wg sync.WaitGroup

	// Fetch reviews
	wg.Add(1)
	go func() {
		defer wg.Done()
		fetchedReviews, err := fetchReviews(ctx, client, repoOwner, repoName, prNumber)
		if err != nil {
			p.API.LogWarn("Failed to fetch reviews for PR details", "error", err.Error())
			return
		}
		reviewsList = fetchedReviews
	}()

	// Fetch reviewers and status
	wg.Add(1)
	go func() {
		defer wg.Done()
		prInfo, _, err := client.PullRequests.Get(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			p.API.LogWarn("Failed to fetch PR for PR details", "error", err.Error())
			return
		}

		mergeable = prInfo.GetMergeable()

		for _, v := range prInfo.RequestedReviewers {
			requestedReviewers = append(requestedReviewers, v.Login)
		}
		statuses, _, err := client.Repositories.GetCombinedStatus(ctx, repoOwner, repoName, prInfo.GetHead().GetSHA(), nil)
		if err != nil {
			p.API.LogWarn("Failed to fetch combined status", "error", err.Error())
			return
		}
		status = *statuses.State
	}()

	wg.Wait()
	return &PRDetails{
		URL:                prURL,
		Number:             prNumber,
		Status:             status,
		Mergeable:          mergeable,
		RequestedReviewers: requestedReviewers,
		Reviews:            reviewsList,
	}
}

func fetchReviews(ctx context.Context, client *github.Client, repoOwner string, repoName string, number int) ([]*github.PullRequestReview, error) {
	reviewsList, _, err := client.PullRequests.ListReviews(ctx, repoOwner, repoName, number, nil)

	if err != nil {
		return []*github.PullRequestReview{}, errors.Wrap(err, "could not list reviews")
	}

	return reviewsList, nil
}

func getRepoOwnerAndNameFromURL(url string) (string, string) {
	splitted := strings.Split(url, "/")
	return splitted[len(splitted)-2], splitted[len(splitted)-1]
}

func (p *Plugin) searchIssues(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)

	searchTerm := r.FormValue("term")
	query := getIssuesSearchQuery(config.GitHubOrg, searchTerm)
	result, _, err := githubClient.Search.Issues(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for issues", "query", query, "error", err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getPermaLink(postID string) string {
	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	return fmt.Sprintf("%v/_redirect/pl/%v", siteURL, postID)
}

func getFailReason(code int, repo string, username string) string {
	cause := ""
	switch code {
	case http.StatusInternalServerError:
		cause = "Internal server error"
	case http.StatusBadRequest:
		cause = "Bad request"
	case http.StatusNotFound:
		cause = fmt.Sprintf("Sorry, either you don't have access to the repo %s with the user %s or it is no longer available", repo, username)
	case http.StatusUnauthorized:
		cause = fmt.Sprintf("Sorry, your user %s is unauthorized to do this action", username)
	case http.StatusForbidden:
		cause = fmt.Sprintf("Sorry, you don't have enough permissions to comment in the repo %s with the user %s", repo, username)
	default:
		cause = fmt.Sprintf("Unknown status code %d", code)
	}
	return cause
}

func (p *Plugin) createIssueComment(w http.ResponseWriter, r *http.Request, userID string) {
	type CreateIssueCommentRequest struct {
		PostID  string `json:"post_id"`
		Owner   string `json:"owner"`
		Repo    string `json:"repo"`
		Number  int    `json:"number"`
		Comment string `json:"comment"`
	}

	req := &CreateIssueCommentRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.API.LogWarn("Error decoding CreateIssueCommentRequest JSON body", "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.PostID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid post id", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Owner == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repo owner.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Repo == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repo.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Number == 0 {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid issue number.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Comment == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid non empty comment.", StatusCode: http.StatusBadRequest})
		return
	}

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load post " + req.PostID, StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load post " + req.PostID + ": not found", StatusCode: http.StatusNotFound})
		return
	}

	commentUsername, err := p.getUsername(post.UserId)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
		return
	}

	currentUsername := info.GitHubUsername
	permalink := p.getPermaLink(req.PostID)
	permalinkMessage := fmt.Sprintf("*@%s attached a* [message](%s) *from %s*\n\n", currentUsername, permalink, commentUsername)

	req.Comment = permalinkMessage + req.Comment
	comment := &github.IssueComment{
		Body: &req.Comment,
	}

	result, rawResponse, err := githubClient.Issues.CreateComment(context.Background(), req.Owner, req.Repo, req.Number, comment)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to create an issue comment: " + getFailReason(rawResponse.StatusCode, req.Repo, currentUsername), StatusCode: rawResponse.StatusCode})
		return
	}

	rootID := req.PostID
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	permalinkReplyMessage := fmt.Sprintf("[Message](%v) attached to GitHub issue [#%v](%v)", permalink, req.Number, result.GetHTMLURL())
	reply := &model.Post{
		Message:   permalinkReplyMessage,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		ParentId:  rootID,
		UserId:    userID,
	}

	_, appErr = p.API.CreatePost(reply)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to create notification post " + req.PostID, StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeJSON(w, result)
}

func (p *Plugin) getYourAssignments(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}
	githubClient := p.githubConnect(*info.Token)

	username := info.GitHubUsername
	query := getYourAssigneeSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for assignments", "query", query, "error", err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) postToDo(w http.ResponseWriter, r *http.Request, userID string) {
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	githubClient := p.githubConnect(*info.Token)
	username := info.GitHubUsername

	text, err := p.GetToDo(context.Background(), username, githubClient)
	if err != nil {
		p.API.LogWarn("Failed to get Todos", "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	p.CreateBotDMPost(userID, text, "custom_git_todo")

	resp := struct {
		Status string
	}{"OK"}

	p.writeJSON(w, resp)
}

func (p *Plugin) updateSettings(w http.ResponseWriter, r *http.Request, userID string) {
	var settings *UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		p.API.LogWarn("Error decoding settings from JSON body", "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if settings == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, err := p.getGitHubUserInfo(userID)
	if err != nil {
		p.API.LogWarn("Failed to get GitHub user info", "error", err.Error())
		p.writeAPIError(w, err)
		return
	}

	info.Settings = settings

	if err := p.storeGitHubUserInfo(info); err != nil {
		p.API.LogWarn("Failed to store GitHub user info", "error", err.Error())
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
		return
	}

	p.writeJSON(w, info.Settings)
}

func (p *Plugin) getIssueByNumber(w http.ResponseWriter, r *http.Request, userID string) {
	owner := r.FormValue("owner")
	repo := r.FormValue("repo")
	number := r.FormValue("number")
	numberInt, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
		return
	}

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}
	githubClient := p.githubConnect(*info.Token)

	result, _, err := githubClient.Issues.Get(context.Background(), owner, repo, numberInt)
	if err != nil {
		// If the issue is not found, it's probably behind a private repo.
		// Return an empty repose in this case.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) && gerr.Response.StatusCode == http.StatusNotFound {
			p.API.LogDebug("Issue not found", "owner", owner, "repo", repo, "number", numberInt)
			p.writeJSON(w, nil)
			return
		}

		p.API.LogDebug("Could not get issue", "owner", owner, "repo", repo, "number", numberInt, "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{Message: "Could not get issue", StatusCode: http.StatusInternalServerError})
		return
	}
	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getPrByNumber(w http.ResponseWriter, r *http.Request, userID string) {
	owner := r.FormValue("owner")
	repo := r.FormValue("repo")
	number := r.FormValue("number")

	numberInt, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
		return
	}

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}
	githubClient := p.githubConnect(*info.Token)

	result, _, err := githubClient.PullRequests.Get(context.Background(), owner, repo, numberInt)
	if err != nil {
		// If the pull request is not found, it's probably behind a private repo.
		// Return an empty repose in this case.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) && gerr.Response.StatusCode == http.StatusNotFound {
			p.API.LogDebug("Pull request not found", "owner", owner, "repo", repo, "number", numberInt)
			p.writeJSON(w, nil)
			return
		}

		p.API.LogDebug("Could not get pull request", "owner", owner, "repo", repo, "number", numberInt, "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{Message: "Could not get pull request", StatusCode: http.StatusInternalServerError})
		return
	}
	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getLabels(w http.ResponseWriter, r *http.Request, userID string) {
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnect(*info.Token)
	var allLabels []*github.Label
	opt := github.ListOptions{PerPage: 50}

	for {
		labels, resp, err := githubClient.Issues.ListLabels(context.Background(), owner, repo, &opt)
		if err != nil {
			p.API.LogWarn("Failed to list labels", "error", err.Error())
			p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch labels", StatusCode: http.StatusInternalServerError})
			return
		}
		allLabels = append(allLabels, labels...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	p.writeJSON(w, allLabels)
}

func (p *Plugin) getAssignees(w http.ResponseWriter, r *http.Request, userID string) {
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnect(*info.Token)
	var allAssignees []*github.User
	opt := github.ListOptions{PerPage: 50}

	for {
		assignees, resp, err := githubClient.Issues.ListAssignees(context.Background(), owner, repo, &opt)
		if err != nil {
			p.API.LogWarn("Failed to list assignees", "error", err.Error())
			p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch assignees", StatusCode: http.StatusInternalServerError})
			return
		}
		allAssignees = append(allAssignees, assignees...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	p.writeJSON(w, allAssignees)
}

func (p *Plugin) getMilestones(w http.ResponseWriter, r *http.Request, userID string) {
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnect(*info.Token)
	var allMilestones []*github.Milestone
	opt := github.ListOptions{PerPage: 50}

	for {
		milestones, resp, err := githubClient.Issues.ListMilestones(context.Background(), owner, repo, &github.MilestoneListOptions{ListOptions: opt})
		if err != nil {
			p.API.LogWarn("Failed to list milestones", "error", err.Error())
			p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch milestones", StatusCode: http.StatusInternalServerError})
			return
		}
		allMilestones = append(allMilestones, milestones...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	p.writeJSON(w, allMilestones)
}

func (p *Plugin) getRepositories(w http.ResponseWriter, r *http.Request, userID string) {
	info, err := p.getGitHubUserInfo(userID)
	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	githubClient := p.githubConnect(*info.Token)

	ctx := context.Background()
	org := p.getConfiguration().GitHubOrg

	var allRepos []*github.Repository
	opt := github.ListOptions{PerPage: 50}

	if org == "" {
		for {
			repos, resp, err := githubClient.Repositories.List(ctx, "", &github.RepositoryListOptions{ListOptions: opt})
			if err != nil {
				p.API.LogWarn("Failed to list repositories", "error", err.Error())
				p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
				return
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	} else {
		for {
			repos, resp, err := githubClient.Repositories.ListByOrg(ctx, org, &github.RepositoryListByOrgOptions{Sort: "full_name", ListOptions: opt})
			if err != nil {
				p.API.LogWarn("Failed to list repositories by org", "error", err.Error())
				p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
				return
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	// Only send down fields to client that are needed
	type RepositoryResponse struct {
		Name        string          `json:"name,omitempty"`
		FullName    string          `json:"full_name,omitempty"`
		Permissions map[string]bool `json:"permissions,omitempty"`
	}

	resp := make([]RepositoryResponse, len(allRepos))
	for i, r := range allRepos {
		resp[i].Name = r.GetName()
		resp[i].FullName = r.GetFullName()
		resp[i].Permissions = r.GetPermissions()
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) createIssue(w http.ResponseWriter, r *http.Request, userID string) {
	type IssueRequest struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Repo      string   `json:"repo"`
		PostID    string   `json:"post_id"`
		Labels    []string `json:"labels"`
		Assignees []string `json:"assignees"`
		Milestone int      `json:"milestone"`
	}

	// get data for the issue from the request body and fill IssueRequest object
	issue := &IssueRequest{}
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		p.API.LogWarn("Error decoding JSON body", "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Title == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid issue title.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Repo == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repo name.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.PostID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a postID", StatusCode: http.StatusBadRequest})
		return
	}

	// Make sure user has a connected github account
	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	post, appErr := p.API.GetPost(issue.PostID)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load post " + issue.PostID, StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load post " + issue.PostID + ": not found", StatusCode: http.StatusNotFound})
		return
	}

	username, err := p.getUsername(post.UserId)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
		return
	}

	ghIssue := &github.IssueRequest{
		Title:     &issue.Title,
		Body:      &issue.Body,
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
	}

	// submitting the request with an invalid milestone ID results in a 422 error
	// we make sure it's not zero here, because the webapp client might have left this field empty
	if issue.Milestone > 0 {
		ghIssue.Milestone = &issue.Milestone
	}

	permalink := p.getPermaLink(issue.PostID)

	mmMessage := fmt.Sprintf("_Issue created from a [Mattermost message](%v) *by %s*._", permalink, username)

	if ghIssue.GetBody() != "" {
		mmMessage = "\n\n" + mmMessage
	}
	*ghIssue.Body = ghIssue.GetBody() + mmMessage

	currentUser, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	splittedRepo := strings.Split(issue.Repo, "/")
	owner := splittedRepo[0]
	repoName := splittedRepo[1]

	githubClient := p.githubConnect(*info.Token)
	result, resp, err := githubClient.Issues.Create(context.Background(), owner, repoName, ghIssue)
	if err != nil {
		p.API.LogWarn("Failed to create issue", "error", err.Error())
		p.writeAPIError(w,
			&APIErrorResponse{
				ID: "",
				Message: "failed to create issue: " + getFailReason(resp.StatusCode,
					issue.Repo,
					currentUser.Username,
				),
				StatusCode: resp.StatusCode,
			})
		return
	}

	if resp.Response.StatusCode == http.StatusGone {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
		return
	}

	rootID := issue.PostID
	if post.RootId != "" {
		rootID = post.RootId
	}

	message := fmt.Sprintf("Created GitHub issue [#%v](%v) from a [message](%s)", result.GetNumber(), result.GetHTMLURL(), permalink)
	reply := &model.Post{
		Message:   message,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		ParentId:  rootID,
		UserId:    userID,
	}

	_, appErr = p.API.CreatePost(reply)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to create notification post " + issue.PostID, StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeJSON(w, result)
}

func (p *Plugin) getConfig(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	p.writeJSON(w, config)
}

func (p *Plugin) getToken(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("userID")
	if userID == "" {
		http.Error(w, "please provide a userID", http.StatusBadRequest)
		return
	}

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		http.Error(w, apiErr.Error(), apiErr.StatusCode)
		return
	}

	p.writeJSON(w, info.Token)
}

// parseRepo parses the owner & repository name from the repo query parameter
func parseRepo(repoParam string) (owner, repo string, err error) {
	if repoParam == "" {
		return "", "", errors.New("repository cannot be blank")
	}

	splitted := strings.Split(repoParam, "/")
	if len(splitted) != 2 {
		return "", "", errors.New("invalid repository")
	}

	return splitted[0], splitted[1], nil
}
