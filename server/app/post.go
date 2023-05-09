package app

import "github.com/mattermost/mattermost-server/v6/model"

// CreateBotDMPost posts a direct message using the bot account.
// Any error are not returned and instead logged.
func (a *App) CreateBotDMPost(userID, message, postType string) {
	channel, err := a.client.Channel.GetDirect(userID, a.BotUserID)
	if err != nil {
		a.client.Log.Warn("Couldn't get bot's DM channel", "userID", userID, "error", err.Error())
		return
	}

	post := &model.Post{
		UserId:    a.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if err = a.client.Post.CreatePost(post); err != nil {
		a.client.Log.Warn("Failed to create DM post", "userID", userID, "post", post, "error", err.Error())
		return
	}
}
