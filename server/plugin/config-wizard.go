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

type propertyStore struct {
}

func (ps *propertyStore) SetProperty(userID, propertyName string, value interface{}) error {
	return nil
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

	oauthController        flow.Controller
	webhokController       flow.Controller
	announcementController flow.Controller
	fullWizardController   flow.Controller
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
		store:            flow.NewFlowStore(*p.client, "flow_store"),
	}

	fm.oauthController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow(fm.getOAuthSteps(), "/wizard", nil))
	fm.webhokController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow(fm.getWebhookSteps(), "/webhook", nil))
	fm.announcementController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow(fm.getAnnouncemenSteps(), "/announcement", nil))

	allSteps := append(fm.getOAuthSteps(), append(fm.getWebhookSteps(), fm.getAnnouncemenSteps()...)...)
	allSteps = append(allSteps, steps.NewEmptyStep("", ":tada: You successfully installed GitHub."))
	fm.fullWizardController = fm.newController(p.router, &p.client.Frontend, flow.NewFlow(allSteps, "/full-wizzard", nil))

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
		&propertyStore{},
	)
}

func (fm *FlowManager) cancelFlow(userID string) int {
	_, err := fm.poster.DMWithAttachments(userID, &model.SlackAttachment{
		Text: fmt.Sprintf("You can restart the wizard by running `/github setup`. Learn more about the plugin [here](%s)", Manifest.HomepageURL),
	})
	if err != nil {
		fm.logger.WithError(err).Warnf("Failed to DM with cancel information")
	}

	return 999
}

func (fm *FlowManager) StartConfigurationWizard(userID string, fromInvite bool) error {
	err := fm.oauthController.Start(userID)
	if err != nil {
		return err
	}

	fm.trackStartConfigurationWizard(userID, fromInvite)

	return nil
}

func (fm *FlowManager) trackStartConfigurationWizard(userID string, fromInvite bool) {
	_ = fm.tracker.TrackUserEvent("configuration_wizard_start", userID, map[string]interface{}{
		"from_invite": fromInvite,
		"time":        time.Now().UnixMilli(),
	})
}

func (fm *FlowManager) trackCompleteConfigurationWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("configuration_wizard_complete", userID, map[string]interface{}{
		"time": time.Now().UnixMilli(),
	})
}

func (fm *FlowManager) getOAuthSteps() []steps.Step {
	config := fm.getConfiguration()

	welcomeText := ":wave:  Welcome to GitHub for Mattermost! To begin using the integration, let’s complete the configuration. [Learn more](https://example.org)"
	welcomeStep := steps.NewCustomStepBuilder("", welcomeText).
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				if fm.getConfiguration().UsePreregisteredApplication {
					return 2
				}

				return 0
			},
		}).
		WithButton(steps.Button{
			Name:    "Not now",
			Style:   steps.Default,
			OnClick: fm.cancelFlow,
		}).
		Build()

	handoverQuestionText := "Do you need another person to finish setting GitHub up?"
	handoverQuestionStep := steps.NewCustomStepBuilder("", handoverQuestionText).
		WithButton(steps.Button{
			Name:  "I'll do it myself",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				return 1
			},
		}).
		WithButton(steps.Button{
			Name:  "I need someone else",
			Style: steps.Default,
		}).
		Build()

	handoverSelectionTitle := "Who are Mattermost teammate you'd like to send instructions to?"
	handoverSelectionText := "Add your teammates, and they will receive a message with all the instructions to complete the configuration."
	handoverSelectionStep := steps.NewCustomStepBuilder(handoverSelectionTitle, handoverSelectionText).
		WithButton(steps.Button{
			Name:  "Add teammate",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title: "TODO",
					//IntroductionText: "",
					SubmitLabel: "Add teammate",
					Elements: []model.DialogElement{
						{
							DisplayName: "Aider",
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

	enterpriseText := "Are you using GitHub Enterprise?"
	enterpriseStep := steps.NewCustomStepBuilder("", enterpriseText).
		WithButton(steps.Button{
			Name:  "Yes",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "TODO",
					IntroductionText: "Please enter the **Enterprise Base URL** and **Enterprise Upload URL**. Set these values to your GitHub Enterprise URLs, e.g. https://github.example.com. The Base and Upload URLs are often the same.",
					SubmitLabel:      "Continue",
					Elements: []model.DialogElement{
						{
							DisplayName: "Enterprise Base URL",
							Name:        "base_url",
							Type:        "text",
							SubType:     "text",
						},
						{
							DisplayName: "Enterprise Upload URL",
							Name:        "upload_url",
							Type:        "text",
							SubType:     "text",
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
		Build()

	requirementsText := `
**You should have:**
- GitHub account.
[Learn more](https://example.org)`
	requirementsStep := steps.NewCustomStepBuilder("", requirementsText).Build()

	oauthPretext := `
##### :white_check_mark: Step 1: Register an OAuth Application in GitHub
You must first register the Mattermost GitHub Plugin as an authorized OAuth app.`
	oauthMessage := fmt.Sprintf(""+
		"1. Go to %ssettings/applications/new to register an OAuth app.\n"+
		"2. Set the following values:\n"+
		"	- Application name: `Mattermost GitHub Plugin - <your company name>`\n"+
		"	- Homepage URL: `https://github.com/mattermost/mattermost-plugin-github`\n"+
		"	- Authorization callback URL: `%s/oauth/complete`\n"+
		"3. Submit\n"+
		"4. Click **Generate a new client secret** and and provide your GitHub password to continue.\n",
		config.getBaseURL(),
		fm.pluginURL,
	)

	oauthStep := steps.NewCustomStepBuilder("", oauthMessage).
		WithPretext(oauthPretext).
		WithImage("public/new-oauth-application.png").
		WithButton(steps.Button{
			Name:  "Enter OAuth Credentials",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "TODO",
					IntroductionText: "Please enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret**.",
					SubmitLabel:      "Save & Continue",
					Elements: []model.DialogElement{
						{
							DisplayName: "GitHub OAuth Client ID",
							Name:        "client_id",
							Type:        "text",
							SubType:     "text",
						},
						{
							DisplayName: "GitHub OAuth Client Secret",
							Name:        "client_secret",
							Type:        "text",
							SubType:     "text",
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
			Name:    "Cancel",
			Style:   steps.Danger,
			OnClick: fm.cancelFlow,
		}).
		Build()

	connectURL := fmt.Sprintf("%s/oauth/connect", fm.pluginURL)
	connectText := fmt.Sprintf(`
:tada: Awesome! Let's connect your GitHub account!
Click [here](%s) to connect your account.`,
		connectURL,
	)
	conntectStep := steps.NewCustomStepBuilder("", connectText).
		IsNotEmpty(). // The API handler will advance to the next step and complete the flow
		Build()

	steps := []steps.Step{
		welcomeStep,
		handoverQuestionStep,
		handoverSelectionStep,
		enterpriseStep,
		requirementsStep,
		oauthStep,
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

	err = fm.StartConfigurationWizard(aider.Id, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed start configration wizzard").Error(), nil
	}

	attachment := &model.SlackAttachment{
		Text: fmt.Sprintf("Github configuration instructions have been sent to @%s", aider.Username),
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

	fm.trackCompleteConfigurationWizard(userID)

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
		"time": time.Now().UnixMilli(),
	})
}

func (fm *FlowManager) trackCompleteWebhookWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("webhook_wizard_complete", userID, map[string]interface{}{
		"time": time.Now().UnixMilli(),
	})
}

func (fm *FlowManager) getWebhookSteps() []steps.Step {
	questionPretext := `##### :white_check_mark: Step 2: Create a Webhook in GitHub
As a system admin, you must create a webhook for each organization or repository you want to receive notifications for or subscribe to.	`
	questionStep := steps.NewCustomStepBuilder("", "Do you want to create a webhook?").
		WithPretext(questionPretext).
		WithButton(steps.Button{
			Name:  "Yes",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "TODO",
					IntroductionText: "For which repository or organization do you want to create a webhook?",
					SubmitLabel:      "Create",
					Elements: []model.DialogElement{
						{
							DisplayName: "Repository or organization name",
							Name:        "repo_org",
							Type:        "text",
							SubType:     "text",
							Placeholder: "mattermost/mattermost-server",
							// HelpText:    "For which repository or organization do you want to create a webhook?",
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
		Build()

	confirmationStep := steps.NewEmptyStep("", ":tada: You can now use `/github subscriptions add` to subscribe any channel to your repository.")

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

func (fm *FlowManager) getAnnouncemenSteps() []steps.Step {
	defaultMessage := "Hi team,\n" +
		"\n" +
		"We've set up the Mattermost GitHub plugin, so you can get notifications from GitHub in Mattermost. To get started, run the `/github connect` slash command from any channel within Mattermost to connect your Mattermost account with GitHub. Then, take a look at the slash commands section for details about how to use the plugin."
	questionPretext := `##### :white_check_mark: Step 3: Notify your team`
	questionStep := steps.NewCustomStepBuilder("", "Do you want to let your team know, that they can use GitHub integration?").
		WithPretext(questionPretext).
		WithButton(steps.Button{
			Name:  "Yes",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title: "Notify your team",
					//IntroductionText: "Into which channel do you want to send the ",
					SubmitLabel: "Send message",
					Elements: []model.DialogElement{
						{
							DisplayName: "Channel",
							Name:        "channel_id",
							Type:        "select",
							DataSource:  "channels",
						},
						{
							DisplayName: "Message",
							Name:        "message",
							Type:        "textarea",
							Default:     defaultMessage,
							HelpText:    "Edit a message to suit your requirements, and send it out.",
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
			Name:  "No",
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
		"time": time.Now().UnixMilli(),
	})
}

func (fm *FlowManager) trackCompletAnnouncementWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("announcement_wizard_complete", userID, map[string]interface{}{
		"time": time.Now().UnixMilli(),
	})
}
