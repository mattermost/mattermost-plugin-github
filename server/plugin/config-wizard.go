package plugin

import (
	"context"
	"fmt"

	"github.com/google/go-github/v41/github"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"
)

type propertyStore struct {
}

func (ps *propertyStore) SetProperty(userID, propertyName string, value interface{}) error {
	return nil
}

type FlowManager struct {
	client           *pluginapi.Client
	getConfiguration func() *Configuration
	pluginURL        string

	logger           logger.Logger
	poster           poster.Poster
	store            flow.Store
	wizardController flow.Controller
	webhokController flow.Controller
}

func (p *Plugin) NewFlowManager() *FlowManager {
	fm := &FlowManager{
		client:           p.client,
		logger:           p.log,
		pluginURL:        *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id,
		poster:           poster.NewPoster(&p.client.Post, p.BotUserID),
		store:            flow.NewFlowStore(*p.client, "flow_store"),
		getConfiguration: p.getConfiguration,
	}

	fm.wizardController = flow.NewFlowController(
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

func (fm *FlowManager) getConfigurationFlow() flow.Flow {
	config := fm.getConfiguration()

	step1Text := ":wave: Welcome to GitHub for Mattermost! Finish integrating Mattermost and GitHub by loggin in into your GitHub account."
	step1 := steps.NewCustomStepBuilder("", step1Text).
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

	/*
			step4Text := `
		**You should have:**
		- You have a GitHub account.
		- You're a Mattermost System Admin.`
			step4 := steps.NewCustomStepBuilder("", step4Text).Build()
	*/

	step2Text := "Are you using GitHub Enterprise?"
	step2 := steps.NewCustomStepBuilder("", step2Text).
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

	step5Pretext := `
##### :white_check_mark: Step 1: Register an OAuth Application in GitHub
You must first register the Mattermost GitHub Plugin as an authorized OAuth app.`
	step5Message := fmt.Sprintf(""+
		"1. Go to %s/settings/applications/new to register an OAuth app.\n"+
		"2. Set the following values:\n"+
		"	- Application name: `Mattermost GitHub Plugin - <your company name>`\n"+
		"	- Homepage URL: `https://github.com/mattermost/mattermost-plugin-github`\n"+
		"	- Authorization callback URL: `%s/oauth/complete`\n"+
		"3. Submit\n"+
		"4. Click **Generate a new client secret** and and provide your GitHub password to continue.\n",
		config.getBaseURL(),
		fm.pluginURL,
	)

	step5 := steps.NewCustomStepBuilder("", step5Message).
		WithPretext(step5Pretext).
		WithButton(steps.Button{
			Name:  "I have created the OAuth Application",
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
Let's connect your GitHub account!
Click [here](%s) to connect your account.`,
		connectURL,
	)

	step6 := steps.NewEmptyStep("", connectText)

	steps := []steps.Step{
		/*
			steps.NewEmptyStep("Some Title", "Some message"),
			steps.NewSimpleStep("Simple: Title", "Simple: Message", "property", "true", "false", "selected true", "selected false", 0, 1),
			steps.NewEmptyStep("Some Title1", "Some message1"),
			steps.NewEmptyStep("Some Title2", "Some message2"),
		*/
		step1,
		step2,
		//step3,
		//step4,
		step5,
		step6,

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

func (fm *FlowManager) StartConfigurationWizard(userID string) error {
	err := fm.wizardController.Start(userID)
	if err != nil {
		return err
	}

	return nil
}

func (fm *FlowManager) submitEnterpriseConfig(submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
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

func (fm *FlowManager) submitOAuthConfig(submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
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

	return 0, nil, "", nil
}

func (fm *FlowManager) StartWebhookWizard(userID string) error {
	err := fm.wizardController.Start(userID)
	if err != nil {
		return err
	}

	return nil
}

func (fm *FlowManager) getWebhookFlow() flow.Flow {
	step1 := steps.NewCustomStepBuilder("", "Do you want to create a webhook?").
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
					Title:            "TODO",
					IntroductionText: "For which repository or organization do you want to create a webhook?",
					SubmitLabel:      "Create",
					Elements: []model.DialogElement{
						{
							DisplayName: "Repository or organization",
							Name:        "repo_org",
							Type:        "text",
							SubType:     "text",
							Placeholder: "mattermost/mattermost-server",
							HelpText:    "For which repository or organization do you want to create a webhook?",
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

	steps := []steps.Step{
		step1,
	}
	f := flow.NewFlow(steps, "/webhook", nil)

	return f
}

func (fm *FlowManager) submitWebhook(submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
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
		"content_type": "application/json",
		"insecure_ssl": false,
		"secret":       fm.getConfiguration().WebhookSecret,
		"url":          fmt.Sprintf("%s/webhook", fm.pluginURL),
	}

	hook := &github.Hook{
		Events: webhookEvents,
		Config: config,
	}

	var githubClient *github.Client // := p.githubConnectUser(ctx, userInfo)

	var err error
	if repo == "" {
		_, _, err = githubClient.Organizations.CreateHook(context.TODO(), org, hook)
	} else {
		_, _, err = githubClient.Repositories.CreateHook(context.TODO(), org, repo, hook)
	}

	if err != nil {
		return 0, nil, errors.Wrap(err, "Failed to create hook ").Error(), nil
	}

	return 0, nil, "", nil
}
