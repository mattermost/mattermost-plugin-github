package plugin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"
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
	getConfiguration func() *Configuration
	getGitHubClient  func(ctx context.Context, userID string) (*github.Client, error)
	pluginURL        string

	pingBroker PingBroker
	tracker    telemetry.Tracker

	logger logger.Logger
	poster poster.Poster
	store  flow.Store

	setupController        flow.Controller
	oauthController        flow.Controller
	webhokController       flow.Controller
	announcementController flow.Controller
}

func (p *Plugin) NewFlowManager() *FlowManager {
	fm := &FlowManager{
		client:           p.client,
		getConfiguration: p.getConfiguration,
		getGitHubClient:  p.GetGitHubClient,
		pingBroker:       p.webhookManager,
		tracker:          p.tracker,
		pluginURL:        *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id,
		logger:           p.log,
		poster:           poster.NewPoster(&p.client.Post, p.BotUserID),
		store:            flow.NewFlowStore(&p.client.KV, "flow_store"),
	}

	setupSteps := append(fm.getOAuthSteps(), append(fm.getWebhookSteps(), fm.getAnnouncementSteps()...)...)
	setupSteps = append(setupSteps, steps.NewEmptyStep("final", "", ":tada: You successfully installed GitHub."))
	fm.setupController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow("setup", setupSteps, nil))

	fm.oauthController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow("wizard", fm.getOAuthSteps(), nil))
	fm.webhokController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow("webhook", fm.getWebhookSteps(), nil))
	fm.announcementController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow("announcement", fm.getAnnouncementSteps(), nil))

	return fm
}

func (fm *FlowManager) newController(router *mux.Router, frontend *pluginapi.FrontendService, f flow.Flow) flow.Controller {
	return flow.NewFlowController(
		fm.logger,
		router,
		fm.poster,
		frontend,
		fm.pluginURL,
		f,
		fm.store,
	)
}

func (fm *FlowManager) cancelFlow(userID string) int {
	_, err := fm.poster.DMWithAttachments(userID, &model.SlackAttachment{
		Text:  fmt.Sprintf("GitHub integration setup has stopped. Restart setup later by running `/github setup`. Learn more about the plugin [here](%s).", Manifest.HomepageURL),
		Color: string(steps.Danger),
	})
	if err != nil {
		fm.logger.WithError(err).Warnf("Failed to DM with cancel information")
	}

	return 999
}

func (fm *FlowManager) StartSetupWizard(userID string, fromInvite bool) error {
	err := fm.oauthController.Start(userID)
	if err != nil {
		return err
	}

	fm.trackStartSetupWizard(userID, fromInvite)

	return nil
}

func (fm *FlowManager) trackStartSetupWizard(userID string, fromInvite bool) {
	_ = fm.tracker.TrackUserEvent("setup_wizard_start", userID, map[string]interface{}{
		"from_invite": fromInvite,
		"time":        model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteSetupWizard(userID string) { //nolint:unused // TODO: call this somewhere
	_ = fm.tracker.TrackUserEvent("setup_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) StartOauthWizard(userID string) error {
	err := fm.oauthController.Start(userID)
	if err != nil {
		return err
	}

	fm.trackStartOauthizard(userID)

	return nil
}

func (fm *FlowManager) trackStartOauthizard(userID string) {
	_ = fm.tracker.TrackUserEvent("oauthwizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteOauthWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("oauth_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) getOAuthSteps() []steps.Step {
	config := fm.getConfiguration()
	usePreregisteredApplication := fm.getConfiguration().UsePreregisteredApplication

	welcomePretext := ":wave: Welcome to GitHub integration! [Learn more](https://github.com/mattermost/mattermost-plugin-github#readme)"

	var welcomeText string
	if usePreregisteredApplication {
		welcomeText = `
Just a few more configuration steps to go!
- **Step 1:** Connect your GitHub account
- **Step 2:** Create a webhook in GitHub`
	} else {
		welcomeText = `
Just a few more configuration steps to go!
- **Step 1:** Register an OAuth application in GitHub and enter OAuth values.
- **Step 2:** Connect your GitHub account
- **Step 3:** Create a webhook in GitHub`
	}

	welcomeStep := steps.NewCustomStepBuilder("welcome", "", welcomeText).
		WithPretext(welcomePretext).
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				if usePreregisteredApplication {
					return 4
				}

				return 0
			},
		}).
		Build()

	handoverQuestionText := "Are you setting this GitHub integration up, or is someone else?"
	handoverQuestionStep := steps.NewCustomStepBuilder("handoverQuestion", "", handoverQuestionText).
		WithButton(steps.Button{
			Name:  "I'll do it myself",
			Style: steps.Primary,
		}).
		WithButton(steps.Button{
			Name:  "I need someone else",
			Style: steps.Default,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:       "Send instructions to",
					SubmitLabel: "Send",
					Elements: []model.DialogElement{
						{
							DisplayName: "", // TODO: This will still show a *
							Name:        "aider",
							Type:        "select",
							DataSource:  "users",
						},
					},
				},
				OnDialogSubmit: fm.submitHandoverSelection,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		Build()

	enterpriseText := "Do you have GitHub Enterprise account?"
	enterpriseStep := steps.NewCustomStepBuilder("enterprise", "", enterpriseText).
		WithButton(steps.Button{
			Name:  "Yes",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "Enterprise account",
					IntroductionText: "Please enter the **Enterprise Base URL** and **Enterprise Upload URL**. Set these values to your GitHub Enterprise URLs, e.g. https://github.example.com. The Base and Upload URLs are often the same.",
					SubmitLabel:      "Save & Continue",
					Elements: []model.DialogElement{
						{

							DisplayName: "", // TODO: This will still show a *
							Name:        "base_url",
							Type:        "text",
							SubType:     "text",
							Placeholder: "Enterprise Base URL",
						},
						{
							DisplayName: "", // TODO: This will still show a *
							Name:        "upload_url",
							Type:        "text",
							SubType:     "text",
							Placeholder: "Enterprise Upload URL",
						},
					},
				},
				OnDialogSubmit: fm.submitEnterpriseConfig,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "No",
			Style: steps.Default,
		}).
		WithButton(steps.Button{
			Name:    "Cancel setup",
			Style:   steps.Danger,
			OnClick: fm.cancelFlow,
		}).
		Build()

	oauthPretext := `
##### :white_check_mark: Step 1: Register an OAuth Application in GitHub
You must first register the Mattermost GitHub Plugin as an authorized OAuth app.`
	oauthMessage := fmt.Sprintf(""+
		"1. In a browser, go to %ssettings/applications/new.\n"+
		"2. Set the following values:\n"+
		"	- Application name: `Mattermost GitHub Plugin - <your company name>`\n"+
		"	- Homepage URL: `https://github.com/mattermost/mattermost-plugin-github`\n"+
		"	- Authorization callback URL: `%s/oauth/complete`\n"+
		"3. Select submit\n"+
		"4. Select **Generate a new client secret**.\n"+
		"5. Enter your **GitHub password**.",
		config.getBaseURL(),
		fm.pluginURL,
	)

	oauthInfoStep := steps.NewCustomStepBuilder("oauthInfo", "", oauthMessage).
		WithPretext(oauthPretext).
		WithImage("public/new-oauth-application.png").
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
		}).
		WithButton(steps.Button{
			Name:    "Cancel setup",
			Style:   steps.Danger,
			OnClick: fm.cancelFlow,
		}).
		Build()

	oauthInputStep := steps.NewCustomStepBuilder("oauth-input", "", "Please enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret**.").
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "GitHub Oauth values",
					IntroductionText: "Please enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret** you copied in a previous step.",
					SubmitLabel:      "Save & Continue",
					Elements: []model.DialogElement{
						{
							DisplayName: "", // TODO: This will still show a *
							Name:        "client_id",
							Type:        "text",
							SubType:     "text",
							Placeholder: "GitHub OAuth Client ID",
						},
						{
							DisplayName: "", // TODO: This will still show a *
							Name:        "client_secret",
							Type:        "text",
							SubType:     "text",
							Placeholder: "GitHub OAuth Client Secret",
						},
					},
				},
				OnDialogSubmit: fm.submitOAuthConfig,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:    "Cancel setup",
			Style:   steps.Danger,
			OnClick: fm.cancelFlow,
		}).
		Build()

	var stepNumber int
	if usePreregisteredApplication {
		stepNumber = 1
	} else {
		stepNumber = 2
	}

	connectPretext := fmt.Sprintf("##### :white_check_mark: Step %d: Connect your GitHub account", stepNumber)
	connectURL := fmt.Sprintf("%s/oauth/connect", fm.pluginURL)
	connectText := fmt.Sprintf("Go [here](%s) to connect your account.", connectURL)
	conntectStep := steps.NewCustomStepBuilder("connect", "", connectText).
		WithPretext(connectPretext).
		IsNotEmpty(). // The API handler will advance to the next step and complete the flow
		Build()

	steps := []steps.Step{
		welcomeStep,
		handoverQuestionStep,
		enterpriseStep,
		oauthInfoStep,
		oauthInputStep,
		conntectStep,
	}

	return steps
}

func (fm *FlowManager) submitHandoverSelection(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	aiderIDRaw, ok := submission["aider"]
	if !ok {
		return 0, nil, "aider missing", nil
	}
	aiderID, ok := aiderIDRaw.(string)
	if !ok {
		return 0, nil, "aider is not a string", nil
	}

	aider, err := fm.client.User.Get(aiderID)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed get user").Error(), nil
	}

	err = fm.StartSetupWizard(aider.Id, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed start configration wizzard").Error(), nil
	}

	attachment := &model.SlackAttachment{
		Text: fmt.Sprintf("GitHub integration setup details have been sent to @%s", aider.Username),
	}
	_, err = fm.poster.DMWithAttachments(userID, attachment)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed send confirmation post").Error(), nil
	}

	return 999, nil, "", nil
}

func (fm *FlowManager) submitEnterpriseConfig(_ string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	errorList := map[string]string{}

	baseURLRaw, ok := submission["base_url"]
	if !ok {
		return 0, nil, "base_url missing", nil
	}
	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return 0, nil, "base_url is not a string", nil
	}

	err := isValidURL(baseURL)
	if err != nil {
		errorList["base_url"] = err.Error()
	}

	uploadURLRaw, ok := submission["upload_url"]
	if !ok {
		return 0, nil, "upload_url missing", nil
	}
	uploadURL, ok := uploadURLRaw.(string)
	if !ok {
		return 0, nil, "upload_url is not a string", nil
	}

	err = isValidURL(uploadURL)
	if err != nil {
		errorList["upload_url"] = err.Error()
	}

	if len(errorList) != 0 {
		return 0, nil, "", errorList
	}

	config := fm.getConfiguration()
	config.EnterpriseBaseURL = baseURL
	config.EnterpriseUploadURL = uploadURL

	err = fm.client.Configuration.SavePluginConfig(config.toMap())
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to save plugin config").Error(), nil
	}

	return 0, nil, "", nil
}

func (fm *FlowManager) submitOAuthConfig(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	errorList := map[string]string{}

	clientIDRaw, ok := submission["client_id"]
	if !ok {
		return 0, nil, "client_id missing", nil
	}
	clientID, ok := clientIDRaw.(string)
	if !ok {
		return 0, nil, "client_id is not a string", nil
	}

	if len(clientID) != 20 {
		errorList["client_id"] = "Client ID should be 20 characters long"
	}

	clientSecretRaw, ok := submission["client_secret"]
	if !ok {
		return 0, nil, "client_secret missing", nil
	}
	clientSecret, ok := clientSecretRaw.(string)
	if !ok {
		return 0, nil, "client_secret is not a string", nil
	}

	if len(clientSecret) != 40 {
		errorList["client_secret"] = "Client Secret should be 40 characters long"
	}

	if len(errorList) != 0 {
		return 0, nil, "", errorList
	}

	config := fm.getConfiguration()
	config.GitHubOAuthClientID = clientID
	config.GitHubOAuthClientSecret = clientSecret

	err := fm.client.Configuration.SavePluginConfig(config.toMap())
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to save plugin config").Error(), nil
	}

	fm.trackCompleteOauthWizard(userID)

	return 0, nil, "", nil
}

func (fm *FlowManager) StartWebhookWizard(userID string) error {
	err := fm.webhokController.Start(userID)
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

func (fm *FlowManager) getWebhookSteps() []steps.Step {
	var stepNumber int
	if fm.getConfiguration().UsePreregisteredApplication {
		stepNumber = 2
	} else {
		stepNumber = 3
	}

	questionPretext := fmt.Sprintf(`##### :white_check_mark: Step %d: Create a Webhook in GitHub
Our final setup step requires a Mattermost System Admin to create a webhook for each GitHub organization or repository you want to receive notifications for, or want to subscribe to.`, stepNumber)
	questionStep := steps.NewCustomStepBuilder("webhook-question", "", "Do you want to create a Webhook?").
		WithPretext(questionPretext).
		WithButton(steps.Button{
			Name:  "Yes",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:       "Create Webhook",
					SubmitLabel: "Create",
					Elements: []model.DialogElement{
						{

							DisplayName: "", // TODO: This will still show a *
							Name:        "repo_org",
							Type:        "text",
							SubType:     "text",
							Placeholder: "Repository or organization name, e.g. mattermost/mattermost-server",
							HelpText:    "Specify repository or organization for which you want to create a webhook",
						},
					},
				},
				OnDialogSubmit: fm.submitWebhook,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "No",
			Style: steps.Default,
			OnClick: func(userID string) int {
				return 1
			},
		}).
		WithButton(steps.Button{
			Name:    "Cancel setup",
			Style:   steps.Danger,
			OnClick: fm.cancelFlow,
		}).
		Build()

	confirmationStep := steps.NewEmptyStep("success", "Success! :tada: You've successfully set up your Mattermost GitHub integration! ", "Use `/github subscriptions add` to subscribe any Mattermost channel to your GitHub repository. [Learn more](https://example.org)")

	steps := []steps.Step{
		questionStep,
		confirmationStep,
	}

	return steps
}

func (fm *FlowManager) submitWebhook(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	repoOrgRaw, ok := submission["repo_org"]
	if !ok {
		return 0, nil, "repo_org missing", nil
	}
	repoOrg, ok := repoOrgRaw.(string)
	if !ok {
		return 0, nil, "repo_org is not a string", nil
	}

	org, repo := parseOwnerAndRepo(repoOrg, fm.getConfiguration().getBaseURL())
	if org == "" && repo == "" {
		return 0, nil, "Invalid format", nil
	}

	webhookEvents := []string{"create", "delete", "issue_comment", "issues", "pull_request", "pull_request_review", "pull_request_review_comment", "push", "star"}

	config := map[string]interface{}{
		"content_type": "json",
		"insecure_ssl": "0",
		"secret":       fm.getConfiguration().WebhookSecret,
		"url":          fmt.Sprintf("%s/webhook", fm.pluginURL),
	}

	hook := &github.Hook{
		Events: webhookEvents,
		Config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second) // HTTP request times out after 30 seconds
	defer cancel()

	client, apiErr := fm.getGitHubClient(ctx, userID)
	if apiErr != nil {
		return 0, nil, apiErr.Error(), nil
	}

	ch := fm.pingBroker.SubscribePings()

	var err error
	var resp *github.Response
	var fullName string
	if repo == "" {
		fullName = org
		hook, resp, err = client.Organizations.CreateHook(ctx, org, hook)
	} else {
		fullName = org + "/" + repo
		hook, resp, err = client.Repositories.CreateHook(ctx, org, repo, hook)
	}

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil, fmt.Sprintf("It seems like you don't have access %s. Please ask an administrator of that repository to run /github setup webhook for you.", fullName), nil
	}

	if err != nil {
		var errResp *github.ErrorResponse
		if errors.As(err, &errResp) {
			return 0, nil, printGithubErrorResponse(errResp), nil
		}

		return 0, nil, errors.Wrap(err, "failed to create hook").Error(), nil
	}

	select {
	case event := <-ch:
		if *event.HookID == *hook.ID {
			break
		}
		ctx.Deadline()
	case <-ctx.Done():
		return 0, nil, "Timed out waiting for webhook event. Please check if the webhook was corrected created.", nil
	}

	fm.pingBroker.UnsubscribePings(ch)

	fm.trackCompleteWebhookWizard(userID)

	return 0, nil, "", nil
}

func (fm *FlowManager) StartAnnouncementWizard(userID string) error {
	err := fm.announcementController.Start(userID)
	if err != nil {
		return err
	}

	fm.trackStartAnnouncementWizard(userID)

	return nil
}

func (fm *FlowManager) getAnnouncementSteps() []steps.Step {
	defaultMessage := "Hi team,\n" +
		"\n" +
		"We've set up the Mattermost GitHub plugin, so you can get notifications from GitHub in Mattermost. To get started, run the `/github connect` slash command from any channel within Mattermost to connect your Mattermost account with GitHub. Then, take a look at the slash commands section for details about how to use the plugin."
	questionStep := steps.NewCustomStepBuilder("announcement-question", "", "Want to let your team know?").
		WithButton(steps.Button{
			Name:  "Send Message",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:       "Notify your team",
					SubmitLabel: "Send message",
					Elements: []model.DialogElement{
						{
							DisplayName: "", // TODO: This will still show a *
							Name:        "channel_id",
							Type:        "select",
							Placeholder: "Select channel",
							DataSource:  "channels",
						},
						{
							DisplayName: "",
							Name:        "message",
							Type:        "textarea",
							Default:     defaultMessage,
							HelpText:    "You can edit this message before sending it.",
						},
					},
				},
				OnDialogSubmit: fm.submitChannelAnnouncement,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Not now",
			Style: steps.Default,
		}).
		Build()

	steps := []steps.Step{
		questionStep,
	}

	return steps
}

func (fm *FlowManager) submitChannelAnnouncement(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	channelIDRaw, ok := submission["channel_id"]
	if !ok {
		return 0, nil, "channel_id missing", nil
	}
	channelID, ok := channelIDRaw.(string)
	if !ok {
		return 0, nil, "channel_id is not a string", nil
	}

	channel, err := fm.client.Channel.Get(channelID)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get channel").Error(), nil
	}

	messageRaw, ok := submission["message"]
	if !ok {
		return 0, nil, "message missing", nil
	}
	message, ok := messageRaw.(string)
	if !ok {
		return 0, nil, "message is not a string", nil
	}

	post := &model.Post{
		UserId:    userID,
		ChannelId: channel.Id,
		Message:   message,
	}
	err = fm.client.Post.CreatePost(post)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to create post").Error(), nil
	}

	attachment := &model.SlackAttachment{
		Text: fmt.Sprintf("Message to ~%s was sent.", channel.Name),
	}
	_, err = fm.poster.DMWithAttachments(userID, attachment)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed send confirmation post").Error(), nil
	}

	fm.trackCompletAnnouncementWizard(userID)

	return 0, nil, "", nil
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

func printGithubErrorResponse(err *github.ErrorResponse) string {
	msg := err.Message
	for _, err := range err.Errors {
		msg += ", " + err.Message
	}
	return msg
}
