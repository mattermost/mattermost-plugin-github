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

	"github.com/google/go-github/v48/github"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-github/server/constants"
	"github.com/mattermost/mattermost-plugin-github/server/serializer"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// HTTPHandlerFuncWithUserContext is http.HandleFunc but with a UserContext attached
type HTTPHandlerFuncWithUserContext func(c *serializer.UserContext, w http.ResponseWriter, r *http.Request)

// HTTPHandlerFuncWithContext is http.HandleFunc but with a .ontext attached
type HTTPHandlerFuncWithContext func(c *serializer.Context, w http.ResponseWriter, r *http.Request)

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
	if _, err = w.Write(b); err != nil {
		p.API.LogWarn("Failed to write JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) writeAPIError(w http.ResponseWriter, apiErr *serializer.APIErrorResponse) {
	b, err := json.Marshal(apiErr)
	if err != nil {
		p.API.LogWarn("Failed to marshal API error", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(apiErr.StatusCode)

	if _, err = w.Write(b); err != nil {
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
	apiRouter.Use(p.checkConfigured)

	p.router.HandleFunc("/webhook", p.handleWebhook).Methods(http.MethodPost)

	oauthRouter.HandleFunc("/connect", p.checkAuth(p.attachContext(p.connectUserToGitHub), ResponseTypePlain)).Methods(http.MethodGet)
	oauthRouter.HandleFunc("/complete", p.checkAuth(p.attachContext(p.completeConnectUserToGitHub), ResponseTypePlain)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/connected", p.attachContext(p.getConnected)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/user", p.checkAuth(p.attachContext(p.getGitHubUser), ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/todo", p.checkAuth(p.attachUserContext(p.postToDo), ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/reviews", p.checkAuth(p.attachUserContext(p.getReviews), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/yourprs", p.checkAuth(p.attachUserContext(p.getYourPrs), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/prsdetails", p.checkAuth(p.attachUserContext(p.getPrsDetails), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/searchissues", p.checkAuth(p.attachUserContext(p.searchIssues), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/yourassignments", p.checkAuth(p.attachUserContext(p.getYourAssignments), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/createissue", p.checkAuth(p.attachUserContext(p.createIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/closeorreopenissue", p.checkAuth(p.attachUserContext(p.closeOrReopenIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/updateissue", p.checkAuth(p.attachUserContext(p.updateIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/editissuemodal", p.checkAuth(p.attachUserContext(p.openIssueEditModal), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/closereopenissuemodal", p.checkAuth(p.attachUserContext(p.openCloseOrReopenIssueModal), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/attachcommentissuemodal", p.checkAuth(p.attachUserContext(p.openAttachCommentIssueModal), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/createissuecomment", p.checkAuth(p.attachUserContext(p.createIssueComment), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/mentions", p.checkAuth(p.attachUserContext(p.getMentions), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/unreads", p.checkAuth(p.attachUserContext(p.getUnreads), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/labels", p.checkAuth(p.attachUserContext(p.getLabels), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/milestones", p.checkAuth(p.attachUserContext(p.getMilestones), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/assignees", p.checkAuth(p.attachUserContext(p.getAssignees), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/repositories", p.checkAuth(p.attachUserContext(p.getRepositories), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/settings", p.checkAuth(p.attachUserContext(p.updateSettings), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/issue", p.checkAuth(p.attachUserContext(p.getIssueByNumber), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/pr", p.checkAuth(p.attachUserContext(p.getPrByNumber), ResponseTypePlain)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/config", checkPluginRequest(p.getConfig)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/token", checkPluginRequest(p.getToken)).Methods(http.MethodGet)
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

func (p *Plugin) checkConfigured(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := p.getConfiguration()

		if err := config.IsValid(); err != nil {
			http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) checkAuth(handler http.HandlerFunc, responseType ResponseType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(constants.HeaderMattermostUserID)
		if userID == "" {
			switch responseType {
			case ResponseTypeJSON:
				p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
			case ResponseTypePlain:
				http.Error(w, "Not authorized", http.StatusUnauthorized)
			default:
				p.API.LogError("Unknown ResponseType detected")
			}
			return
		}

		handler(w, r)
	}
}

func (p *Plugin) createContext(_ http.ResponseWriter, r *http.Request) (*serializer.Context, context.CancelFunc) {
	userID := r.Header.Get(constants.HeaderMattermostUserID)

	logger := logger.New(p.API).With(logger.LogContext{
		"userid": userID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)

	context := &serializer.Context{
		Ctx:    ctx,
		UserID: userID,
		Log:    logger,
	}

	return context, cancel
}

func (p *Plugin) attachContext(handler HTTPHandlerFuncWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		context, cancel := p.createContext(w, r)
		defer cancel()

		handler(context, w, r)
	}
}

func (p *Plugin) attachUserContext(handler HTTPHandlerFuncWithUserContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		context, cancel := p.createContext(w, r)
		defer cancel()

		info, apiErr := p.getGitHubUserInfo(context.UserID)
		if apiErr != nil {
			p.writeAPIError(w, apiErr)
			return
		}

		context.Log = context.Log.With(logger.LogContext{
			"github username": info.GitHubUsername,
		})

		userContext := &serializer.UserContext{
			Context: *context,
			GHInfo:  info,
		}

		handler(userContext, w, r)
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
	w.Header().Set("Content-Type", "application/json")

	p.router.ServeHTTP(w, r)
}

func (p *Plugin) connectUserToGitHub(c *serializer.Context, w http.ResponseWriter, r *http.Request) {
	privateAllowed := false
	pValBool, _ := strconv.ParseBool(r.URL.Query().Get("private"))
	if pValBool {
		privateAllowed = true
	}

	conf := p.getOAuthConfig(privateAllowed)

	state := serializer.OAuthState{
		UserID:         c.UserID,
		Token:          model.NewId()[:15],
		PrivateAllowed: privateAllowed,
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		http.Error(w, "json marshal failed", http.StatusInternalServerError)
		return
	}

	appErr := p.API.KVSetWithExpiry(fmt.Sprintf("%s%s", githubOauthKey, state.Token), stateBytes, constants.TokenTTL)
	if appErr != nil {
		http.Error(w, "error setting stored state", http.StatusBadRequest)
		return
	}

	url := conf.AuthCodeURL(state.Token, oauth2.AccessTypeOffline)

	ch := p.oauthBroker.SubscribeOAuthComplete(c.UserID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		var errorMsg string
		select {
		case err := <-ch:
			if err != nil {
				errorMsg = err.Error()
			}
		case <-ctx.Done():
			errorMsg = "Timed out waiting for OAuth connection. Please check if the SiteURL is correct."
		}

		if errorMsg != "" {
			_, err := p.poster.DMWithAttachments(c.UserID, &model.SlackAttachment{
				Text:  fmt.Sprintf("There was an error connecting to your GitHub: `%s` Please double check your configuration.", errorMsg),
				Color: string(flow.ColorDanger),
			})
			if err != nil {
				c.Log.WithError(err).Warnf("Failed to DM with cancel information")
			}
		}

		p.oauthBroker.UnsubscribeOAuthComplete(c.UserID, ch)
	}()

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitHub(c *serializer.Context, w http.ResponseWriter, r *http.Request) {
	var rErr error
	defer func() {
		p.oauthBroker.publishOAuthComplete(c.UserID, rErr, false)
	}()

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		rErr = errors.New("missing authorization code")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	stateToken := r.URL.Query().Get("state")

	storedState, appErr := p.API.KVGet(fmt.Sprintf("%s%s", githubOauthKey, stateToken))
	if appErr != nil {
		c.Log.Warnf("Failed to get state token", "error", appErr.Error())

		rErr = errors.Wrap(appErr, "missing stored state")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}
	var state serializer.OAuthState

	if err := json.Unmarshal(storedState, &state); err != nil {
		rErr = errors.Wrap(err, "json unmarshal failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if appErr = p.API.KVDelete(fmt.Sprintf("%s%s", githubOauthKey, stateToken)); appErr != nil {
		c.Log.WithError(appErr).Warnf("Failed to delete state token")
		rErr = errors.Wrap(appErr, "error deleting stored state")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	if state.Token != stateToken {
		rErr = errors.New("invalid state token")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	if state.UserID != c.UserID {
		rErr = errors.New("not authorized, incorrect user")
		http.Error(w, rErr.Error(), http.StatusUnauthorized)
		return
	}

	conf := p.getOAuthConfig(state.PrivateAllowed)

	ctx, cancel := context.WithTimeout(context.Background(), constants.OauthCompleteTimeout)
	defer cancel()

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to exchange oauth code into token")

		rErr = errors.Wrap(err, "Failed to exchange oauth code into token")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	githubClient := p.githubConnectToken(*tok)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to get authenticated GitHub user")

		rErr = errors.Wrap(err, "failed to get authenticated GitHub user")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	userInfo := &serializer.GitHubUserInfo{
		UserID:         state.UserID,
		Token:          tok,
		GitHubUsername: gitUser.GetLogin(),
		LastToDoPostAt: model.GetMillis(),
		Settings: &serializer.UserSettings{
			SidebarButtons: settingButtonsTeam,
			DailyReminder:  true,
			Notifications:  true,
		},
		AllowedPrivateRepos:   state.PrivateAllowed,
		MM34646ResetTokenDone: true,
	}

	if err = p.storeGitHubUserInfo(userInfo); err != nil {
		c.Log.WithError(err).Warnf("Failed to store GitHub user info")

		rErr = errors.Wrap(err, "Unable to connect user to GitHub")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.storeGitHubToUserIDMapping(gitUser.GetLogin(), state.UserID); err != nil {
		c.Log.WithError(err).Warnf("Failed to store GitHub user info mapping")
	}

	flow := p.flowManager.setupFlow.ForUser(c.UserID)

	stepName, err := flow.GetCurrentStep()
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to get current step")
	}

	if stepName == stepOAuthConnect {
		if err = flow.Go(stepWebhookQuestion); err != nil {
			c.Log.WithError(err).Warnf("Failed go to next step")
		}
	} else {
		// Only post introduction message if no setup wizard is running

		var commandHelp string
		commandHelp, err = renderTemplate("helpText", p.getConfiguration())
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to render help template")
		}

		message := fmt.Sprintf("#### Welcome to the Mattermost GitHub Plugin!\n"+
			"You've connected your Mattermost account to [%s](%s) on GitHub. Read about the features of this plugin below:\n\n"+
			"##### Daily Reminders\n"+
			"The first time you log in each day, you'll get a post right here letting you know what messages you need to read and what pull requests are awaiting your review.\n"+
			"Turn off reminders with `/github settings reminders off`.\n\n"+
			"##### Notifications\n"+
			"When someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you, you'll get a post here about it.\n"+
			"Turn off notifications with `/github settings notifications off`.\n\n"+
			"##### Sidebar Buttons\n"+
			"Check out the buttons in the left-hand sidebar of Mattermost.\n"+
			"It shows your Open PRs, PRs that are awaiting your review, issues assigned to you, and all your unread messages you have in GitHub. \n"+
			"* The first button tells you how many pull requests you have submitted.\n"+
			"* The second shows the number of PR that are awaiting your review.\n"+
			"* The third shows the number of PR and issues your are assiged to.\n"+
			"* The fourth tracks the number of unread messages you have.\n"+
			"* The fifth will refresh the numbers.\n\n"+
			"Click on them!\n\n"+
			"##### Slash Commands\n"+
			commandHelp, gitUser.GetLogin(), gitUser.GetHTMLURL())

		p.CreateBotDMPost(state.UserID, message, "custom_git_welcome")
	}

	config := p.getConfiguration()

	p.API.PublishWebSocketEvent(
		wsEventConnect,
		map[string]interface{}{
			"connected":           true,
			"github_username":     userInfo.GitHubUsername,
			"github_client_id":    config.GitHubOAuthClientID,
			"enterprise_base_url": config.EnterpriseBaseURL,
			"organization":        config.GitHubOrg,
			"configuration":       config.ClientConfiguration(),
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
	if _, err = w.Write([]byte(html)); err != nil {
		c.Log.WithError(err).Warnf("Failed to write HTML response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) getGitHubUser(c *serializer.Context, w http.ResponseWriter, r *http.Request) {
	req := &serializer.GitHubUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding GitHubUserRequest from JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.UserID == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a JSON object with a non-blank user_id field.", StatusCode: http.StatusBadRequest})
		return
	}

	userInfo, apiErr := p.getGitHubUserInfo(req.UserID)
	if apiErr != nil {
		if apiErr.ID == constants.APIErrorIDNotConnected {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		} else {
			p.writeAPIError(w, apiErr)
		}
		return
	}

	if userInfo == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		return
	}

	resp := &serializer.GitHubUserResponse{Username: userInfo.GitHubUsername}
	p.writeJSON(w, resp)
}

func (p *Plugin) getConnected(c *serializer.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	resp := &serializer.ConnectedResponse{
		Connected:           false,
		EnterpriseBaseURL:   config.EnterpriseBaseURL,
		Organization:        config.GitHubOrg,
		ClientConfiguration: p.getConfiguration().ClientConfiguration(),
	}

	if c.UserID == "" {
		p.writeJSON(w, resp)
		return
	}

	info, _ := p.getGitHubUserInfo(c.UserID)
	if info == nil || info.Token == nil {
		p.writeJSON(w, resp)
		return
	}

	resp.Connected = true
	resp.GitHubUsername = info.GitHubUsername
	resp.GitHubClientID = config.GitHubOAuthClientID
	resp.UserSettings = info.Settings

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
				if err := p.PostToDo(info, c.UserID); err != nil {
					c.Log.WithError(err).Warnf("Failed to create GitHub todo message")
				}
				info.LastToDoPostAt = now
				if err := p.storeGitHubUserInfo(info); err != nil {
					c.Log.WithError(err).Warnf("Failed to store github info for new user")
				}
			}
		}
	}

	privateRepoStoreKey := fmt.Sprintf("%s%s", info.UserID, githubPrivateRepoKey)
	if config.EnablePrivateRepo && !info.AllowedPrivateRepos {
		val, err := p.API.KVGet(privateRepoStoreKey)
		if err != nil {
			c.Log.WithError(err).Warnf("Unable to get private repo key value")
			return
		}

		// Inform the user once that private repositories enabled
		if val == nil {
			message := "Private repositories have been enabled for this plugin. To be able to use them you must disconnect and reconnect your GitHub account. To reconnect your account, use the following slash commands: `/github disconnect` followed by %s"
			if config.ConnectToPrivateByDefault {
				p.CreateBotDMPost(info.UserID, fmt.Sprintf(message, "`/github connect`."), "")
			} else {
				p.CreateBotDMPost(info.UserID, fmt.Sprintf(message, "`/github connect private`."), "")
			}
			if err := p.API.KVSet(privateRepoStoreKey, []byte("1")); err != nil {
				c.Log.WithError(err).Warnf("Unable to set private repo key value")
			}
		}
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) getMentions(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	username := c.GHInfo.GitHubUsername
	query := getMentionSearchQuery(username, config.GitHubOrg)

	result, _, err := githubClient.Search.Issues(c.Ctx, query, &github.SearchOptions{})
	if err != nil {
		c.Log.WithError(err).With(logger.LogContext{"query": query}).Warnf("Failed to search for issues")
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getUnreads(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	notifications, _, err := githubClient.Activity.ListNotifications(c.Ctx, &github.NotificationListOptions{})
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to list notifications")
		return
	}

	filteredNotifications := []*serializer.FilteredNotification{}
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

		filteredNotifications = append(filteredNotifications, &serializer.FilteredNotification{
			Notification: *n,
			HTMLUrl:      fixGithubNotificationSubjectURL(subjectURL, issueNum),
		})
	}

	p.writeJSON(w, filteredNotifications)
}

func (p *Plugin) getReviews(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	username := c.GHInfo.GitHubUsername

	query := getReviewSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(c.Ctx, query, &github.SearchOptions{})
	if err != nil {
		c.Log.WithError(err).With(logger.LogContext{"query": query}).Warnf("Failed to search for review")
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getYourPrs(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	username := c.GHInfo.GitHubUsername

	query := getYourPrsSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(c.Ctx, query, &github.SearchOptions{})
	if err != nil {
		c.Log.WithError(err).With(logger.LogContext{"query": query}).Warnf("Failed to search for PRs")
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) getPrsDetails(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	var prList []*serializer.PRDetails
	if err := json.NewDecoder(r.Body).Decode(&prList); err != nil {
		c.Log.WithError(err).Warnf("Error decoding PRDetails JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	prDetails := make([]*serializer.PRDetails, len(prList))
	var wg sync.WaitGroup
	for i, pr := range prList {
		i := i
		pr := pr
		wg.Add(1)
		go func() {
			defer wg.Done()
			prDetail := p.fetchPRDetails(c, githubClient, pr.URL, pr.Number)
			prDetails[i] = prDetail
		}()
	}

	wg.Wait()

	p.writeJSON(w, prDetails)
}

func (p *Plugin) fetchPRDetails(c *serializer.UserContext, client *github.Client, prURL string, prNumber int) *serializer.PRDetails {
	var status string
	var mergeable bool
	// Initialize to a non-nil slice to simplify JSON handling semantics
	requestedReviewers := []*string{}
	reviewsList := []*github.PullRequestReview{}

	repoOwner, repoName := getRepoOwnerAndNameFromURL(prURL)

	var wg sync.WaitGroup

	// Fetch reviews
	wg.Add(1)
	go func() {
		defer wg.Done()
		fetchedReviews, err := fetchReviews(c, client, repoOwner, repoName, prNumber)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to fetch reviews for PR details")
			return
		}
		reviewsList = fetchedReviews
	}()

	// Fetch reviewers and status
	wg.Add(1)
	go func() {
		defer wg.Done()
		prInfo, _, err := client.PullRequests.Get(c.Ctx, repoOwner, repoName, prNumber)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to fetch PR for PR details")
			return
		}

		mergeable = prInfo.GetMergeable()

		for _, v := range prInfo.RequestedReviewers {
			requestedReviewers = append(requestedReviewers, v.Login)
		}
		statuses, _, err := client.Repositories.GetCombinedStatus(c.Ctx, repoOwner, repoName, prInfo.GetHead().GetSHA(), nil)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to fetch combined status")
			return
		}
		status = *statuses.State
	}()

	wg.Wait()
	return &serializer.PRDetails{
		URL:                prURL,
		Number:             prNumber,
		Status:             status,
		Mergeable:          mergeable,
		RequestedReviewers: requestedReviewers,
		Reviews:            reviewsList,
	}
}

func fetchReviews(c *serializer.UserContext, client *github.Client, repoOwner string, repoName string, number int) ([]*github.PullRequestReview, error) {
	reviewsList, _, err := client.PullRequests.ListReviews(c.Ctx, repoOwner, repoName, number, nil)

	if err != nil {
		return []*github.PullRequestReview{}, errors.Wrap(err, "could not list reviews")
	}

	return reviewsList, nil
}

func getRepoOwnerAndNameFromURL(url string) (string, string) {
	splitted := strings.Split(url, "/")
	return splitted[len(splitted)-2], splitted[len(splitted)-1]
}

func (p *Plugin) searchIssues(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	searchTerm := r.FormValue("term")
	query := getIssuesSearchQuery(config.GitHubOrg, searchTerm)
	result, _, err := githubClient.Search.Issues(c.Ctx, query, &github.SearchOptions{})
	if err != nil {
		c.Log.WithError(err).With(logger.LogContext{"query": query}).Warnf("Failed to search for issues")
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

func (p *Plugin) createIssueComment(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	req := &serializer.CreateIssueCommentRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding CreateIssueCommentRequest JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.PostID == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid post id", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Owner == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid repository owner.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Repo == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid repository.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Number == 0 {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid issue number.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Comment == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid non empty comment.", StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	commentUsername, err := p.getUsername(post.UserId)
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
		return
	}

	currentUsername := c.GHInfo.GitHubUsername
	permalink := p.getPermaLink(req.PostID)
	permalinkMessage := fmt.Sprintf("*@%s attached a* [message](%s) *from %s*\n\n", currentUsername, permalink, commentUsername)

	if req.ShowAttachedMessage {
		req.Comment = fmt.Sprintf("%s%s", permalinkMessage, req.Comment)
	}
	comment := &github.IssueComment{
		Body: &req.Comment,
	}

	result, rawResponse, err := githubClient.Issues.CreateComment(c.Ctx, req.Owner, req.Repo, req.Number, comment)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if rawResponse != nil {
			statusCode = rawResponse.StatusCode
		}
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create an issue comment: %s", getFailReason(statusCode, req.Repo, currentUsername)), StatusCode: statusCode})
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
		UserId:    c.UserID,
	}

	if _, appErr = p.API.CreatePost(reply); appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeJSON(w, result)
}

func (p *Plugin) getYourAssignments(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	username := c.GHInfo.GitHubUsername
	query := getYourAssigneeSearchQuery(username, config.GitHubOrg)
	result, _, err := githubClient.Search.Issues(c.Ctx, query, &github.SearchOptions{})
	if err != nil {
		c.Log.WithError(err).With(logger.LogContext{"query": query}).Warnf("Failed to search for assignments")
		return
	}

	p.writeJSON(w, result.Issues)
}

func (p *Plugin) postToDo(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	username := c.GHInfo.GitHubUsername

	text, err := p.GetToDo(c.Ctx, username, githubClient)
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to get Todos")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	p.CreateBotDMPost(c.UserID, text, "custom_git_todo")

	resp := struct {
		Status string
	}{"OK"}

	p.writeJSON(w, resp)
}

func (p *Plugin) updateSettings(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	var settings *serializer.UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		c.Log.WithError(err).Warnf("Error decoding settings from JSON body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if settings == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info := c.GHInfo
	info.Settings = settings

	if err := p.storeGitHubUserInfo(info); err != nil {
		c.Log.WithError(err).Warnf("Failed to store GitHub user info")
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
		return
	}

	p.writeJSON(w, info.Settings)
}

func (p *Plugin) openAttachCommentIssueModal(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	req := &serializer.OpenCreateCommentOrEditIssueModalRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	userID := r.Header.Get(constants.HeaderMattermostUserID)
	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	p.API.PublishWebSocketEvent(
		wsEventAttachCommentToIssue,
		map[string]interface{}{
			"postId": post.Id,
			"owner":  req.RepoOwner,
			"repo":   req.RepoName,
			"number": req.IssueNumber,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) openCloseOrReopenIssueModal(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	req := &serializer.OpenCreateCommentOrEditIssueModalRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	userID := r.Header.Get(constants.HeaderMattermostUserID)

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	p.API.PublishWebSocketEvent(
		wsEventCloseOrReopenIssue,
		map[string]interface{}{
			"channel_id": post.ChannelId,
			"owner":      req.RepoOwner,
			"repo":       req.RepoName,
			"number":     req.IssueNumber,
			"status":     req.Status,
			"postId":     req.PostID,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) openIssueEditModal(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	req := &serializer.OpenCreateCommentOrEditIssueModalRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	issue, _, err := githubClient.Issues.Get(c.Ctx, req.RepoOwner, req.RepoName, req.IssueNumber)
	if err != nil {
		// If the issue is not found, it probably belongs to a private repo.
		// Return an empty response in that case.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) && gerr.Response.StatusCode == http.StatusNotFound {
			c.Log.WithError(err).With(logger.LogContext{
				"owner":  req.RepoOwner,
				"repo":   req.RepoName,
				"number": req.IssueNumber,
			}).Debugf("Issue not found")
			p.writeJSON(w, nil)
			return
		}

		c.Log.WithError(err).With(logger.LogContext{
			"owner":  req.RepoOwner,
			"repo":   req.RepoName,
			"number": req.IssueNumber,
		}).Debugf("Could not get the issue")
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Could not get the issue", StatusCode: http.StatusInternalServerError})
		return
	}

	description := ""
	if issue.Body != nil {
		*issue.Body = mdCommentRegex.ReplaceAllString(issue.GetBody(), "")
		description = *issue.Body
	}

	assignees := make([]string, len(issue.Assignees))
	for index, user := range issue.Assignees {
		assignees[index] = user.GetLogin()
	}

	labels := make([]string, len(issue.Labels))
	for index, label := range issue.Labels {
		labels[index] = label.GetName()
	}

	milestoneTitle := ""
	var milestoneNumber int
	if issue.Milestone != nil && issue.Milestone.Title != nil {
		milestoneTitle = *issue.Milestone.Title
		milestoneNumber = *issue.Milestone.Number
	}

	userID := r.Header.Get(constants.HeaderMattermostUserID)
	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	p.API.PublishWebSocketEvent(
		wsEventCreateOrUpdateIssue,
		map[string]interface{}{
			"title":            *issue.Title,
			"channel_id":       post.ChannelId,
			"postId":           req.PostID,
			"milestone_title":  milestoneTitle,
			"milestone_number": milestoneNumber,
			"assignees":        assignees,
			"labels":           labels,
			"description":      description,
			"repo_full_name":   fmt.Sprintf("%s/%s", req.RepoOwner, req.RepoName),
			"issue_number":     *issue.Number,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)

	p.writeJSON(w, issue)
}

func (p *Plugin) getIssueByNumber(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue(constants.OwnerQueryParam)
	repo := r.FormValue(constants.RepoQueryParam)
	number := r.FormValue(constants.NumberQueryParam)
	issueNumber, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	result, _, err := githubClient.Issues.Get(c.Ctx, owner, repo, issueNumber)
	if err != nil {
		// If the issue is not found, it probably belongs to a private repo.
		// Return an empty response in that case.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) && gerr.Response.StatusCode == http.StatusNotFound {
			c.Log.WithError(err).With(logger.LogContext{
				"owner":  owner,
				"repo":   repo,
				"number": issueNumber,
			}).Debugf("Issue not found")
			p.writeJSON(w, nil)
			return
		}

		c.Log.WithError(err).With(logger.LogContext{
			"owner":  owner,
			"repo":   repo,
			"number": issueNumber,
		}).Debugf("Could not get the issue")
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Could not get the issue", StatusCode: http.StatusInternalServerError})
		return
	}

	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getPrByNumber(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue(constants.OwnerQueryParam)
	repo := r.FormValue(constants.RepoQueryParam)
	number := r.FormValue(constants.NumberQueryParam)

	prNumber, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	result, _, err := githubClient.PullRequests.Get(c.Ctx, owner, repo, prNumber)
	if err != nil {
		// If the pull request is not found, it's probably behind a private repo.
		// Return an empty response in that case.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) && gerr.Response.StatusCode == http.StatusNotFound {
			c.Log.With(logger.LogContext{
				"owner":  owner,
				"repo":   repo,
				"number": prNumber,
			}).Debugf("Pull request not found")

			p.writeJSON(w, nil)
			return
		}

		c.Log.WithError(err).With(logger.LogContext{
			"owner":  owner,
			"repo":   repo,
			"number": prNumber,
		}).Debugf("Could not get pull request")
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Could not get pull request", StatusCode: http.StatusInternalServerError})
		return
	}
	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getLabels(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allLabels []*github.Label
	opt := github.ListOptions{PerPage: 50}

	for {
		labels, resp, err := githubClient.Issues.ListLabels(c.Ctx, owner, repo, &opt)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list labels")
			p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Failed to fetch labels", StatusCode: http.StatusInternalServerError})
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

func (p *Plugin) getAssignees(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allAssignees []*github.User
	opt := github.ListOptions{PerPage: 50}

	for {
		assignees, resp, err := githubClient.Issues.ListAssignees(c.Ctx, owner, repo, &opt)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list assignees")
			p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Failed to fetch assignees", StatusCode: http.StatusInternalServerError})
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

func (p *Plugin) getMilestones(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allMilestones []*github.Milestone
	opt := github.ListOptions{PerPage: 50}

	for {
		milestones, resp, err := githubClient.Issues.ListMilestones(c.Ctx, owner, repo, &github.MilestoneListOptions{ListOptions: opt})
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list milestones")
			p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Failed to fetch milestones", StatusCode: http.StatusInternalServerError})
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

func (p *Plugin) getRepositories(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	org := p.getConfiguration().GitHubOrg

	var allRepos []*github.Repository
	opt := github.ListOptions{PerPage: 50}

	if org == "" {
		for {
			repos, resp, err := githubClient.Repositories.List(c.Ctx, "", &github.RepositoryListOptions{ListOptions: opt})
			if err != nil {
				c.Log.WithError(err).Warnf("Failed to list repositories")
				p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
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
			repos, resp, err := githubClient.Repositories.ListByOrg(c.Ctx, org, &github.RepositoryListByOrgOptions{Sort: "full_name", ListOptions: opt})
			if err != nil {
				c.Log.WithError(err).Warnf("Failed to list repositories by org")
				p.writeAPIError(w, &serializer.APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
				return
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	resp := make([]serializer.RepositoryResponse, len(allRepos))
	for i, r := range allRepos {
		resp[i].Name = r.GetName()
		resp[i].FullName = r.GetFullName()
		resp[i].Permissions = r.GetPermissions()
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) updateIssue(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	// get data for the issue from the request body and fill UpdateIssueRequest to update the issue
	issue := &serializer.UpdateIssueRequest{}
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if !p.validateIssueRequestForUpdation(issue, w) {
		return
	}

	var post *model.Post
	permalink := ""
	if issue.PostID != "" {
		var appErr *model.AppError
		post, appErr = p.API.GetPost(issue.PostID)
		if appErr != nil {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
			return
		}
		if post == nil {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", issue.PostID), StatusCode: http.StatusNotFound})
			return
		}
		permalink = p.getPermaLink(issue.PostID)
	}

	githubIssue := &github.IssueRequest{
		Title:     &issue.Title,
		Body:      &issue.Body,
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
	}

	// submitting the request with an invalid milestone ID results in a 422 error
	// we should make sure it's not zero here because the webapp client might have left this field empty
	if issue.Milestone > 0 {
		githubIssue.Milestone = &issue.Milestone
	}

	currentUser, appErr := p.API.GetUser(c.UserID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	splittedRepo := strings.Split(issue.Repo, "/")
	if len(splittedRepo) < 2 {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid repository", StatusCode: http.StatusBadRequest})
	}

	owner, repoName := splittedRepo[0], splittedRepo[1]
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	result, resp, err := githubClient.Issues.Edit(c.Ctx, owner, repoName, issue.IssueNumber, githubIssue)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusGone {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
			return
		}

		c.Log.WithError(err).Warnf("Failed to update the issue")
		p.writeAPIError(w, &serializer.APIErrorResponse{
			ID: "",
			Message: fmt.Sprintf("failed to update the issue: %s", getFailReason(resp.StatusCode,
				issue.Repo,
				currentUser.Username,
			)),
			StatusCode: resp.StatusCode,
		})
		return
	}

	rootID := issue.PostID
	channelID := issue.ChannelID
	message := fmt.Sprintf("Updated GitHub issue [#%v](%v)", result.GetNumber(), result.GetHTMLURL())
	if post != nil {
		if post.RootId != "" {
			rootID = post.RootId
		}
		channelID = post.ChannelId
		message += fmt.Sprintf(" from a [message](%s)", permalink)
	}

	reply := &model.Post{
		Message:   message,
		ChannelId: channelID,
		RootId:    rootID,
		UserId:    c.UserID,
	}

	if post != nil {
		_, appErr = p.API.CreatePost(reply)
	} else {
		_ = p.API.SendEphemeralPost(c.UserID, reply)
	}
	if appErr != nil {
		c.Log.WithError(appErr).Warnf("failed to create the notification post")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post, postID: %s, channelID: %s", issue.PostID, channelID), StatusCode: http.StatusInternalServerError})
		return
	}

	p.updatePost(post, issue, w)
	p.writeJSON(w, result)
}

func (p *Plugin) closeOrReopenIssue(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	req := &serializer.CommentAndCloseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	if _, err := p.getUsername(post.UserId); err != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
		return
	}
	if req.IssueComment != "" {
		p.CreateCommentToIssue(c, w, req.IssueComment, req.Owner, req.Repository, post, req.Number)
	}

	if req.Status == constants.Close {
		p.CloseOrReopenIssue(c, w, constants.IssueClose, req.StatusReason, req.Owner, req.Repository, post, req.Number)
	} else {
		p.CloseOrReopenIssue(c, w, constants.IssueOpen, req.StatusReason, req.Owner, req.Repository, post, req.Number)
	}
}

func (p *Plugin) createIssue(c *serializer.UserContext, w http.ResponseWriter, r *http.Request) {
	// get data for the issue from the request body and fill CreateIssueRequest object to create the issue
	issue := &serializer.CreateIssueRequest{}
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Title == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid issue title.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Repo == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide a valid repository name.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.PostID == "" && issue.ChannelID == "" {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Please provide either a postID or a channelID", StatusCode: http.StatusBadRequest})
		return
	}

	mmMessage := ""
	var post *model.Post
	permalink := ""
	if issue.PostID != "" {
		var appErr *model.AppError
		post, appErr = p.API.GetPost(issue.PostID)
		if appErr != nil {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
			return
		}
		if post == nil {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", issue.PostID), StatusCode: http.StatusNotFound})
			return
		}

		username, err := p.getUsername(post.UserId)
		if err != nil {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
			return
		}

		permalink = p.getPermaLink(issue.PostID)

		mmMessage = fmt.Sprintf("_Issue created from a [Mattermost message](%v) *by %s*._", permalink, username)
	}

	githubIssue := &github.IssueRequest{
		Title:     &issue.Title,
		Body:      &issue.Body,
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
	}

	// submitting the request with an invalid milestone ID results in a 422 error
	// we should make sure it's not zero here because the webapp client might have left this field empty
	if issue.Milestone > 0 {
		githubIssue.Milestone = &issue.Milestone
	}

	if githubIssue.GetBody() != "" && mmMessage != "" {
		mmMessage = "\n\n" + mmMessage
	}
	*githubIssue.Body = fmt.Sprintf("%s%s", githubIssue.GetBody(), mmMessage)

	currentUser, appErr := p.API.GetUser(c.UserID)
	if appErr != nil {
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	splittedRepo := strings.Split(issue.Repo, "/")
	owner, repoName := splittedRepo[0], splittedRepo[1]

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	result, resp, err := githubClient.Issues.Create(c.Ctx, owner, repoName, githubIssue)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusGone {
			p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
			return
		}

		c.Log.WithError(err).Warnf("Failed to create issue")
		p.writeAPIError(w, &serializer.APIErrorResponse{
			ID:         "",
			Message:    fmt.Sprintf("failed to create issue: %s", getFailReason(resp.StatusCode, issue.Repo, currentUser.Username)),
			StatusCode: resp.StatusCode,
		})
		return
	}

	rootID := issue.PostID
	channelID := issue.ChannelID
	message := fmt.Sprintf("Created GitHub issue [#%v](%v)", result.GetNumber(), result.GetHTMLURL())
	if post != nil {
		if post.RootId != "" {
			rootID = post.RootId
		}
		channelID = post.ChannelId
		message += fmt.Sprintf(" from a [message](%s)", permalink)
	}

	reply := &model.Post{
		Message:   message,
		ChannelId: channelID,
		RootId:    rootID,
		UserId:    c.UserID,
	}

	if post != nil {
		_, appErr = p.API.CreatePost(reply)
	} else {
		_ = p.API.SendEphemeralPost(c.UserID, reply)
	}
	if appErr != nil {
		c.Log.WithError(appErr).Warnf("failed to create the notification post")
		p.writeAPIError(w, &serializer.APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post, postID: %s, channelID: %s", issue.PostID, channelID), StatusCode: http.StatusInternalServerError})
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
