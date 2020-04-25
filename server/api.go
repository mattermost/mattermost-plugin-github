package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"
)

const (
	apiErrorIDNotConnected = "not_connected"
	// the OAuth token expiry in seconds
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

type PRDetails struct {
	URL                string                      `json:"url"`
	Number             int                         `json:"number"`
	Status             string                      `json:"status"`
	RequestedReviewers []*string                   `json:"requestedReviewers"`
	Reviews            []*github.PullRequestReview `json:"reviews"`
}

type HTTPHandlerFuncWithUser func(w http.ResponseWriter, r *http.Request, userID string)

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

	oauthRouter := p.router.PathPrefix("/oauth").Subrouter()
	apiRouter := p.router.PathPrefix("/api/v1").Subrouter()

	p.router.HandleFunc("/webhook", p.handleWebhook).Methods("POST")

	oauthRouter.HandleFunc("/connect", p.extractUserMiddleWare(p.connectUserToGitHub, false)).Methods("GET")
	oauthRouter.HandleFunc("/complete", p.extractUserMiddleWare(p.completeConnectUserToGitHub, false)).Methods("GET")

	apiRouter.HandleFunc("/connected", p.extractUserMiddleWare(p.getConnected, true)).Methods("GET")
	apiRouter.HandleFunc("/todo", p.extractUserMiddleWare(p.postToDo, true)).Methods("POST")
	apiRouter.HandleFunc("/reviews", p.extractUserMiddleWare(p.getReviews, false)).Methods("GET")
	apiRouter.HandleFunc("/yourprs", p.extractUserMiddleWare(p.getYourPrs, false)).Methods("GET")
	apiRouter.HandleFunc("/prsdetails", p.extractUserMiddleWare(p.getPrsDetails, false)).Methods("POST")
	apiRouter.HandleFunc("/searchissues", p.extractUserMiddleWare(p.searchIssues, false)).Methods("GET")
	apiRouter.HandleFunc("/yourassignments", p.extractUserMiddleWare(p.getYourAssignments, false)).Methods("GET")
	apiRouter.HandleFunc("/createissuecomment", p.extractUserMiddleWare(p.createIssueComment, false)).Methods("POST")
	apiRouter.HandleFunc("/mentions", p.extractUserMiddleWare(p.getMentions, false)).Methods("GET")
	apiRouter.HandleFunc("/unreads", p.extractUserMiddleWare(p.getUnreads, false)).Methods("GET")
	apiRouter.HandleFunc("/settings", p.extractUserMiddleWare(p.updateSettings, false)).Methods("POST")
	apiRouter.HandleFunc("/user", p.extractUserMiddleWare(p.getGitHubUser, true)).Methods("POST")
}

func (p *Plugin) extractUserMiddleWare(handler HTTPHandlerFuncWithUser, jsonResponse bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			if jsonResponse {
				p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
			} else {
				http.Error(w, "Not authorized", http.StatusUnauthorized)
			}
			return
		}

		handler(w, r, userID)
	}
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

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
		fmt.Println(appErr.Error())
		http.Error(w, "missing stored state", http.StatusBadRequest)
		return
	}
	appErr = p.API.KVDelete(stateToken)
	if appErr != nil {
		fmt.Println(appErr.Error())
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
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubClient := p.githubConnect(*tok)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		fmt.Println(err.Error())
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
		fmt.Println(err.Error())
		http.Error(w, "Unable to connect user to GitHub", http.StatusInternalServerError)
		return
	}

	if err = p.storeGitHubToUserIDMapping(gitUser.GetLogin(), state.UserID); err != nil {
		fmt.Println(err.Error())
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
		strings.Replace(commandHelp, "|", "`", -1), gitUser.GetLogin(), gitUser.GetHTMLURL())
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

type ConnectedResponse struct {
	Connected         bool          `json:"connected"`
	GitHubUsername    string        `json:"github_username"`
	GitHubClientID    string        `json:"github_client_id"`
	EnterpriseBaseURL string        `json:"enterprise_base_url,omitempty"`
	Organization      string        `json:"organization"`
	Settings          *UserSettings `json:"settings"`
}

type CreateIssueCommentRequest struct {
	PostID      string `json:"post_id"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Comment     string `json:"comment"`
	CurrentTeam string `json:"current_team"`
}

type GitHubUserRequest struct {
	UserID string `json:"user_id"`
}

type GitHubUserResponse struct {
	Username string `json:"username"`
}

func (p *Plugin) getGitHubUser(w http.ResponseWriter, r *http.Request, _ string) {
	req := &GitHubUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Error("Error decoding GitHubUserRequest from JSON body", mlog.Err(err))
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

	resp := &GitHubUserResponse{Username: userInfo.GitHubUsername}

	p.writeJSON(w, resp)
}

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request, userID string) {
	config := p.getConfiguration()

	resp := &ConnectedResponse{
		Connected:         false,
		EnterpriseBaseURL: config.EnterpriseBaseURL,
		Organization:      config.GitHubOrg,
	}

	info, _ := p.getGitHubUserInfo(userID)
	if info != nil && info.Token != nil {
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
			hasBeenNotified := false
			val, err := p.API.KVGet(privateRepoStoreKey)
			if err != nil {
				mlog.Error("Unable to get private repo key value, err=" + err.Error())
			}
			hasBeenNotified = val != nil

			if !hasBeenNotified {
				p.CreateBotDMPost(info.UserID, "Private repositories have been enabled for this plugin. To be able to use them you must disconnect and reconnect your GitHub account. To reconnect your account, use the following slash commands: `/github disconnect` followed by `/github connect private`.", "")
				if err := p.API.KVSet(privateRepoStoreKey, []byte("1")); err != nil {
					mlog.Error("Unable to set private repo key value, err=" + err.Error())
				}
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

	result, _, err := githubClient.Search.Issues(context.Background(), getMentionSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
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
		mlog.Error(err.Error())
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

		filteredNotifications = append(filteredNotifications, &filteredNotification{
			Notification: *n,
			HTMLUrl:      fixGithubNotificationSubjectURL(n.GetSubject().GetURL()),
		})
	}

	p.writeJSON(w, filteredNotifications)
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

	result, _, err := githubClient.Search.Issues(context.Background(), getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
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

	result, _, err := githubClient.Search.Issues(context.Background(), getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
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
		mlog.Error("Error decoding PRDetails JSON body", mlog.Err(err))
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
			prDetail := fetchPRDetails(ctx, githubClient, pr.URL, pr.Number)
			prDetails[i] = prDetail
		}()
	}

	wg.Wait()

	p.writeJSON(w, prDetails)
}

func fetchPRDetails(ctx context.Context, client *github.Client, prURL string, prNumber int) *PRDetails {
	status := ""
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
			mlog.Error(err.Error())
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
			mlog.Error(err.Error())
			return
		}
		for _, v := range prInfo.RequestedReviewers {
			requestedReviewers = append(requestedReviewers, v.Login)
		}
		statuses, _, err := client.Repositories.GetCombinedStatus(ctx, repoOwner, repoName, prInfo.GetHead().GetSHA(), nil)
		if err != nil {
			mlog.Error(err.Error())
			return
		}
		status = *statuses.State
	}()

	wg.Wait()
	return &PRDetails{
		URL:                prURL,
		Number:             prNumber,
		Status:             status,
		RequestedReviewers: requestedReviewers,
		Reviews:            reviewsList,
	}
}

func fetchReviews(ctx context.Context, client *github.Client, repoOwner string, repoName string, number int) ([]*github.PullRequestReview, error) {
	reviewsList, _, err := client.PullRequests.ListReviews(ctx, repoOwner, repoName, number, nil)

	if err != nil {
		return []*github.PullRequestReview{}, err
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

	result, _, err := githubClient.Search.Issues(context.Background(), getIssuesSearchQuery(config.GitHubOrg, searchTerm), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	p.writeJSON(w, result.Issues)
}

func getPermaLink(siteURL string, postID string, currentTeam string) string {
	return fmt.Sprintf("%v/%v/pl/%v", siteURL, currentTeam, postID)
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
	req := &CreateIssueCommentRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Error("Error decoding CreateIssueCommentRequest JSON body", mlog.Err(err))
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

	if req.CurrentTeam == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid team", StatusCode: http.StatusBadRequest})
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

	commentUser, appErr := p.API.GetUser(post.UserId)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load post.UserID " + post.UserId + ": not found", StatusCode: http.StatusInternalServerError})
		return
	}

	currentUser, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	permalink := getPermaLink(siteURL, req.PostID, req.CurrentTeam)

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message](%s) *from @%s*\n", currentUser.Username, permalink, commentUser.Username)

	req.Comment = permalinkMessage + req.Comment
	comment := &github.IssueComment{
		Body: &req.Comment,
	}

	result, rawResponse, err := githubClient.Issues.CreateComment(context.Background(), req.Owner, req.Repo, req.Number, comment)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to create an issue comment: " + getFailReason(rawResponse.StatusCode, req.Repo, currentUser.Username), StatusCode: rawResponse.StatusCode})
		return
	}
	rootID := req.PostID
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	reply := &model.Post{
		Message:   fmt.Sprintf("Message attached to [#%v](https://github.com/%v/%v/issues/%v)", req.Number, req.Owner, req.Repo, req.Number),
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

	result, _, err := githubClient.Search.Issues(context.Background(), getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
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
		mlog.Error(err.Error())
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
		mlog.Error("Error decoding settings from JSON body", mlog.Err(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if settings == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, err := p.getGitHubUserInfo(userID)
	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	info.Settings = settings

	if err := p.storeGitHubUserInfo(info); err != nil {
		mlog.Error(err.Error())
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
		return
	}

	p.writeJSON(w, info.Settings)
}
