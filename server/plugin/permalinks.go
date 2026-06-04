// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v54/github"
)

// maxPermalinkReplacements sets the maximum limit to the number of
// permalink replacements that can be performed on a single message.
const maxPermalinkReplacements = 10

const permalinkReqTimeout = 5 * time.Second

// maxPreviewLines sets the maximum number of preview lines that will be shown
// while replacing a permalink.
const maxPreviewLines = 10

// permalinkLineContext shows the number of lines before and after to show
// if the link points to a single line.
const permalinkLineContext = 3

// replacement holds necessary info to replace forgejo permalinks
// in messages with a code preview block.
type replacement struct {
	index         int      // index of the permalink in the string
	word          string   // the permalink
	permalinkInfo struct { // holds the necessary metadata of a permalink
		haswww string
		commit string
		user   string
		repo   string
		path   string
		line   string
	}
}

// getReplacements returns the permalink replacements that needs to be performed
// on a message. The returned slice is sorted by the index in ascending order.
func (p *Plugin) getReplacements(msg string) []replacement {
	// find the permalinks from the msg using a regex
	matches := p.forgejoPermalinkRegex.FindAllStringSubmatch(msg, -1)
	indices := p.forgejoPermalinkRegex.FindAllStringIndex(msg, -1)
	var replacements []replacement
	for i, m := range matches {
		// have a limit on the number of replacements to do
		if i > maxPermalinkReplacements {
			break
		}
		word := m[0]
		index := indices[i][0]
		r := replacement{
			index: index,
			word:  word,
		}
		// ignore if the word is inside a link
		if isInsideLink(msg, index) {
			continue
		}
		// populate the permalinkInfo with the extracted groups of the regex
		for j, name := range p.forgejoPermalinkRegex.SubexpNames() {
			if j == 0 {
				continue
			}
			switch name {
			case "haswww":
				r.permalinkInfo.haswww = m[j]
			case "user":
				r.permalinkInfo.user = m[j]
			case "repo":
				r.permalinkInfo.repo = m[j]
			case "commit":
				r.permalinkInfo.commit = m[j]
			case "path":
				r.permalinkInfo.path = strings.TrimPrefix(path.Join("/", m[j]), "/")
			case "line":
				r.permalinkInfo.line = m[j]
			}
		}
		replacements = append(replacements, r)
	}
	return replacements
}

// makeReplacements perform the given replacements on the msg and returns
// the new msg. The replacements slice needs to be sorted by the index in ascending order.
func (p *Plugin) makeReplacements(msg string, replacements []replacement, ghClient *github.Client) string {
	config := p.getConfiguration()

	// iterating the slice in reverse to preserve the replacement indices.
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]

		ctx, cancel := context.WithTimeout(context.Background(), permalinkReqTimeout)
		defer cancel()

		// Check if repo is public
		if config.EnableCodePreview != "privateAndPublic" {
			repo, _, err := ghClient.Repositories.Get(ctx, r.permalinkInfo.user, r.permalinkInfo.repo)
			if err != nil {
				p.client.Log.Warn("Error while fetching repository information",
					"error", err.Error(),
					"repo", r.permalinkInfo.repo,
					"user", r.permalinkInfo.user)
				continue
			}

			if repo.GetPrivate() {
				continue
			}
		}

		// get the file contents
		opts := github.RepositoryContentGetOptions{
			Ref: r.permalinkInfo.commit,
		}
		// TODO: make all of these requests concurrently.
		fileContent, _, _, err := ghClient.Repositories.GetContents(ctx,
			r.permalinkInfo.user, r.permalinkInfo.repo, r.permalinkInfo.path, &opts)
		if err != nil {
			p.client.Log.Warn("Error while fetching file contents", "error", err.Error(), "path", r.permalinkInfo.path)
			continue
		}
		// this is not a file, ignore.
		if fileContent == nil {
			p.client.Log.Warn("Permalink is not a file", "file", r.permalinkInfo.path)
			continue
		}
		decoded, err := fileContent.GetContent()
		if err != nil {
			p.client.Log.Warn("Error while decoding file contents", "error", err.Error(), "path", r.permalinkInfo.path)
			continue
		}

		// get the required lines.
		start, end := getLineNumbers(r.permalinkInfo.line)
		// bad anchor tag, ignore.
		if start == -1 || end == -1 {
			continue
		}
		isTruncated := false
		if end-start > maxPreviewLines {
			end = start + maxPreviewLines
			isTruncated = true
		}
		lines, err := filterLines(decoded, start, end)
		if err != nil {
			p.client.Log.Warn("Error while filtering lines", "error", err.Error(), "path", r.permalinkInfo.path)
		}
		if lines == "" {
			p.client.Log.Warn("Line numbers out of range. Skipping.", "file", r.permalinkInfo.path, "start", start, "end", end)
			continue
		}
		final := getCodeMarkdown(r.permalinkInfo.user, r.permalinkInfo.repo, r.permalinkInfo.path, r.word, lines, isTruncated)

		// replace word in msg starting from r.index only once.
		msg = msg[:r.index] + strings.Replace(msg[r.index:], r.word, final, 1)
	}
	return msg
}
