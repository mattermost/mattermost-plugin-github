package plugin

import (
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"
	"github.com/mattermost/mattermost-plugin-api/experimental/freetextfetcher"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

type propertyStore struct {
}

func (ps *propertyStore) SetProperty(userID, propertyName string, value interface{}) error {
	return nil
}

func (p *Plugin) handleGetStarted(c *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *GitHubUserInfo) string {
	client := pluginapi.NewClient(p.API, p.Driver)
	poster := poster.NewPoster(&client.Post, p.BotUserID)

	log := logger.New(p.API)

	flowStore := flow.NewFlowStore(*client, "TODO-prefix")

	freeTextStore := freetextfetcher.NewFreetextStore(*client, "Freetext-prefix")
	err := freeTextStore.StartFetching(args.UserId, "some fetcher id", "payload")
	if err != nil {
		return err.Error()
	}

	pluginURL := *client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id

	/*
		validate := func(message string) string {
			if message == "foo" {
				return ""
			}

			return "you must enter `foo`"
		}
	*/
	var controller flow.Controller
	b1 := steps.Button{
		Name:  "Continue",
		Style: steps.Primary,
	}
	b2 := steps.Button{
		Name:      "Not now",
		Style:     steps.Default,
		SkipSteps: 999,
	}

	step2Pretext := `**You should have:**
- You have a GitHub account.
- You're a Mattermost System Admin.
- You're running Mattermost v5.12 or higher.
	`

	step3Pretext := `##### :white_check_mark: Step 1: Register an OAuth Application in GitHub
You must first register the Mattermost GitHub Plugin as an authorized OAuth app regardless of whether you're setting up the GitHub plugin as a system admin or a Mattermost user.`

	step3Message := `
1. Set the following values:
	- Foo
	- bar
2. Submit
3. Click **Generate a new client secret** and
`

	steps := []steps.Step{
		steps.NewCustomStepBuilder("", "").WithButton(b1).WithButton(b2).WithPretext(":wave: Welcome to GitHub for Mattermost! Finish integrating Mattermost and GitHub by loggin in into your GitHub account.").Build(),
		steps.NewCustomStepBuilder("", "").WithPretext(step2Pretext).Build(),
		steps.NewCustomStepBuilder("", step3Message).
			WithPretext(step3Pretext).
			WithButton(steps.Button{
				Name:      "Register",
				Style:     steps.Primary,
				SkipSteps: 0,
			}).
			WithButton(steps.Button{
				Name:      "Register",
				Style:     steps.Default,
				SkipSteps: 0,
			}).
			WithButton(steps.Button{
				Name:      "Cancel",
				Style:     steps.Danger,
				SkipSteps: 999,
			}).
			Build(),

		//steps.NewEmptyStep("Some Title", "Some message"),
		/*

			steps.NewSimpleStep("Simple: Title", "Simple: Message", "property", "true", "false", "selected true", "selected false", 0, 1),
			steps.NewFreetextStep("Freetext: Title", "Freetext: Message", "property", "/freetext", freeTextStore, validate, p.router, poster),
		*/

		//steps.NewEmptyStep("Some Title", "Some message"),
		//steps.NewEmptyStep("Some Title2", "Some message2"),
		//steps.NewEmptyStep("Some Title3", "Some message3"),
	}

	f := flow.NewFlow(steps, "/flow", nil)

	controller = flow.NewFlowController(poster, log, p.router, pluginURL, f, flowStore, &propertyStore{})

	err = controller.Start(args.UserId)
	if err != nil {
		return err.Error()
	}

	return ""
}
