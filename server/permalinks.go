package main

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/go-github/v25/github"
)

// maxPermalinkReplacements sets the maximum limit to the no. of
// permalink replacements that can be performed on a single message.
const maxPermalinkReplacements = 10

const permalinkReqTimeout = 5 * time.Second

// maxPreviewLines sets the maximum no. of preview lines that will be shown
// while replacing a permalink.
const maxPreviewLines = 10

// permalinkLineContext shows the number of lines before and after to show
// if the link points to a single line.
const permalinkLineContext = 3

// replacement holds necessary info to replace github permalinks
// in messages with a code preview block
type replacement struct {
	index      int               // index of the permalink in the string
	word       string            // the permalink
	captureMap map[string]string // named regex capture group of the link
}

// getReplacements returns the permalink replacements that needs to be performed
// on a message.
func (p *Plugin) getReplacements(msg string) []replacement {
	// find the permalinks from the msg using a regex
	matches := p.githubPermalinkRegex.FindAllStringSubmatch(msg, -1)
	indices := p.githubPermalinkRegex.FindAllStringIndex(msg, -1)
	var replacements []replacement
	for i, m := range matches {
		// have a limit on the no. of replacements to do
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
		// populate the captureMap with the extracted groups of the regex
		r.captureMap = make(map[string]string)
		for j, name := range p.githubPermalinkRegex.SubexpNames() {
			if j == 0 {
				continue
			}
			r.captureMap[name] = m[j]
		}
		replacements = append(replacements, r)
	}
	return replacements
}

// makeReplacements perform the given replacements on the msg and returns
// the new msg.
func (p *Plugin) makeReplacements(msg string, replacements []replacement) string {
	// iterating the slice in reverse to preserve the replacement indices.
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		// quick bailout if the commit hash is not proper.
		if _, err := hex.DecodeString(r.captureMap["commit"]); err != nil {
			p.API.LogError("bad git commit hash in permalink", "error", err.Error(), "hash", r.captureMap["commit"])
			continue
		}

		// get the file contents
		opts := github.RepositoryContentGetOptions{
			Ref: r.captureMap["commit"],
		}
		// TODO: make all of these requests concurrently.
		reqctx, cancel := context.WithTimeout(context.Background(), permalinkReqTimeout)
		fileContent, _, _, err := p.githubClient.Repositories.GetContents(reqctx,
			r.captureMap["user"], r.captureMap["repo"], r.captureMap["path"], &opts)
		if err != nil {
			p.API.LogError("error while fetching file contents", "error", err.Error(), "path", r.captureMap["path"])
			cancel()
			continue
		}
		cancel()
		// this is not a file, ignore.
		if fileContent == nil {
			p.API.LogWarn("permalink is not a file", "file", r.captureMap["path"])
			continue
		}
		decoded, err := fileContent.GetContent()
		if err != nil {
			p.API.LogError("error while decoding file contents", "error", err.Error(), "path", r.captureMap["path"])
			continue
		}

		// get the required lines.
		start, end := getLineNumbers(r.captureMap["line"])
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
			p.API.LogError("error while filtering lines", "error", err.Error(), "path", r.captureMap["path"])
		}
		if lines == "" {
			p.API.LogError("line numbers out of range. Skipping.", "file", r.captureMap["path"], "start", start, "end", end)
			continue
		}
		final := getCodeMarkdown(r.captureMap, r.word, lines, isTruncated)

		// replace word in msg starting from r.index only once.
		msg = msg[:r.index] + strings.Replace(msg[r.index:], r.word, final, 1)
	}
	return msg
}
