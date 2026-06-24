// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package githuberrors

import (
	"net/http"
	"strings"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
)

const samlEnforcementMessage = "organization saml enforcement"

// IsSAMLSSORequired reports whether err indicates GitHub rejected the request
// because the OAuth token lacks SAML SSO authorization for an organization.
func IsSAMLSSORequired(err error) bool {
	if err == nil {
		return false
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		if ssoHeader := ghErr.Response.Header.Get("X-GitHub-SSO"); ssoHeader != "" {
			lower := strings.ToLower(ssoHeader)
			if strings.Contains(lower, "required") || strings.Contains(lower, "partial-results") {
				return true
			}
		}

		if ghErr.Response.StatusCode == http.StatusForbidden && containsSAMLMessage(ghErr.Message) {
			return true
		}
	}

	return containsSAMLMessage(errorMessage(err))
}

func errorMessage(err error) string {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Message != "" {
		return ghErr.Message
	}

	return err.Error()
}

func containsSAMLMessage(message string) bool {
	return strings.Contains(strings.ToLower(message), samlEnforcementMessage)
}
