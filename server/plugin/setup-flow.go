package plugin

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"
	"github.com/mattermost/mattermost-plugin-api/experimental/freetextfetcher"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

type propertyStore struct {
}

func (ps *propertyStore) SetProperty(userID, propertyName string, value interface{}) error {
	return nil
}

func (p *Plugin) submitEnterpriseConfig(submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
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

	config := p.getConfiguration()
	config.EnterpriseBaseURL = baseURL
	config.EnterpriseUploadURL = uploadURL

	err = p.client.Configuration.SavePluginConfig(config.toMap())
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to save plugin config").Error(), nil
	}

	return 0, nil, "", nil
}

func (p *Plugin) submitOAuthConfig(submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
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

	config := p.getConfiguration()
	config.GitHubOAuthClientID = clientID
	config.GitHubOAuthClientSecret = clientSecret

	err := p.client.Configuration.SavePluginConfig(config.toMap())
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to save plugin config").Error(), nil
	}

	return 0, nil, "", nil
}

func (p *Plugin) handleGetStarted(c *plugin.Context, args *model.CommandArgs, parameters []string) string {
	poster := poster.NewPoster(&p.client.Post, p.BotUserID)

	log := logger.New(p.API)

	flowStore := flow.NewFlowStore(*p.client, "TODO-prefix")

	freeTextStore := freetextfetcher.NewFreetextStore(*p.client, "Freetext-prefix")
	err := freeTextStore.StartFetching(args.UserId, "some fetcher id", "payload")
	if err != nil {
		return err.Error()
	}

	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id

	var controller flow.Controller

	step1Text := ":wave: Welcome to GitHub for Mattermost! Finish integrating Mattermost and GitHub by loggin in into your GitHub account."
	step1 := steps.NewCustomStepBuilder("", step1Text).
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			OnClick: func() int {
				if p.getConfiguration().UsePreregisteredApplication {
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
				OnDialogSubmit: p.submitEnterpriseConfig,
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
		p.getBaseURL(),
		pluginURL,
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
				OnDialogSubmit: p.submitOAuthConfig,
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

	connectURL := fmt.Sprintf("%s/oauth/connect", pluginURL)
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

	f := flow.NewFlow(steps, "/flow", nil)

	controller = flow.NewFlowController(log, p.router, poster, &p.client.Frontend, pluginURL, f, flowStore, &propertyStore{})

	err = controller.Start(args.UserId)
	if err != nil {
		return err.Error()
	}

	return ""
}
