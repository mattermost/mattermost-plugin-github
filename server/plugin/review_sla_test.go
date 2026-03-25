// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewSLAStartKeyStable(t *testing.T) {
	k1 := reviewSLAStartKey("Mattermost", "mattermost", 12345, "octocat")
	k2 := reviewSLAStartKey("mattermost", "mattermost", 12345, "OctoCat")
	assert.Equal(t, k1, k2, "key should be case-insensitive")

	k3 := reviewSLAStartKey("Mattermost", "mattermost", 99999, "octocat")
	assert.NotEqual(t, k1, k3)
}
