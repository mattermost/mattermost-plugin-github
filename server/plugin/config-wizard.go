package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v41/github"
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

	logger                  logger.Logger
	poster                  poster.Poster
	store                   flow.Store
	configurationController flow.Controller
	webhokController        flow.Controller
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

	fm.configurationController = flow.NewFlowController(
		fm.logger,
		p.router,
		fm.poster,
		&p.client.Frontend,
		fm.pluginURL,
		fm.getConfigurationFlow(),
		fm.store,
		&propertyStore{},
	)

	fm.webhokController = flow.NewFlowController(
		fm.logger,
		p.router,
		fm.poster,
		&p.client.Frontend,
		fm.pluginURL,
		fm.getWebhookFlow(),
		fm.store,
		&propertyStore{},
	)

	return fm
}

func (fm *FlowManager) StartConfigurationWizard(userID string, fromInvite bool) error {
	err := fm.configurationController.Start(userID)
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

func (fm *FlowManager) getConfigurationFlow() flow.Flow {
	config := fm.getConfiguration()

	welcomeText := ":wave: Welcome to GitHub for Mattermost! Finish integrating Mattermost and GitHub by loggin in into your GitHub account."
	welcomeStep := steps.NewCustomStepBuilder("", welcomeText).
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			OnClick: func() int {
				if fm.getConfiguration().UsePreregisteredApplication {
					return 2
				}

				return 0
			},
		}).
		WithButton(steps.Button{
			Name:  "Not now",
			Style: steps.Default,
			OnClick: func() int {
				return 999
			},
		}).
		Build()

	handoverQuestionText := "Do you need another person to finish setting GitHub up?"
	handoverQuestionStep := steps.NewCustomStepBuilder("", handoverQuestionText).
		WithButton(steps.Button{
			Name:  "I'll do it myself",
			Style: steps.Primary,
			OnClick: func() int {
				return 2
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
			OnClick: func() int {
				return 999
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
			OnClick: func() int {
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
	- You have a GitHub account.`
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
					SubmitLabel:      "Continue",
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
			OnClick: func() int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func() int {
				return 999
			},
		}).
		Build()

	connectURL := fmt.Sprintf("%s/oauth/connect", fm.pluginURL)
	connectText := fmt.Sprintf(`
:tada: Awesome! Let's connect your GitHub account!
Click [here](%s) to connect your account.`,
		connectURL,
	)
	conntectStep := steps.NewEmptyStep("", connectText)

	steps := []steps.Step{
		welcomeStep,
		handoverQuestionStep,
		handoverSelectionStep,
		enterpriseStep,
		requirementsStep,
		oauthStep,
		conntectStep,

		/*

			steps.NewFreetextStep("Freetext: Title", "Freetext: Message", "property", "/freetext", freeTextStore, validate, p.router, poster),

			steps.NewEmptyStep("Some Title", "Some message"),
			steps.NewEmptyStep("Some Title", "Some message"),
			steps.NewEmptyStep("Some Title2", "Some message2"),
			steps.NewEmptyStep("Some Title3", "Some message3"),
		*/
	}

	f := flow.NewFlow(steps, "/wizard", nil)

	return f
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

	return -1, nil, "", nil
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

func (fm *FlowManager) getWebhookFlow() flow.Flow {
	step1 := steps.NewCustomStepBuilder("", "Do you want to create a webhook?").
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
			OnClick: func() int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "No",
			Style: steps.Default,
			OnClick: func() int {
				return 999
			},
		}).
		Build()

	step2 := steps.NewEmptyStep("", ":tada: You can now use `/github subscriptions add` to subscribe any channel to your repository.")

	steps := []steps.Step{
		step1,
		step2,
	}
	f := flow.NewFlow(steps, "/webhook", nil)

	return f
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
	if repo == "" {
		hook, _, err = client.Organizations.CreateHook(ctx, org, hook)
	} else {
		hook, _, err = client.Repositories.CreateHook(ctx, org, repo, hook)
	}

	if err != nil {
		return 0, nil, errors.Wrap(err, "Failed to create hook").Error(), nil
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
