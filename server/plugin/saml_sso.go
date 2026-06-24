// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-github/server/githuberrors"
)

const (
	apiErrorIDSAMLSSORequired = "saml_sso_required"
	samlSSONotifiedKey        = "_samlSSONotified"

	samlSSOUserMessage = "GitHub is rejecting API requests because SAML SSO authorization is required for one or more organizations. Run `/github disconnect` and then `/github connect` again. When prompted on GitHub, authorize access for your organizations."
)

func (p *Plugin) notifySAMLSSORequired(userID string) {
	key := userID + samlSSONotifiedKey
	var notified bool
	if err := p.store.Get(key, &notified); err == nil && notified {
		return
	}

	p.CreateBotDMPost(userID, samlSSOUserMessage, "custom_git_saml_sso")
	if _, err := p.store.Set(key, true); err != nil {
		p.client.Log.Warn("Failed to store SAML SSO notification state", "userID", userID, "error", err.Error())
	}
}

func (p *Plugin) clearSAMLSSONotification(userID string) {
	if err := p.store.Delete(userID + samlSSONotifiedKey); err != nil {
		p.client.Log.Debug("Failed to clear SAML SSO notification state", "userID", userID, "error", err.Error())
	}
}

func (p *Plugin) writeSAMLSSOErrorIfNeeded(c *UserContext, w http.ResponseWriter, err error) bool {
	if !githuberrors.IsSAMLSSORequired(err) {
		return false
	}

	p.notifySAMLSSORequired(c.UserID)
	p.writeAPIError(w, &APIErrorResponse{
		ID:         apiErrorIDSAMLSSORequired,
		Message:    samlSSOUserMessage,
		StatusCode: http.StatusForbidden,
	})
	return true
}

func (p *Plugin) handleGitHubAPIError(c *UserContext, err error) {
	if githuberrors.IsSAMLSSORequired(err) {
		p.notifySAMLSSORequired(c.UserID)
	}
}
