package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionFlagsString(t *testing.T) {
	tcs := []struct {
		name  string
		input SubscriptionFlags
		want  string
	}{
		{
			name: "should return correct string when one flag set",
			input: SubscriptionFlags{
				ExcludeOrgMembers: true,
			},
			want: "--exclude-org-member",
		},
		{
			name: "should return correct string when one flag unset",
			input: SubscriptionFlags{
				ExcludeOrgMembers: false,
			},
			want: "",
		},
		{
			name:  "should return correct string when no flag set",
			input: SubscriptionFlags{},
			want:  "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.input.String())
		})
	}
}
