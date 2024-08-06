package plugin

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
)

func getMentionSearchQuery(username, org string) string {
	return buildSearchQuery("is:open mentions:%v archived:false %v", username, org)
}

func getReviewSearchQuery(username, org string) string {
	return buildSearchQuery("is:pr is:open review-requested:%v archived:false %v", username, org)
}

func getYourPrsSearchQuery(username, org string) string {
	return buildSearchQuery("is:pr is:open author:%v archived:false %v", username, org)
}

func getYourAssigneeSearchQuery(username, org string) string {
	return buildSearchQuery("is:open assignee:%v archived:false %v", username, org)
}

func getIssuesSearchQuery(org, searchTerm string) string {
	query := "is:open is:issue archived:false %v %v"
	orgField := ""
	if len(org) != 0 {
		orgField = fmt.Sprintf("org:%v", org)
	}

	return fmt.Sprintf(query, orgField, searchTerm)
}

func buildSearchQuery(query, username, org string) string {
	orgField := ""
	if len(org) != 0 {
		orgField = fmt.Sprintf("org:%v", org)
	}

	return fmt.Sprintf(query, username, orgField)
}

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

func encrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "could not create a cipher block, check key")
	}

	msg := pad([]byte(text))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", errors.Wrap(err, "readFull was unsuccessful, check buffer size")
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], msg)
	finalMsg := base64.URLEncoding.EncodeToString(ciphertext)
	return finalMsg, nil
}

func decrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "could not create a cipher block, check key")
	}

	decodedMsg, err := base64.URLEncoding.DecodeString(text)
	if err != nil {
		return "", errors.Wrap(err, "could not decode the message")
	}

	if (len(decodedMsg) % aes.BlockSize) != 0 {
		return "", errors.New("blocksize must be multiple of decoded message length")
	}

	iv := decodedMsg[:aes.BlockSize]
	msg := decodedMsg[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := unpad(msg)
	if err != nil {
		return "", errors.Wrap(err, "unpad error, check key")
	}

	return string(unpadMsg), nil
}

func parseOwnerAndRepo(full, baseURL string) (string, string) {
	full = strings.TrimSuffix(strings.TrimSpace(strings.Replace(full, baseURL, "", 1)), "/")
	splitStr := strings.Split(full, "/")

	if len(splitStr) == 1 {
		owner := splitStr[0]
		return owner, ""
	}

	owner := splitStr[0]
	repo := splitStr[1]
	return owner, repo
}

func parseGitHubUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	for _, word := range strings.FieldsFunc(text, func(c rune) bool {
		return !(c == '-' || c == '@' || unicode.IsLetter(c) || unicode.IsNumber(c))
	}) {
		if len(word) < 2 || word[0] != '@' {
			continue
		}

		if word[1] == '-' || word[len(word)-1] == '-' {
			continue
		}

		if strings.Contains(word, "--") {
			continue
		}

		name := word[1:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func fixGithubNotificationSubjectURL(url, issueNum string) string {
	url = strings.Replace(url, "api.", "", 1)
	url = strings.Replace(url, "repos/", "", 1)
	url = strings.Replace(url, "/pulls/", "/pull/", 1)
	url = strings.Replace(url, "/commits/", "/commit/", 1)
	url = strings.Replace(url, "/api/v3", "", 1)
	url = strings.Replace(url, "comments/", issueNum+"#issuecomment-", 1)
	return url
}

func fullNameFromOwnerAndRepo(owner, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

func isFlag(text string) bool {
	return strings.HasPrefix(text, "--")
}

func parseFlag(flag string) string {
	return strings.TrimPrefix(flag, "--")
}

func containsValue(arr []string, value string) bool {
	for _, element := range arr {
		if element == value {
			return true
		}
	}

	return false
}

// filterLines filters lines in a string from start to end.
func filterLines(s string, start, end int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var buf strings.Builder
	for i := 1; scanner.Scan() && i <= end; i++ {
		if i < start {
			continue
		}
		buf.Write(scanner.Bytes())
		buf.WriteByte(byte('\n'))
	}

	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "scanner error occurred")
	}
	return buf.String(), nil
}

// getLineNumbers return the start and end lines from an anchor tag
// of a github permalink.
func getLineNumbers(s string) (start, end int) {
	// split till -
	parts := strings.Split(s, "-")

	if len(parts) > 2 {
		return -1, -1
	}

	switch len(parts) {
	case 1:
		// just a single line
		l := getLine(parts[0])
		if l == -1 {
			return -1, -1
		}
		if l < permalinkLineContext {
			return 0, l + permalinkLineContext
		}
		return l - permalinkLineContext, l + permalinkLineContext
	case 2:
		// a line range
		start := getLine(parts[0])
		end := getLine(parts[1])
		if start > end && (start != -1 && end != -1) {
			return -1, -1
		}
		return start, end
	}
	return -1, -1
}

// getLine returns the line number in int from a string
// of form L<num>.
func getLine(s string) int {
	// check starting L and minimum length.
	if !strings.HasPrefix(s, "L") || len(s) < 2 {
		return -1
	}

	line, err := strconv.Atoi(s[1:])
	if err != nil {
		return -1
	}
	return line
}

// isInsideLink reports whether the given index in a string is preceded
// by zero or more space, then (, then ].
//
// It is a poor man's version of checking markdown hyperlinks without
// using a full-blown markdown parser. The idea is to quickly confirm
// whether a permalink is inside a markdown link or not. Something like
// "text ]( permalink" is rare enough. Even then, it is okay if
// there are false positives, but there cannot be any false negatives.
//
// Note: it is fine to go one byte at a time instead of one rune because
// we are anyways looking for ASCII chars.
func isInsideLink(msg string, index int) bool {
	stage := 0 // 0 is looking for space or ( and 1 for ]

	for i := index; i > 0; i-- {
		char := msg[i-1]
		switch stage {
		case 0:
			if char == ' ' {
				continue
			}
			if char == '(' {
				stage++
				continue
			}
			return false
		case 1:
			if char == ']' {
				return true
			}
			return false
		}
	}
	return false
}

// getCodeMarkdown returns the constructed markdown for a permalink.
func getCodeMarkdown(user, repo, repoPath, word, lines string, isTruncated bool) string {
	final := fmt.Sprintf("\n[%s/%s/%s](%s)\n", user, repo, repoPath, word)
	ext := path.Ext(repoPath)
	// remove the preceding dot
	if len(ext) > 1 {
		ext = strings.TrimPrefix(ext, ".")
	}
	final += "```" + ext + "\n"
	final += lines
	if isTruncated { // add an ellipsis if lines were cut off
		final += "...\n"
	}
	final += "```\n"
	return final
}

// getToDoDisplayText returns the text to be displayed in todo listings.
func getToDoDisplayText(baseURL, title, url, notifType string, repository *github.Repository) string {
	var owner, repo, repoURL, titlePart string
	if repository == nil {
		owner, repo = parseOwnerAndRepo(url, baseURL)
		repoURL = fmt.Sprintf("%s%s/%s", baseURL, owner, repo)
	} else {
		owner = repository.GetOwner().GetLogin()
		repo = repository.GetName()
		repoURL = repository.GetHTMLURL()
	}

	repoWords := strings.Split(repo, "-")
	if len(repo) > 20 && len(repoWords) > 1 {
		repo = "..." + repoWords[len(repoWords)-1]
	}
	repoPart := fmt.Sprintf("[%s/%s](%s)", owner, repo, repoURL)

	if len(title) > 80 {
		title = strings.TrimSpace(title[:80]) + "..."
	}

	titlePart = fmt.Sprintf(": %s", title)
	if url != "" {
		titlePart = fmt.Sprintf(": [%s](%s)", title, url)
	}

	if notifType == "" {
		return fmt.Sprintf("* %s %s\n", repoPart, titlePart)
	}

	return fmt.Sprintf("* %s %s %s\n", repoPart, notifType, titlePart)
}

// isValidURL checks if a given URL is a valid URL with a host and a http or http scheme.
func isValidURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Errorf("URL schema must either be %q or %q", "http", "https")
	}

	if u.Host == "" {
		return errors.New("URL must contain a host")
	}

	return nil
}

func (p *Plugin) validateIssueRequestForUpdation(issue *UpdateIssueRequest, w http.ResponseWriter) bool {
	if issue.Title == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a valid issue title.", StatusCode: http.StatusBadRequest})
		return false
	}
	if issue.PostID == "" && issue.ChannelID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide either a postID or a channelID", StatusCode: http.StatusBadRequest})
		return false
	}

	return true
}

func (p *Plugin) updatePost(issue *UpdateIssueRequest, w http.ResponseWriter) {
	post, appErr := p.API.GetPost(issue.PostID)
	if appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
		return
	}
	if post == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to load the post %s : not found", issue.PostID), StatusCode: http.StatusNotFound})
		return
	}

	attachments, err := getAttachmentsFromProps(post.Props)
	if err != nil {
		p.client.Log.Warn("Error occurred while getting attachments from props", "PostID", post.Id, "error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("existing attachments format error: %v", err), StatusCode: http.StatusInternalServerError})
		return
	}

	attachments[0].Fields = p.CreateFieldsForIssuePost(issue.Assignees, issue.Labels)
	attachments[0].Title = fmt.Sprintf("%s #%d", issue.Title, issue.IssueNumber)
	attachments[0].Text = issue.Body

	post.Props[attachmentsForProps] = attachments

	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to update the post %s", issue.PostID), StatusCode: http.StatusInternalServerError})
	}
}

func getAttachmentsFromProps(props map[string]interface{}) ([]*model.SlackAttachment, error) {
	attachments, ok := props["attachments"]
	if !ok {
		return nil, fmt.Errorf("no attachments found in props")
	}

	attachmentsData, err := json.Marshal(attachments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attachments: %v", err)
	}

	var slackAttachments []*model.SlackAttachment
	err = json.Unmarshal(attachmentsData, &slackAttachments)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal attachments: %v", err)
	}

	return slackAttachments, nil
}

func (p *Plugin) CreateCommentToIssue(c *UserContext, w http.ResponseWriter, comment, owner, repo string, post *model.Post, issueNumber int) {
	currentUsername := c.GHInfo.GitHubUsername
	issueComment := &github.IssueComment{
		Body: &comment,
	}
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)

	result, rawResponse, err := githubClient.Issues.CreateComment(c.Ctx, owner, repo, issueNumber, issueComment)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if rawResponse != nil {
			statusCode = rawResponse.StatusCode
		}
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create an issue comment: %s", getFailReason(statusCode, repo, currentUsername)), StatusCode: statusCode})
		return
	}

	rootID := post.Id
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	permalinkReplyMessage := fmt.Sprintf("Comment attached to GitHub issue [#%v](%v)", issueNumber, result.GetHTMLURL())
	reply := &model.Post{
		Message:   permalinkReplyMessage,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		UserId:    c.UserID,
	}

	if _, appErr := p.API.CreatePost(reply); appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post %s", post.Id), StatusCode: http.StatusInternalServerError})
		return
	}
}

func (p *Plugin) CloseOrReopenIssue(c *UserContext, w http.ResponseWriter, status, statusReason, owner, repo string, post *model.Post, issueNumber int) {
	currentUsername := c.GHInfo.GitHubUsername
	githubClient := p.githubConnectUser(c.Context.Ctx, c.GHInfo)
	githubIssue := &github.IssueRequest{
		State:       &(status),
		StateReason: &(statusReason),
	}

	issue, resp, err := githubClient.Issues.Edit(c.Ctx, owner, repo, issueNumber, githubIssue)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusGone {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Issues are disabled on this repository.", StatusCode: http.StatusMethodNotAllowed})
			return
		}

		c.Log.WithError(err).Warnf("Failed to update the issue")
		p.writeAPIError(w, &APIErrorResponse{
			ID: "",
			Message: fmt.Sprintf("failed to update the issue: %s", getFailReason(resp.StatusCode,
				repo,
				currentUsername,
			)),
			StatusCode: resp.StatusCode,
		})
		return
	}

	var permalinkReplyMessage string
	switch statusReason {
	case issueCompleted:
		permalinkReplyMessage = fmt.Sprintf("Issue closed as completed [#%v](%v)", issueNumber, issue.GetHTMLURL())
	case issueNotPlanned:
		permalinkReplyMessage = fmt.Sprintf("Issue closed as not planned [#%v](%v)", issueNumber, issue.GetHTMLURL())
	default:
		permalinkReplyMessage = fmt.Sprintf("Issue reopend [#%v](%v)", issueNumber, issue.GetHTMLURL())
	}

	rootID := post.Id
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	reply := &model.Post{
		Message:   permalinkReplyMessage,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		UserId:    c.UserID,
	}

	if _, appErr := p.API.CreatePost(reply); appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to create the notification post %s", post.Id), StatusCode: http.StatusInternalServerError})
		return
	}

	var actionButtonTitle string
	if status == issueClose {
		post.Props[issueStatus] = statusReopen
		actionButtonTitle = statusReopen
	} else {
		post.Props[issueStatus] = statusClose
		actionButtonTitle = statusClose
	}

	attachment, err := getAttachmentsFromProps(post.Props)
	if err != nil {
		p.client.Log.Error("Error occurred while getting attachments from props", "PostID", post.Id, "Error", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("existing attachments format error: %v", err), StatusCode: http.StatusInternalServerError})
		return
	}
	actions := attachment[0].Actions
	for _, action := range actions {
		if action.Name == statusClose || action.Name == statusReopen {
			action.Name = actionButtonTitle
			if status == issueClose {
				action.Integration.Context["status"] = "close"
			} else {
				action.Integration.Context["status"] = "open"
			}
		}
	}
	attachment[0].Actions = actions
	post.Props[attachmentsForProps] = attachment

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("failed to update the post %s", post.Id), StatusCode: http.StatusInternalServerError})
	}
	p.writeJSON(w, issue)
}

// lastN returns the last n characters of a string, with the rest replaced by *.
// At most 3 characters are replaced. The rest is cut off.
func lastN(s string, n int) string {
	if n < 0 {
		return ""
	}

	out := []byte(s)
	if len(out) > n+3 {
		out = out[len(out)-n-3:]
	}
	for i := range out {
		if i < len(out)-n {
			out[i] = '*'
		}
	}

	return string(out)
}

func (p *Plugin) CreateFieldsForIssuePost(assignees []string, labels []string) []*model.SlackAttachmentField {
	fields := []*model.SlackAttachmentField{}
	if len(assignees) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Assignees",
			Value: strings.Join(assignees, ", "),
			Short: true,
		})
	}

	if len(labels) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Labels",
			Value: strings.Join(labels, ", "),
			Short: true,
		})
	}

	return fields
}
