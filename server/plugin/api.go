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

	"github.com/google/go-github/v54/github"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"
)

// HTTPHandlerFuncWithUserContext is http.HandleFunc but with a UserContext attached
type HTTPHandlerFuncWithUserContext func(c *UserContext, w http.ResponseWriter, r *http.Request)

// HTTPHandlerFuncWithContext is http.HandleFunc but with a .ontext attached
type HTTPHandlerFuncWithContext func(c *Context, w http.ResponseWriter, r *http.Request)

// ResponseType indicates type of response returned by api
type ResponseType string

type UpdateIssueRequest struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Repo        string   `json:"repo"`
	PostID      string   `json:"post_id"`
	ChannelID   string   `json:"channel_id"`
	Labels      []string `json:"labels"`
	Assignees   []string `json:"assignees"`
	Milestone   int      `json:"milestone"`
	IssueNumber int      `json:"issue_number"`
}

type PRDetails struct {
	URL                string                      `json:"url"`
	Number             int                         `json:"number"`
	Status             string                      `json:"status"`
	Mergeable          bool                        `json:"mergeable"`
	RequestedReviewers []*string                   `json:"requestedReviewers"`
	Reviews            []*github.PullRequestReview `json:"reviews"`
}

const (
	// ResponseTypeJSON indicates that response type is json
	ResponseTypeJSON ResponseType = "JSON_RESPONSE"
	// ResponseTypePlain indicates that response type is text plain
	ResponseTypePlain ResponseType = "TEXT_RESPONSE"

	KeyRepoName    string = "repo_name"
	KeyRepoOwner   string = "repo_owner"
	KeyIssueNumber string = "issue_number"
	KeyIssueID     string = "issue_id"
	KeyStatus      string = "status"
	KeyChannelID   string = "channel_id"
	KeyPostID      string = "postId"

	WebsocketEventOpenCommentModal string = "open_comment_modal"
	WebsocketEventOpenStatusModal  string = "open_status_modal"
	WebsocketEventOpenEditModal    string = "open_edit_modal"

	PathOpenIssueCommentModal string = "/open-comment-modal"
	PathOpenIssueEditModal    string = "/open-edit-modal"
	PathOpenIssueStatusModal  string = "/open-status-modal"
)

func (p *Plugin) writeJSON(w http.ResponseWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		p.client.Log.Warn("Failed to marshal JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(b); err != nil {
		p.client.Log.Warn("Failed to write JSON response", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) writeAPIError(w http.ResponseWriter, apiErr *APIErrorResponse) {
	b, err := json.Marshal(apiErr)
	if err != nil {
		p.client.Log.Warn("Failed to marshal API error", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(apiErr.StatusCode)

	if _, err := w.Write(b); err != nil {
		p.client.Log.Warn("Failed to write JSON response", "error", err.Error())
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
	apiRouter.HandleFunc("/prs_details", p.checkAuth(p.attachUserContext(p.getPrsDetails), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/search_issues", p.checkAuth(p.attachUserContext(p.searchIssues), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/create_issue", p.checkAuth(p.attachUserContext(p.createIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/close_or_reopen_issue", p.checkAuth(p.attachUserContext(p.closeOrReopenIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/update_issue", p.checkAuth(p.attachUserContext(p.updateIssue), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/issue_info", p.checkAuth(p.attachUserContext(p.getIssueInfo), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/create_issue_comment", p.checkAuth(p.attachUserContext(p.createIssueComment), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/mentions", p.checkAuth(p.attachUserContext(p.getMentions), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/labels", p.checkAuth(p.attachUserContext(p.getLabels), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/milestones", p.checkAuth(p.attachUserContext(p.getMilestones), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/assignees", p.checkAuth(p.attachUserContext(p.getAssignees), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/repositories", p.checkAuth(p.attachUserContext(p.getRepositories), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/settings", p.checkAuth(p.attachUserContext(p.updateSettings), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/issue", p.checkAuth(p.attachUserContext(p.getIssueByNumber), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/pr", p.checkAuth(p.attachUserContext(p.getPrByNumber), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/lhs-content", p.checkAuth(p.attachUserContext(p.getSidebarContent), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc(PathOpenIssueCommentModal, p.checkAuth(p.attachUserContext(p.handleOpenIssueCommentModal), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathOpenIssueEditModal, p.checkAuth(p.attachUserContext(p.handleOpenEditIssueModal), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathOpenIssueStatusModal, p.checkAuth(p.attachUserContext(p.handleOpenIssueStatusModal), ResponseTypePlain)).Methods(http.MethodPost)

	apiRouter.HandleFunc("/config", checkPluginRequest(p.getConfig)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/token", checkPluginRequest(p.getToken)).Methods(http.MethodGet)
}

func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.client.Log.Warn("Recovered from a panic",
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
		userID := r.Header.Get(headerMattermostUserID)
		if userID == "" {
			switch responseType {
			case ResponseTypeJSON:
				p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
			case ResponseTypePlain:
				http.Error(w, "Not authorized", http.StatusUnauthorized)
			default:
				p.client.Log.Debug("Unknown ResponseType detected")
			}
			return
		}

		handler(w, r)
	}
}

func (p *Plugin) createContext(_ http.ResponseWriter, r *http.Request) (*Context, context.CancelFunc) {
	userID := r.Header.Get(headerMattermostUserID)

	logger := logger.New(p.API).With(logger.LogContext{
		"userid": userID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	context := &Context{
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

		userContext := &UserContext{
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

func (p *Plugin) connectUserToGitHub(c *Context, w http.ResponseWriter, r *http.Request) {
	privateAllowed := false
	pValBool, _ := strconv.ParseBool(r.URL.Query().Get("private"))
	if pValBool {
		privateAllowed = true
	}

	conf := p.getOAuthConfig(privateAllowed)

	state := OAuthState{
		UserID:         c.UserID,
		Token:          model.NewId()[:15],
		PrivateAllowed: privateAllowed,
	}

	_, err := p.store.Set(githubOauthKey+state.Token, state, pluginapi.SetExpiry(tokenTTL))
	if err != nil {
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

func (p *Plugin) completeConnectUserToGitHub(c *Context, w http.ResponseWriter, r *http.Request) {
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

	var state OAuthState
	if err := p.store.Get(fmt.Sprintf("%s%s", githubOauthKey, stateToken), &state); err != nil {
		c.Log.Warnf("Failed to get state token", "error", err.Error())
		rErr = errors.Wrap(err, "missing stored state")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	if err := p.store.Delete(fmt.Sprintf("%s%s", githubOauthKey, stateToken)); err != nil {
		c.Log.WithError(err).Warnf("Failed to delete state token")
		rErr = errors.Wrap(err, "error deleting stored state")
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

	ctx, cancel := context.WithTimeout(context.Background(), oauthCompleteTimeout)
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

	// track the successful connection
	p.TrackUserEvent("account_connected", c.UserID, nil)

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

	p.client.Frontend.PublishWebSocketEvent(
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

func (p *Plugin) getGitHubUser(c *Context, w http.ResponseWriter, r *http.Request) {
	req := &GitHubUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding GitHubUserRequest from JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
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

func (p *Plugin) getConnected(c *Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	resp := &ConnectedResponse{
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
		var val []byte
		err := p.store.Get(privateRepoStoreKey, &val)
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
			if _, err := p.store.Set(privateRepoStoreKey, []byte("1")); err != nil {
				c.Log.WithError(err).Warnf("Unable to set private repo key value")
			}
		}
	}

	p.writeJSON(w, resp)
}

func (p *Plugin) getMentions(c *UserContext, w http.ResponseWriter, r *http.Request) {
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

func (p *Plugin) getUnreadsData(c *UserContext) []*FilteredNotification {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	notifications, _, err := githubClient.Activity.ListNotifications(c.Ctx, &github.NotificationListOptions{})
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to list notifications")
		return nil
	}

	filteredNotifications := []*FilteredNotification{}
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

		filteredNotifications = append(filteredNotifications, &FilteredNotification{
			Notification: *n,
			HTMLURL:      fixGithubNotificationSubjectURL(subjectURL, issueNum),
		})
	}

	return filteredNotifications
}

func (p *Plugin) getPrsDetails(c *UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	var prList []*PRDetails
	if err := json.NewDecoder(r.Body).Decode(&prList); err != nil {
		c.Log.WithError(err).Warnf("Error decoding PRDetails JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	prDetails := make([]*PRDetails, len(prList))
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

func (p *Plugin) fetchPRDetails(c *UserContext, client *github.Client, prURL string, prNumber int) *PRDetails {
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
	return &PRDetails{
		URL:                prURL,
		Number:             prNumber,
		Status:             status,
		Mergeable:          mergeable,
		RequestedReviewers: requestedReviewers,
		Reviews:            reviewsList,
	}
}

func fetchReviews(c *UserContext, client *github.Client, repoOwner string, repoName string, number int) ([]*github.PullRequestReview, error) {
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

func (p *Plugin) searchIssues(c *UserContext, w http.ResponseWriter, r *http.Request) {
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
	siteURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL

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

func (p *Plugin) createIssueComment(c *UserContext, w http.ResponseWriter, r *http.Request) {
	type CreateIssueCommentRequest struct {
		PostID              string `json:"post_id"`
		Owner               string `json:"owner"`
		Repo                string `json:"repo"`
		Number              int    `json:"number"`
		Comment             string `json:"comment"`
		ShowAttachedMessage bool   `json:"show_attached_message"`
	}

	req := &CreateIssueCommentRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding CreateIssueCommentRequest JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.PostID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid post id", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Owner == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repository owner.", StatusCode: http.StatusBadRequest})
		return
	}

	if req.Repo == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repository.", StatusCode: http.StatusBadRequest})
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

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	post, err := p.client.Post.GetPost(req.PostID)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	commentUsername, err := p.getUsername(post.UserId)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
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
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create an issue comment: %s", getFailReason(statusCode, req.Repo, currentUsername)), StatusCode: statusCode})
		return
	}

	rootID := req.PostID
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	permalinkReplyMessage := fmt.Sprintf("Comment attached to GitHub issue [#%v](%v)", req.Number, result.GetHTMLURL())
	if req.ShowAttachedMessage {
		permalinkReplyMessage = fmt.Sprintf("[Message](%v) attached to GitHub issue [#%v](%v)", permalink, req.Number, result.GetHTMLURL())
	}

	reply := &model.Post{
		Message:   permalinkReplyMessage,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		UserId:    c.UserID,
	}

	err = p.client.Post.CreatePost(reply)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeJSON(w, result)
}

func (p *Plugin) getLHSData(c *UserContext) (reviewResp []*github.Issue, assignmentResp []*github.Issue, openPRResp []*github.Issue, err error) {
	graphQLClient := p.graphQLConnect(c.GHInfo)

	reviewResp, assignmentResp, openPRResp, err = graphQLClient.GetLHSData(c.Context.Ctx)
	if err != nil {
		return []*github.Issue{}, []*github.Issue{}, []*github.Issue{}, err
	}

	return reviewResp, assignmentResp, openPRResp, nil
}

func (p *Plugin) getSidebarData(c *UserContext) (*SidebarContent, error) {
	reviewResp, assignmentResp, openPRResp, err := p.getLHSData(c)
	if err != nil {
		return nil, err
	}

	return &SidebarContent{
		PRs:         openPRResp,
		Assignments: assignmentResp,
		Reviews:     reviewResp,
		Unreads:     p.getUnreadsData(c),
	}, nil
}

func (p *Plugin) getSidebarContent(c *UserContext, w http.ResponseWriter, r *http.Request) {
	sidebarContent, err := p.getSidebarData(c)
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to search for the sidebar data")
		return
	}

	p.writeJSON(w, sidebarContent)
}

func (p *Plugin) postToDo(c *UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	username := c.GHInfo.GitHubUsername

	text, err := p.GetToDo(c.Ctx, username, githubClient)
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to get Todos")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	p.CreateBotDMPost(c.UserID, text, "custom_git_todo")

	resp := struct {
		Status string
	}{"OK"}

	p.writeJSON(w, resp)
}

func (p *Plugin) updateSettings(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var settings *UserSettings
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

func (p *Plugin) getIssueInfo(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue(ownerQueryParam)
	repo := r.FormValue(repoQueryParam)
	number := r.FormValue(numberQueryParam)
	postID := r.FormValue(postIDQueryParam)

	issueNumber, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	issue, _, err := githubClient.Issues.Get(c.Ctx, owner, repo, issueNumber)
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
		p.writeAPIError(w, &APIErrorResponse{Message: "Could not get the issue", StatusCode: http.StatusInternalServerError})
		return
	}

	description := ""
	if issue.Body != nil {
		description = mdCommentRegex.ReplaceAllString(issue.GetBody(), "")
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

	post, appErr := p.API.GetPost(postID)
	if appErr != nil {
		p.client.Log.Error("Unable to get the post", "PostID", postID, "Error", appErr.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", postID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", postID), StatusCode: http.StatusNotFound})
		return
	}

	issueInfo := map[string]interface{}{
		"title":            *issue.Title,
		"channel_id":       post.ChannelId,
		"postId":           postID,
		"milestone_title":  milestoneTitle,
		"milestone_number": milestoneNumber,
		"assignees":        assignees,
		"labels":           labels,
		"description":      description,
		"repo_full_name":   fmt.Sprintf("%s/%s", owner, repo),
		"issue_number":     *issue.Number,
	}

	p.writeJSON(w, issueInfo)
}

func (p *Plugin) getIssueByNumber(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue(ownerQueryParam)
	repo := r.FormValue(repoQueryParam)
	number := r.FormValue(numberQueryParam)
	issueNumber, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
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
		p.writeAPIError(w, &APIErrorResponse{Message: "Could not get the issue", StatusCode: http.StatusInternalServerError})
		return
	}

	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getPrByNumber(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue(ownerQueryParam)
	repo := r.FormValue(repoQueryParam)
	number := r.FormValue(numberQueryParam)

	prNumber, err := strconv.Atoi(number)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: "Invalid param 'number'.", StatusCode: http.StatusBadRequest})
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
		p.writeAPIError(w, &APIErrorResponse{Message: "Could not get pull request", StatusCode: http.StatusInternalServerError})
		return
	}
	if result.Body != nil {
		*result.Body = mdCommentRegex.ReplaceAllString(result.GetBody(), "")
	}
	p.writeJSON(w, result)
}

func (p *Plugin) getLabels(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allLabels []*github.Label
	opt := github.ListOptions{PerPage: 50}

	for {
		labels, resp, err := githubClient.Issues.ListLabels(c.Ctx, owner, repo, &opt)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list labels")
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

func (p *Plugin) getAssignees(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allAssignees []*github.User
	opt := github.ListOptions{PerPage: 50}

	for {
		assignees, resp, err := githubClient.Issues.ListAssignees(c.Ctx, owner, repo, &opt)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list assignees")
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

func (p *Plugin) getMilestones(c *UserContext, w http.ResponseWriter, r *http.Request) {
	owner, repo, err := parseRepo(r.URL.Query().Get("repo"))
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{Message: err.Error(), StatusCode: http.StatusBadRequest})
		return
	}

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	var allMilestones []*github.Milestone
	opt := github.ListOptions{PerPage: 50}

	for {
		milestones, resp, err := githubClient.Issues.ListMilestones(c.Ctx, owner, repo, &github.MilestoneListOptions{ListOptions: opt})
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list milestones")
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

func getRepositoryList(c context.Context, userName string, githubClient *github.Client, opt github.ListOptions) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	for {
		repos, resp, err := githubClient.Repositories.List(c, userName, &github.RepositoryListOptions{ListOptions: opt})
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

func getRepositoryListByOrg(c context.Context, org string, githubClient *github.Client, opt github.ListOptions) ([]*github.Repository, int, error) {
	var allRepos []*github.Repository
	for {
		repos, resp, err := githubClient.Repositories.ListByOrg(c, org, &github.RepositoryListByOrgOptions{Sort: "full_name", ListOptions: opt})
		if err != nil {
			return nil, resp.StatusCode, err
		}

		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, http.StatusOK, nil
}

func (p *Plugin) getRepositories(c *UserContext, w http.ResponseWriter, r *http.Request) {
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	org := p.getConfiguration().GitHubOrg

	var allRepos []*github.Repository
	var err error
	var statusCode int
	opt := github.ListOptions{PerPage: 50}

	if org == "" {
		allRepos, err = getRepositoryList(c.Ctx, "", githubClient, opt)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed to list repositories")
			p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
			return
		}
	} else {
		allRepos, statusCode, err = getRepositoryListByOrg(c.Ctx, org, githubClient, opt)
		if err != nil {
			if statusCode == http.StatusNotFound {
				allRepos, err = getRepositoryList(c.Ctx, org, githubClient, opt)
				if err != nil {
					c.Log.WithError(err).Warnf("Failed to list repositories")
					p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
					return
				}
			} else {
				c.Log.WithError(err).Warnf("Failed to list repositories")
				p.writeAPIError(w, &APIErrorResponse{Message: "Failed to fetch repositories", StatusCode: http.StatusInternalServerError})
				return
			}
		}
	}

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

func (p *Plugin) updateIssue(c *UserContext, w http.ResponseWriter, r *http.Request) {
	// get data for the issue from the request body and fill UpdateIssueRequest to update the issue
	issue := &UpdateIssueRequest{}
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if !p.validateIssueRequestForUpdation(issue, w) {
		return
	}

	var post *model.Post
	if issue.PostID != "" {
		var appErr *model.AppError
		post, appErr = p.API.GetPost(issue.PostID)
		if appErr != nil {
			p.client.Log.Error("Unable to get the post", "PostID", issue.PostID, "Error", appErr.Error())
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
			return
		}
		if post == nil {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", issue.PostID), StatusCode: http.StatusNotFound})
			return
		}
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
		p.client.Log.Error("Unable to get the user", "UserID", c.UserID, "Error", appErr.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	splittedRepo := strings.Split(issue.Repo, "/")
	if len(splittedRepo) < 2 {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repository", StatusCode: http.StatusBadRequest})
	}

	owner, repoName := splittedRepo[0], splittedRepo[1]
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	result, resp, err := githubClient.Issues.Edit(c.Ctx, owner, repoName, issue.IssueNumber, githubIssue)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusGone {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
			return
		}

		c.Log.WithError(err).Warnf("Failed to update the issue")
		p.writeAPIError(w, &APIErrorResponse{
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
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post, postID: %s, channelID: %s", issue.PostID, channelID), StatusCode: http.StatusInternalServerError})
		return
	}

	p.updatePost(issue, w)
	p.writeJSON(w, result)
}

func (p *Plugin) closeOrReopenIssue(c *UserContext, w http.ResponseWriter, r *http.Request) {
	type CommentAndCloseRequest struct {
		ChannelID    string `json:"channel_id"`
		IssueComment string `json:"issue_comment"`
		StatusReason string `json:"status_reason"`
		Number       int    `json:"number"`
		Owner        string `json:"owner"`
		Repository   string `json:"repo"`
		Status       string `json:"status"`
		PostID       string `json:"postId"`
	}

	req := &CommentAndCloseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil {
		p.client.Log.Error("Unable to get the post", "PostID", req.PostID, "Error", appErr.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", req.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", req.PostID), StatusCode: http.StatusNotFound})
		return
	}

	if _, err := p.getUsername(post.UserId); err != nil {
		p.client.Log.Error("Unable to get the username", "UserID", post.UserId, "Error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
		return
	}
	if req.IssueComment != "" {
		p.CreateCommentToIssue(c, w, req.IssueComment, req.Owner, req.Repository, post, req.Number)
	}

	if req.Status == statusClose {
		p.CloseOrReopenIssue(c, w, issueClose, req.StatusReason, req.Owner, req.Repository, post, req.Number)
	} else {
		p.CloseOrReopenIssue(c, w, issueOpen, req.StatusReason, req.Owner, req.Repository, post, req.Number)
	}
}

func (p *Plugin) createIssue(c *UserContext, w http.ResponseWriter, r *http.Request) {
	type CreateIssueRequest struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Repo      string   `json:"repo"`
		PostID    string   `json:"post_id"`
		ChannelID string   `json:"channel_id"`
		Labels    []string `json:"labels"`
		Assignees []string `json:"assignees"`
		Milestone int      `json:"milestone"`
	}

	// get data for the issue from the request body and fill CreateIssueRequest object to create the issue
	issue := &CreateIssueRequest{}
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		c.Log.WithError(err).Warnf("Error decoding the JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid JSON object.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Title == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid issue title.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.Repo == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid repository name.", StatusCode: http.StatusBadRequest})
		return
	}

	if issue.PostID == "" && issue.ChannelID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide either a postID or a channelID", StatusCode: http.StatusBadRequest})
		return
	}

	mmMessage := ""
	var post *model.Post
	permalink := ""
	if issue.PostID != "" {
		var err error
		post, err = p.client.Post.GetPost(issue.PostID)
		if err != nil {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
			return
		}
		if post == nil {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", issue.PostID), StatusCode: http.StatusNotFound})
			return
		}

		username, err := p.getUsername(post.UserId)
		if err != nil {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to get username", StatusCode: http.StatusInternalServerError})
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

	currentUser, err := p.client.User.Get(c.UserID)
	if err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to load current user", StatusCode: http.StatusInternalServerError})
		return
	}

	splittedRepo := strings.Split(issue.Repo, "/")
	owner, repoName := splittedRepo[0], splittedRepo[1]

	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	result, resp, err := githubClient.Issues.Create(c.Ctx, owner, repoName, githubIssue)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusGone {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
			return
		}

		c.Log.WithError(err).Warnf("Failed to create issue")
		p.writeAPIError(w, &APIErrorResponse{
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
		err = p.client.Post.CreatePost(reply)
	} else {
		p.client.Post.SendEphemeralPost(c.UserID, reply)
	}
	if err != nil {
		c.Log.WithError(err).Warnf("failed to create notification post")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "failed to create notification post, postID: " + issue.PostID + ", channelID: " + channelID, StatusCode: http.StatusInternalServerError})
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

func (p *Plugin) handleOpenEditIssueModal(c *UserContext, w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest params", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	p.client.Frontend.PublishWebSocketEvent(
		WebsocketEventOpenEditModal,
		map[string]interface{}{
			KeyRepoName:    postActionIntegrationRequest.Context[KeyRepoName],
			KeyRepoOwner:   postActionIntegrationRequest.Context[KeyRepoOwner],
			KeyIssueNumber: postActionIntegrationRequest.Context[KeyIssueNumber],
			KeyPostID:      postActionIntegrationRequest.PostId,
			KeyStatus:      postActionIntegrationRequest.Context[KeyStatus],
			KeyChannelID:   postActionIntegrationRequest.ChannelId,
		},
		&model.WebsocketBroadcast{UserId: postActionIntegrationRequest.UserId},
	)

	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) returnPostActionIntegrationResponse(w http.ResponseWriter, res *model.PostActionIntegrationResponse) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		p.API.LogWarn("Failed to write PostActionIntegrationResponse", "Error", err.Error())
	}
}

func (p *Plugin) handleOpenIssueStatusModal(c *UserContext, w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest params", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	p.client.Frontend.PublishWebSocketEvent(
		WebsocketEventOpenStatusModal,
		map[string]interface{}{
			KeyRepoName:    postActionIntegrationRequest.Context[KeyRepoName],
			KeyRepoOwner:   postActionIntegrationRequest.Context[KeyRepoOwner],
			KeyIssueNumber: postActionIntegrationRequest.Context[KeyIssueNumber],
			KeyPostID:      postActionIntegrationRequest.PostId,
			KeyStatus:      postActionIntegrationRequest.Context[KeyStatus],
			KeyChannelID:   postActionIntegrationRequest.ChannelId,
		},
		&model.WebsocketBroadcast{UserId: postActionIntegrationRequest.UserId},
	)

	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handleOpenIssueCommentModal(c *UserContext, w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest params", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	p.client.Frontend.PublishWebSocketEvent(
		WebsocketEventOpenCommentModal,
		map[string]interface{}{
			KeyRepoName:    postActionIntegrationRequest.Context[KeyRepoName],
			KeyRepoOwner:   postActionIntegrationRequest.Context[KeyRepoOwner],
			KeyIssueNumber: postActionIntegrationRequest.Context[KeyIssueNumber],
			KeyPostID:      postActionIntegrationRequest.PostId,
			KeyStatus:      postActionIntegrationRequest.Context[KeyStatus],
			KeyChannelID:   postActionIntegrationRequest.ChannelId,
		},
		&model.WebsocketBroadcast{UserId: postActionIntegrationRequest.UserId},
	)

	p.returnPostActionIntegrationResponse(w, response)
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
