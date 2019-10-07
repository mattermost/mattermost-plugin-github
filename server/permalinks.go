package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v25/github"
)

const maxPermalinkReplacements = 10
const permalinkReqTimeout = 2 * time.Second

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
	matches := p.githubRegex.FindAllStringSubmatch(msg, -1)
	indices := p.githubRegex.FindAllStringIndex(msg, -1)
	replacements := make([]replacement, 0, maxPermalinkReplacements)
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
		for j, name := range p.githubRegex.SubexpNames() {
			if j == 0 {
				r.captureMap = make(map[string]string)
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
	for _, r := range replacements {
		// quick bailout if the commit hash is not proper
		_, err := hex.DecodeString(r.captureMap["commit"])
		if err != nil {
			p.API.LogError("bad git commit hash", "error", err.Error(), "hash", r.captureMap["commit"])
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
			continue
		}
		decoded, err := fileContent.GetContent()
		if err != nil {
			p.API.LogError("error while decoding file contents", "error", err.Error(), "path", r.captureMap["path"])
			continue
		}

		// get the required lines
		start, end := getLineNumbers(r.captureMap["line"])
		// bad anchor tag, ignore.
		if start == -1 || end == -1 {
			continue
		}
		lines := filterLines(decoded, start, end)

		// construct the final string
		final := fmt.Sprintf("\n[%s/%s/%s](%s)\n",
			r.captureMap["user"], r.captureMap["repo"], r.captureMap["path"], r.word)
		ext := path.Ext(r.captureMap["path"])
		// remove the preceding dot
		if len(ext) > 1 {
			ext = ext[1:]
		}
		final += "```" + ext + "\n"
		final += lines
		final += "```\n"

		// replace word in msg starting from r.index only once
		msg = msg[:r.index] + strings.Replace(msg[r.index:], r.word, final, 1)
	}
	return msg
}
