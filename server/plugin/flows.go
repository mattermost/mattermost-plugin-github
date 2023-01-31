package plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"

	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"
)

type PingBroker interface {
	UnsubscribePings(ch <-chan *github.PingEvent)
	SubscribePings() <-chan *github.PingEvent
}

type FlowManager struct {
	client           *pluginapi.Client
	pluginURL        string
	botUserID        string
	router           *mux.Router
	getConfiguration func() *Configuration
	getGitHubClient  func(ctx context.Context, userID string) (*github.Client, error)

	pingBroker PingBroker
	tracker    telemetry.Tracker

	setupFlow        *flow.Flow
	oauthFlow        *flow.Flow
	webhokFlow       *flow.Flow
	announcementFlow *flow.Flow
}

func (p *Plugin) NewFlowManager() *FlowManager {
	fm := &FlowManager{
		client:           p.client,
		pluginURL:        *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id,
		botUserID:        p.BotUserID,
		router:           p.router,
		getConfiguration: p.getConfiguration,
		getGitHubClient:  p.GetGitHubClient,

		pingBroker: p.webhookBroker,
		tracker:    p.tracker,
	}

	fm.setupFlow = fm.newFlow("setup").WithSteps(
		fm.stepWelcome(),

		fm.stepDelegateQuestion(),
		fm.stepDelegateConfirmation(),
		fm.stepDelegateComplete(),

		fm.stepEnterprise(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInput(),
		fm.stepOAuthConnect(),

		fm.stepWebhookQuestion(),
		fm.stepWebhookWarning(),
		fm.stepWebhookConfirmation(),

		fm.stepAnnouncementQuestion(),
		fm.stepAnnouncementConfirmation(),

		fm.doneStep(),

		fm.stepCancel("setup"),
	)

	fm.oauthFlow = fm.newFlow("oauth").WithSteps(
		fm.stepEnterprise(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInput(),
		fm.stepOAuthConnect().Terminal(),

		fm.stepCancel("setup oauth"),
	)
	fm.webhokFlow = fm.newFlow("webhook").WithSteps(
		fm.stepWebhookQuestion(),
		flow.NewStep(stepWebhookConfirmation).
			WithText("Use `/github subscriptions add` to subscribe any Mattermost channel to your GitHub repository. [Learn more](https://github.com/mattermost/mattermost-plugin-github#slash-commands)").
			Terminal(),

		fm.stepCancel("setup webhook"),
	)
	fm.announcementFlow = fm.newFlow("announcement").WithSteps(
		fm.stepAnnouncementQuestion(),
		fm.stepAnnouncementConfirmation().Terminal(),

		fm.stepCancel("setup announcement"),
	)

	return fm
}

func (fm *FlowManager) doneStep() flow.Step {
	return flow.NewStep(stepDone).
		WithText(":tada: You successfully installed GitHub.").
		OnRender(fm.onDone).Terminal()
}

func (fm *FlowManager) onDone(f *flow.Flow) {
	fm.trackCompleteSetupWizard(f.UserID)

	delegatedFrom := f.GetState().GetString(keyDelegatedFrom)
	if delegatedFrom != "" {
		err := fm.setupFlow.ForUser(delegatedFrom).Go(stepDelegateComplete)
		fm.client.Log.Warn("failed start configuration wizard for delegate", "error", err)
	}
}

func (fm *FlowManager) newFlow(name flow.Name) *flow.Flow {
	flow := flow.NewFlow(
		name,
		fm.client,
		fm.pluginURL,
		fm.botUserID,
	)

	flow.InitHTTP(fm.router)

	return flow
}

const (
	// Delegate Steps

	stepDelegateQuestion     flow.Name = "delegate-question"
	stepDelegateConfirmation flow.Name = "delegate-confirmation"
	stepDelegateComplete     flow.Name = "delegate-complete"

	// OAuth steps

	stepEnterprise   flow.Name = "enterprise"
	stepOAuthInfo    flow.Name = "oauth-info"
	stepOAuthInput   flow.Name = "oauth-input"
	stepOAuthConnect flow.Name = "oauth-connect"

	// Webhook steps

	stepWebhookQuestion     flow.Name = "webhook-question"
	stepWebhookWarning      flow.Name = "webhook-warning"
	stepWebhookConfirmation flow.Name = "webhook-confirmation"

	// Announcement steps

	stepAnnouncementQuestion     flow.Name = "announcement-question"
	stepAnnouncementConfirmation flow.Name = "announcement-confirmation"

	// Miscellaneous Steps

	stepWelcome flow.Name = "welcome"
	stepDone    flow.Name = "done"
	stepCancel  flow.Name = "cancel"

	keyDelegatedFrom               = "DelegatedFrom"
	keyDelegatedTo                 = "DelegatedTo"
	keyBaseURL                     = "BaseURL"
	keyUsePreregisteredApplication = "UsePreregisteredApplication"
	keyIsOAuthConfigured           = "IsOAuthConfigured"
)

func cancelButton() flow.Button {
	return flow.Button{
		Name:    "Cancel setup",
		Color:   flow.ColorDanger,
		OnClick: flow.Goto(stepCancel),
	}
}

func (fm *FlowManager) stepCancel(command string) flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithText(fmt.Sprintf("GitHub integration setup has stopped. Restart setup later by running `/github %s`. Learn more about the plugin [here](%s).", command, Manifest.HomepageURL)).
		WithColor(flow.ColorDanger)
}

func continueButtonF(f func(f *flow.Flow) (flow.Name, flow.State, error)) flow.Button {
	return flow.Button{
		Name:    "Continue",
		Color:   flow.ColorPrimary,
		OnClick: f,
	}
}

func continueButton(next flow.Name) flow.Button {
	return continueButtonF(flow.Goto(next))
}

func (fm *FlowManager) getBaseState() flow.State {
	config := fm.getConfiguration()
	isOAuthConfigured := config.GitHubOAuthClientID != "" || config.GitHubOAuthClientSecret != ""
	return flow.State{
		keyBaseURL:                     config.getBaseURL(),
		keyUsePreregisteredApplication: config.UsePreregisteredApplication,
		keyIsOAuthConfigured:           isOAuthConfigured,
	}
}

func (fm *FlowManager) StartSetupWizard(userID string, delegatedFrom string) error {
	state := fm.getBaseState()
	state[keyDelegatedFrom] = delegatedFrom

	err := fm.setupFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartSetupWizard(userID, delegatedFrom != "")

	return nil
}

func (fm *FlowManager) trackStartSetupWizard(userID string, fromInvite bool) {
	_ = fm.tracker.TrackUserEvent("setup_wizard_start", userID, map[string]interface{}{
		"from_invite": fromInvite,
		"time":        model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteSetupWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("setup_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) StartOauthWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.oauthFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartOauthWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartOauthWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("oauth_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteOauthWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("oauth_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepWelcome() flow.Step {
	welcomePretext := ":wave: Welcome to your GitHub integration! [Learn more](https://github.com/mattermost/mattermost-plugin-github#readme)"

	welcomeText := `
{{- if .UsePreregisteredApplication -}}
Just a few configuration steps to go!
- **Step 1:** Connect your GitHub account
- **Step 2:** Create a webhook in GitHub
{{- else -}}
Just a few configuration steps to go!
- **Step 1:** Register an OAuth application in GitHub and enter OAuth values.
- **Step 2:** Connect your GitHub account
- **Step 3:** Create a webhook in GitHub
{{- end -}}`

	return flow.NewStep(stepWelcome).
		WithText(welcomeText).
		WithPretext(welcomePretext).
		WithButton(continueButton(""))
}

func (fm *FlowManager) stepDelegateQuestion() flow.Step {
	delegateQuestionText := "Are you setting this GitHub integration up, or is someone else?"
	return flow.NewStep(stepDelegateQuestion).
		WithText(delegateQuestionText).
		WithButton(flow.Button{
			Name:  "I'll do it myself",
			Color: flow.ColorPrimary,
			OnClick: func(f *flow.Flow) (flow.Name, flow.State, error) {
				if f.GetState().GetBool(keyUsePreregisteredApplication) {
					return stepOAuthConnect, nil, nil
				}

				return stepEnterprise, nil, nil
			},
		}).
		WithButton(flow.Button{
			Name:  "I need someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions",
				SubmitLabel: "Send",
				Elements: []model.DialogElement{
					{
						DisplayName: "To",
						Name:        "delegate",
						Type:        "select",
						DataSource:  "users",
						Placeholder: "Search for people",
					},
				},
			},
			OnDialogSubmit: fm.submitDelegateSelection,
		})
}

func (fm *FlowManager) submitDelegateSelection(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	delegateIDRaw, ok := submitted["delegate"]
	if !ok {
		return "", nil, nil, errors.New("delegate missing")
	}
	delegateID, ok := delegateIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("delegate is not a string")
	}

	delegate, err := fm.client.User.Get(delegateID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed get user")
	}

	err = fm.StartSetupWizard(delegate.Id, f.UserID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed start configuration wizard")
	}

	return stepDelegateConfirmation, flow.State{
		keyDelegatedTo: delegate.Username,
	}, nil, nil
}

func (fm *FlowManager) stepDelegateConfirmation() flow.Step {
	return flow.NewStep(stepDelegateConfirmation).
		WithText("GitHub integration setup details have been sent to @{{ .DelegatedTo }}").
		WithButton(flow.Button{
			Name:     "Waiting for @{{ .DelegatedTo }}...",
			Color:    flow.ColorDefault,
			Disabled: true,
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) stepDelegateComplete() flow.Step {
	return flow.NewStep(stepDelegateComplete).
		WithText("@{{ .DelegatedTo }} completed configuring the integration.").
		Next(stepDone)
}

func (fm *FlowManager) stepEnterprise() flow.Step {
	enterpriseText := "Do you have a GitHub Enterprise account?"
	return flow.NewStep(stepEnterprise).
		WithText(enterpriseText).
		WithButton(flow.Button{
			Name:  "Yes",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "Enterprise account",
				IntroductionText: "Enter an **Enterprise Base URL** and **Enterprise Upload URL** by setting these values to match your GitHub Enterprise URL (Example: https://github.example.com). It's not necessary to have separate Base and Upload URLs.",
				SubmitLabel:      "Save & continue",
				Elements: []model.DialogElement{
					{

						DisplayName: "Enterprise Base URL",
						Name:        "base_url",
						Type:        "text",
						SubType:     "url",
						Placeholder: "Enter Enterprise Base URL",
					},
					{
						DisplayName: "Enterprise Upload URL",
						Name:        "upload_url",
						Type:        "text",
						SubType:     "url",
						Placeholder: "Enter Enterprise Upload URL",
					},
				},
			},
			OnDialogSubmit: fm.submitEnterpriseConfig,
		}).
		WithButton(flow.Button{
			Name:    "No",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepOAuthInfo),
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) submitEnterpriseConfig(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	errorList := map[string]string{}

	baseURLRaw, ok := submitted["base_url"]
	if !ok {
		return "", nil, nil, errors.New("base_url missing")
	}
	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("base_url is not a string")
	}

	baseURL = strings.TrimSpace(baseURL)

	err := isValidURL(baseURL)
	if err != nil {
		errorList["base_url"] = err.Error()
	}

	uploadURLRaw, ok := submitted["upload_url"]
	if !ok {
		return "", nil, nil, errors.New("upload_url missing")
	}
	uploadURL, ok := uploadURLRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("upload_url is not a string")
	}

	uploadURL = strings.TrimSpace(uploadURL)

	err = isValidURL(uploadURL)
	if err != nil {
		errorList["upload_url"] = err.Error()
	}

	if len(errorList) != 0 {
		return "", nil, errorList, nil
	}

	config := fm.getConfiguration()
	config.EnterpriseBaseURL = baseURL
	config.EnterpriseUploadURL = uploadURL
	config.sanitize()

	configMap, err := config.ToMap()
	if err != nil {
		return "", nil, nil, err
	}

	err = fm.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to save plugin config")
	}

	return "", flow.State{
		keyBaseURL: config.getBaseURL(),
	}, nil, nil
}

func (fm *FlowManager) stepOAuthInfo() flow.Step {
	oauthPretext := `
##### :white_check_mark: Step 1: Register an OAuth Application in GitHub
You must first register the Mattermost GitHub Plugin as an authorized OAuth app.`
	oauthMessage := fmt.Sprintf(""+
		"1. In a browser, go to {{ .BaseURL}}settings/applications/new.\n"+
		"2. Set the following values:\n"+
		"	- Application name: `Mattermost GitHub Plugin - <your company name>`\n"+
		"	- Homepage URL: `https://github.com/mattermost/mattermost-plugin-github`\n"+
		"	- Authorization callback URL: `%s/oauth/complete`\n"+
		"3. Select **Register application**\n"+
		"4. Select **Generate a new client secret**.\n"+
		"5. Enter your **GitHub password**. (if prompted)",
		fm.pluginURL,
	)

	return flow.NewStep(stepOAuthInfo).
		WithPretext(oauthPretext).
		WithText(oauthMessage).
		WithImage("public/new-oauth-application.png").
		WithButton(continueButton("")).
		WithButton(cancelButton())
}

func (fm *FlowManager) stepOAuthInput() flow.Step {
	return flow.NewStep(stepOAuthInput).
		WithText("Click the Continue button below to open a dialog to enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret**.").
		WithButton(flow.Button{
			Name:  "Continue",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "GitHub OAuth values",
				IntroductionText: "Please enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret** you copied in a previous step.{{ if .IsOAuthConfigured }}\n\n**Any existing OAuth configuration will be overwritten.**{{end}}",
				SubmitLabel:      "Save & continue",
				Elements: []model.DialogElement{
					{
						DisplayName: "GitHub OAuth Client ID",
						Name:        "client_id",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitHub OAuth Client ID",
					},
					{
						DisplayName: "GitHub OAuth Client Secret",
						Name:        "client_secret",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitHub OAuth Client Secret",
					},
				},
			},
			OnDialogSubmit: fm.submitOAuthConfig,
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) submitOAuthConfig(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	errorList := map[string]string{}

	clientIDRaw, ok := submitted["client_id"]
	if !ok {
		return "", nil, nil, errors.New("client_id missing")
	}
	clientID, ok := clientIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("client_id is not a string")
	}

	clientID = strings.TrimSpace(clientID)

	if len(clientID) != 20 {
		errorList["client_id"] = "Client ID should be 20 characters long"
	}

	clientSecretRaw, ok := submitted["client_secret"]
	if !ok {
		return "", nil, nil, errors.New("client_secret missing")
	}
	clientSecret, ok := clientSecretRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("client_secret is not a string")
	}

	clientSecret = strings.TrimSpace(clientSecret)

	if len(clientSecret) != 40 {
		errorList["client_secret"] = "Client Secret should be 40 characters long"
	}

	if len(errorList) != 0 {
		return "", nil, errorList, nil
	}

	config := fm.getConfiguration()
	config.GitHubOAuthClientID = clientID
	config.GitHubOAuthClientSecret = clientSecret

	configMap, err := config.ToMap()
	if err != nil {
		return "", nil, nil, err
	}

	err = fm.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to save plugin config")
	}

	return "", nil, nil, nil
}

func (fm *FlowManager) stepOAuthConnect() flow.Step {
	connectPretext := "##### :white_check_mark: Step {{ if .UsePreregisteredApplication }}1{{ else }}2{{ end }}: Connect your GitHub account"
	connectURL := fmt.Sprintf("%s/oauth/connect", fm.pluginURL)
	connectText := fmt.Sprintf("Go [here](%s) to connect your account.", connectURL)
	return flow.NewStep(stepOAuthConnect).
		WithText(connectText).
		WithPretext(connectPretext).
		OnRender(func(f *flow.Flow) { fm.trackCompleteOauthWizard(f.UserID) })
	// The API handler will advance to the next step and complete the flow
}

func (fm *FlowManager) StartWebhookWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.webhokFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartWebhookWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartWebhookWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("webhook_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteWebhookWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("webhook_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepWebhookQuestion() flow.Step {
	questionPretext := `##### :white_check_mark: Step {{ if .UsePreregisteredApplication }}2{{ else }}3{{ end }}: Create a Webhook in GitHub
The final setup step requires a Mattermost System Admin to create a webhook for each GitHub organization or repository to receive notifications for, or want to subscribe to.`
	return flow.NewStep(stepWebhookQuestion).
		WithText("Do you want to create a webhook?").
		WithPretext(questionPretext).
		WithButton(flow.Button{
			Name:  "Yes",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Create webhook",
				SubmitLabel: "Create",
				Elements: []model.DialogElement{
					{

						DisplayName: "GitHub repository or organization name",
						Name:        "repo_org",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitHub repository or organization name",
						HelpText:    "Specify the GitHub repository or organization to connect to Mattermost. For example, mattermost/mattermost-server.",
					},
				},
			},
			OnDialogSubmit: fm.submitWebhook,
		}).
		WithButton(flow.Button{
			Name:    "No",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepWebhookWarning),
		})
}

func (fm *FlowManager) submitWebhook(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	repoOrgRaw, ok := submitted["repo_org"]
	if !ok {
		return "", nil, nil, errors.New("repo_org missing")
	}
	repoOrg, ok := repoOrgRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("repo_org is not a string")
	}

	repoOrg = strings.TrimSpace(repoOrg)

	config := fm.getConfiguration()

	org, repo := parseOwnerAndRepo(repoOrg, config.getBaseURL())
	if org == "" && repo == "" {
		return "", nil, nil, errors.New("invalid format")
	}

	webhookEvents := []string{"create", "delete", "issue_comment", "issues", "pull_request", "pull_request_review", "pull_request_review_comment", "push", "star"}

	webhookConfig := map[string]interface{}{
		"content_type": "json",
		"insecure_ssl": "0",
		"secret":       config.WebhookSecret,
		"url":          fmt.Sprintf("%s/webhook", fm.pluginURL),
	}

	hook := &github.Hook{
		Events: webhookEvents,
		Config: webhookConfig,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second) // HTTP request times out after 30 seconds
	defer cancel()

	client, err := fm.getGitHubClient(ctx, f.UserID)
	if err != nil {
		return "", nil, nil, err
	}

	ch := fm.pingBroker.SubscribePings()

	var resp *github.Response
	var fullName string
	var repoOrOrg string
	if repo == "" {
		fullName = org
		repoOrOrg = "organization"
		hook, resp, err = client.Organizations.CreateHook(ctx, org, hook)
	} else {
		fullName = org + "/" + repo
		repoOrOrg = "repository"
		hook, resp, err = client.Repositories.CreateHook(ctx, org, repo, hook)
	}

	if resp.StatusCode == http.StatusNotFound {
		err = errors.Errorf("It seems like you don't have privileges to create webhooks in %s. Ask an admin of that %s to run /github setup webhook for you.", fullName, repoOrOrg)
		return "", nil, nil, err
	}

	if err != nil {
		var errResp *github.ErrorResponse
		if errors.As(err, &errResp) {
			return "", nil, nil, printGithubErrorResponse(errResp)
		}

		return "", nil, nil, errors.Wrap(err, "failed to create hook")
	}

	var found bool
	for !found {
		select {
		case event, ok := <-ch:
			if ok && event != nil && *event.HookID == *hook.ID {
				found = true
			}
		case <-ctx.Done():
			return "", nil, nil, errors.New("timed out waiting for webhook event. Please check if the webhook was correctly created")
		}
	}

	fm.pingBroker.UnsubscribePings(ch)

	return stepWebhookConfirmation, nil, nil, nil
}

func (fm *FlowManager) stepWebhookWarning() flow.Step {
	warnText := "The GitHub plugin uses a webhook to connect a GitHub account to Mattermost to listen for incoming GitHub events. " +
		"You can't subscribe a channel to a repository for notifications until webhooks are configured.\n" +
		"Restart setup later by running `/github setup webhook`"

	return flow.NewStep(stepWebhookWarning).
		WithText(warnText).
		WithColor(flow.ColorDanger).
		Next("")
}

func (fm *FlowManager) stepWebhookConfirmation() flow.Step {
	return flow.NewStep(stepWebhookConfirmation).
		WithTitle("Success! :tada: You've successfully set up your Mattermost GitHub integration! ").
		WithText("Use `/github subscriptions add` to subscribe any Mattermost channel to your GitHub repository. [Learn more](https://github.com/mattermost/mattermost-plugin-github#slash-commands)").
		OnRender(func(f *flow.Flow) { fm.trackCompleteWebhookWizard(f.UserID) }).
		Next("")
}

func (fm *FlowManager) StartAnnouncementWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.announcementFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartAnnouncementWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartAnnouncementWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("announcement_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompletAnnouncementWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("announcement_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepAnnouncementQuestion() flow.Step {
	defaultMessage := "Hi team,\n" +
		"\n" +
		"We've set up the Mattermost GitHub plugin to enable notifications from GitHub in Mattermost. To get started, run the `/github connect` slash command from any channel within Mattermost to connect that channel with GitHub. See the [documentation](https://github.com/mattermost/mattermost-plugin-github/blob/master/README.md#slash-commands) for details on using the GitHub plugin."

	return flow.NewStep(stepAnnouncementQuestion).
		WithText("Want to let your team know?").
		WithButton(flow.Button{
			Name:  "Send Message",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Notify your team",
				SubmitLabel: "Send message",
				Elements: []model.DialogElement{
					{
						DisplayName: "To",
						Name:        "channel_id",
						Type:        "select",
						Placeholder: "Select channel",
						DataSource:  "channels",
					},
					{
						DisplayName: "Message",
						Name:        "message",
						Type:        "textarea",
						Default:     defaultMessage,
						HelpText:    "You can edit this message before sending it.",
					},
				},
			},
			OnDialogSubmit: fm.submitChannelAnnouncement,
		}).
		WithButton(flow.Button{
			Name:    "Not now",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepDone),
		})
}

func (fm *FlowManager) submitChannelAnnouncement(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	channelIDRaw, ok := submitted["channel_id"]
	if !ok {
		return "", nil, nil, errors.New("channel_id missing")
	}
	channelID, ok := channelIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("channel_id is not a string")
	}

	channel, err := fm.client.Channel.Get(channelID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to get channel")
	}

	messageRaw, ok := submitted["message"]
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}
	message, ok := messageRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}

	post := &model.Post{
		UserId:    f.UserID,
		ChannelId: channel.Id,
		Message:   message,
	}
	err = fm.client.Post.CreatePost(post)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to create announcement post")
	}

	return stepAnnouncementConfirmation, flow.State{
		"ChannelName": channel.Name,
	}, nil, nil
}

func (fm *FlowManager) stepAnnouncementConfirmation() flow.Step {
	return flow.NewStep(stepAnnouncementConfirmation).
		WithText("Message to ~{{ .ChannelName }} was sent.").
		Next("").
		OnRender(func(f *flow.Flow) { fm.trackCompletAnnouncementWizard(f.UserID) })
}

func printGithubErrorResponse(err *github.ErrorResponse) error {
	msg := err.Message
	for _, err := range err.Errors {
		msg += ", " + err.Message
	}
	return errors.New(msg)
}
