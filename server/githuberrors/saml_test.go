// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package githuberrors

import (
	"net/http"
	"testing"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsSAMLSSORequired(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.False(t, IsSAMLSSORequired(nil))
	})

	t.Run("unrelated error", func(t *testing.T) {
		assert.False(t, IsSAMLSSORequired(errors.New("something went wrong")))
	})

	t.Run("403 github error with SAML message", func(t *testing.T) {
		err := &github.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusForbidden},
			Message:  "Resource protected by organization SAML enforcement. You must grant your personal token access to this organization.",
		}
		assert.True(t, IsSAMLSSORequired(err))
	})

	t.Run("403 github error without SAML message", func(t *testing.T) {
		err := &github.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusForbidden},
			Message:  "Resource not accessible by integration",
		}
		assert.False(t, IsSAMLSSORequired(err))
	})

	t.Run("X-GitHub-SSO required header", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Github-Sso": {"required; url=https://github.com/orgs/acme/sso"}},
		}
		err := &github.ErrorResponse{Response: resp, Message: "Forbidden"}
		assert.True(t, IsSAMLSSORequired(err))
	})

	t.Run("X-GitHub-SSO partial-results header", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Github-Sso": {"partial-results; organizations=123,456"}},
		}
		err := &github.ErrorResponse{Response: resp}
		assert.True(t, IsSAMLSSORequired(err))
	})

	t.Run("graphql wrapped SAML error", func(t *testing.T) {
		err := errors.Wrap(errors.New("GraphQL: Resource protected by organization SAML enforcement. You must grant your Personal Access token access to this organization."), "error in executing query")
		assert.True(t, IsSAMLSSORequired(err))
	})
}
