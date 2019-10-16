package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReplacements(t *testing.T) {
	p := NewPlugin()

	tcs := []struct {
		name            string
		input           string
		numReplacements int
		replacements    []replacement
	}{
		{
			name:            "basic one link",
			input:           "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		}, {
			name:            "duplicate expansions",
			input:           "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 2,
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				}, {
					index: 142,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		}, {
			name:            "inside link",
			input:           "should not expand [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) here",
			numReplacements: 0,
			replacements:    []replacement{},
		}, {
			name:            "one link, one expansion",
			input:           "first should not expand [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) this should https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 168,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		}, {
			name:            "one expansion, one link",
			input:           "first should expand https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum , this should not [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 20,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		}, {
			name:            "2 links",
			input:           "both should not expand- [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) and [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)",
			numReplacements: 0,
			replacements:    []replacement{},
		}, {
			name:            "multiple expansions",
			input:           "multiple - https://github.com/golang/go/blob/27fc32ff01cc699e160890546816bd99d6c57823/src/debug/macho/macho.go#L13-L16 second https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
			numReplacements: 2,
			replacements: []replacement{
				{
					index: 11,
					word:  "https://github.com/golang/go/blob/27fc32ff01cc699e160890546816bd99d6c57823/src/debug/macho/macho.go#L13-L16",
					captureMap: map[string]string{
						"commit": "27fc32ff01cc699e160890546816bd99d6c57823",
						"haswww": "",
						"line":   "L13-L16",
						"path":   "src/debug/macho/macho.go",
						"user":   "golang",
						"repo":   "go",
					},
				}, {
					index: 126,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					captureMap: map[string]string{
						"commit": "cbb25838a61872b624ac512556d7bc932486a64c",
						"haswww": "",
						"line":   "L15-L22",
						"path":   "app/authentication.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		}, {
			name:            "single line",
			input:           "this is a one line permalink https://github.com/mattermost/mattermost-server/blob/4225977966cf0855c8a5e55f8a0fef702b19dc18/api4/bot.go#L16",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 29,
					word:  "https://github.com/mattermost/mattermost-server/blob/4225977966cf0855c8a5e55f8a0fef702b19dc18/api4/bot.go#L16",
					captureMap: map[string]string{
						"commit": "4225977966cf0855c8a5e55f8a0fef702b19dc18",
						"haswww": "",
						"line":   "L16",
						"path":   "api4/bot.go",
						"user":   "mattermost",
						"repo":   "mattermost-server",
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			replacements := p.getReplacements(tc.input)
			require.Equalf(t, tc.numReplacements, len(replacements), "unexpected no. of replacements for %s", tc.input)
			for i, r := range replacements {
				assert.Equalf(t, tc.replacements[i].index, r.index, "unexpected replacement index")
				assert.Equalf(t, tc.replacements[i].word, r.word, "unexpected replacement word")
				assert.Equalf(t, tc.replacements[i].captureMap["commit"], r.captureMap["commit"], "unexpected github commit")
				assert.Equalf(t, tc.replacements[i].captureMap["haswww"], r.captureMap["haswww"], "unexpected github www domain")
				assert.Equalf(t, tc.replacements[i].captureMap["line"], r.captureMap["line"], "unexpected line number")
				assert.Equalf(t, tc.replacements[i].captureMap["path"], r.captureMap["path"], "unexpected file path")
				assert.Equalf(t, tc.replacements[i].captureMap["user"], r.captureMap["user"], "unexpected github user")
				assert.Equalf(t, tc.replacements[i].captureMap["repo"], r.captureMap["repo"], "unexpected github repo")
			}
		})
	}
}
