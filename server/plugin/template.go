package plugin

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
)

const mdCommentRegexPattern string = `(<!--[\S\s]+?-->)`

// There is no public documentation of what constitutes a GitHub username, but
// according to the error messages returned in https://github.com/join, it must:
//  1. be between 1 and 39 characters long.
//  2. contain only alphanumeric characters or non-adjacent hyphens.
//  3. not begin or end with a hyphen.
//
// When matching a valid GitHub username in the body of messages, it must:
//  4. not be preceded by an underscore, a backtick (that cryptic \x60) or an
//     alphanumeric character.
//
// Ensuring the maximum length is not trivial without lookaheads, so this
// regexp ensures only the minimum length, besides points 2, 3 and 4.
// Note that the username, with the @ sign, is in the second capturing group.
const gitHubUsernameRegexPattern string = `(^|[^_\x60[:alnum:]])(@[[:alnum:]](-?[[:alnum:]]+)*)`

var mdCommentRegex = regexp.MustCompile(mdCommentRegexPattern)
var gitHubUsernameRegex = regexp.MustCompile(gitHubUsernameRegexPattern)
var masterTemplate *template.Template
var gitHubToUsernameMappingCallback func(string) string

func init() {
	var funcMap = sprig.TxtFuncMap()

	// Try to parse out email footer junk
	funcMap["trimBody"] = func(body string) string {
		if strings.Contains(body, "notifications@github.com") {
			return strings.Split(body, "\n\nOn")[0]
		}

		return body
	}

	// Trim a ref to use in constructing a link.
	funcMap["trimRef"] = func(ref string) string {
		return strings.Replace(ref, "refs/heads/", "", 1)
	}

	// Resolve a GitHub username to the corresponding Mattermost username, if linked.
	funcMap["lookupMattermostUsername"] = lookupMattermostUsername

	// Trim away markdown comments in the text
	funcMap["removeComments"] = func(body string) string {
		if len(strings.TrimSpace(body)) == 0 {
			return ""
		}
		return mdCommentRegex.ReplaceAllString(body, "")
	}

	// Replace any GitHub username with its corresponding Mattermost username, if any
	funcMap["replaceAllGitHubUsernames"] = func(body string) string {
		return gitHubUsernameRegex.ReplaceAllStringFunc(body, func(matched string) string {
			// The matched string contains the @ sign, and may contain a single
			// character prepending the whole thing.
			gitHubUsernameFirstCharIndex := strings.LastIndex(matched, "@") + 1
			prefix := matched[:gitHubUsernameFirstCharIndex]
			gitHubUsername := matched[gitHubUsernameFirstCharIndex:]

			username := lookupMattermostUsername(gitHubUsername)
			if username == "" {
				return matched
			}

			return prefix + username
		})
	}

	// Quote the body
	funcMap["quote"] = func(body string) string {
		return ">" + strings.ReplaceAll(body, "\n", "\n>")
	}

	// Escape characters not allowed in URL path
	funcMap["pathEscape"] = url.PathEscape

	// Transform multiple variables to dictionary
	funcMap["dict"] = func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, errors.New("invalid dict call, exactly one value is required for every key")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, errors.New("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	}

	masterTemplate = template.Must(template.New("master").Funcs(funcMap).Parse(""))

	// The user template links to the corresponding GitHub user. If the GitHub user is a known
	// Mattermost user, their Mattermost handle is referenced as an at-mention instead.
	template.Must(masterTemplate.New("user").Parse(`
{{- $mattermostUsername := .GetLogin | lookupMattermostUsername}}
{{- if $mattermostUsername }}@{{$mattermostUsername}}
{{- else}}[{{.GetLogin}}]({{.GetHTMLURL}})
{{- end -}}
	`))

	// The repo template links to the corresponding repository.
	template.Must(masterTemplate.New("repo").Parse(
		`[\[{{.GetFullName}}\]]({{.GetHTMLURL}})`,
	))

	// The eventRepoPullRequest links to the corresponding pull request, anchored at the repo.
	template.Must(masterTemplate.New("eventRepoPullRequest").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetPullRequest.GetNumber}}]({{.GetPullRequest.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("eventRepoPullRequestWithTitle").Parse(
		`{{template "eventRepoPullRequest" .}} - {{.GetPullRequest.GetTitle}}`,
	))

	// The reviewRepoPullRequest links to the corresponding pull request, anchored at the repo.
	template.Must(masterTemplate.New("reviewRepoPullRequest").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetPullRequest.GetNumber}}]({{.GetReview.GetHTMLURL}})`,
	))

	// this reviewRepoPullRequestWithTitle just adds title
	template.Must(masterTemplate.New("reviewRepoPullRequestWithTitle").Parse(
		`{{template "reviewRepoPullRequest" .}} - {{.GetPullRequest.GetTitle}}`,
	))

	// The pullRequest links to the corresponding pull request, skipping the repo title.
	template.Must(masterTemplate.New("pullRequest").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	// The issue links to the corresponding issue.
	template.Must(masterTemplate.New("issue").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	// The eventRepoIssue links to the corresponding issue. Note that, for some events, the
	// issue *is* a pull request, and so we still use .GetIssue and this template accordingly.
	template.Must(masterTemplate.New("eventRepoIssue").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetIssue.GetNumber}}]({{.GetIssue.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("eventRepoIssueWithTitle").Parse(
		`{{template "eventRepoIssue" .}} - {{.GetIssue.GetTitle}}`,
	))

	// The eventRepoIssueFullLink links to the corresponding comment in the issue. Note that, for some events, the
	// issue *is* a pull request, and so we still use .GetIssue and this template accordingly.
	// and .GetComment return full link to the comment as long as comment object is present in the payload
	template.Must(masterTemplate.New("eventRepoIssueFullLink").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetIssue.GetNumber}}]({{.GetComment.GetHTMLURL}})`,
	))

	// eventRepoIssueFullLinkWithTitle template is sibling of eventRepoIssueWithTitle
	// this one refers to the comment instead of the issue itself
	template.Must(masterTemplate.New("eventRepoIssueFullLinkWithTitle").Parse(
		`{{template "eventRepoIssueFullLink" .}} - {{.GetIssue.GetTitle}}`,
	))

	template.Must(masterTemplate.New("labels").Funcs(funcMap).Parse(`
{{- if .Labels }}
Labels: {{range $i, $el := .Labels -}}` + "{{- if $i}}, {{end}}[`{{ $el.Name }}`]({{ $.RepositoryURL }}/labels/{{ $el.Name | pathEscape }})" + `{{end -}}
{{ end -}}
`))

	template.Must(masterTemplate.New("assignee").Funcs(funcMap).Parse(`
{{- if .Assignees }}
Assignees: {{range $i, $el := .Assignees -}} {{- if $i}}, {{end}}{{template "user" $el}}{{end -}}
{{- end -}}
`))

	template.Must(masterTemplate.New("newPR").Funcs(funcMap).Parse(`
{{ if eq .Config.Style "collapsed" -}}
{{template "repo" .Event.GetRepo}} New pull request {{template "pullRequest" .Event.GetPullRequest}} was opened by {{template "user" .Event.GetSender}}.
{{- else -}}
#### {{.Event.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .Event}}
#new-pull-request by {{template "user" .Event.GetSender}}
{{- if ne .Config.Style "skip-body" -}}
{{- template "labels" dict "Labels" .Event.GetPullRequest.Labels "RepositoryURL" .Event.GetRepo.GetHTMLURL  }}
{{- template "assignee" .Event.GetPullRequest }}

{{.Event.GetPullRequest.GetBody | removeComments | replaceAllGitHubUsernames}}
{{- end -}}
{{- end }}
`))

	template.Must(masterTemplate.New("closedPR").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} Pull request {{template "pullRequest" .GetPullRequest}} was
{{- if .GetPullRequest.GetMerged }} merged
{{- else }} closed
{{- end }} by {{template "user" .GetSender}}.
`))

	template.Must(masterTemplate.New("pullRequestLabelled").Funcs(funcMap).Parse(`
#### {{.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .}}
#pull-request-labeled ` + "`{{.GetLabel.GetName}}`" + ` by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("pullRequestMentionNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} mentioned you on [{{.GetRepo.GetFullName}}#{{.GetPullRequest.GetNumber}}]({{.GetPullRequest.GetHTMLURL}}) - {{.GetPullRequest.GetTitle}}:
{{.GetPullRequest.GetBody | trimBody | quote | replaceAllGitHubUsernames}}`))

	template.Must(masterTemplate.New("newIssue").Funcs(funcMap).Parse(`
{{ if eq .Config.Style "collapsed" -}}
{{template "repo" .Event.GetRepo}} New issue {{template "issue" .Event.GetIssue}} opened by {{template "user" .Event.GetSender}}.
{{- else -}}
#### {{.Event.GetIssue.GetTitle}}
##### {{template "eventRepoIssue" .Event}}
#new-issue by {{template "user" .Event.GetSender}}
{{- if ne .Config.Style "skip-body" -}}
{{- template "labels" dict "Labels" .Event.GetIssue.Labels "RepositoryURL" .Event.GetRepo.GetHTMLURL  }}
{{- template "assignee" .Event.GetIssue }}

{{.Event.GetIssue.GetBody | removeComments | replaceAllGitHubUsernames}}
{{- end -}}
{{- end }}
`))

	template.Must(masterTemplate.New("closedIssue").Funcs(funcMap).Parse(`
{{template "repo" .Event.GetRepo}} Issue {{template "issue" .Event.GetIssue}} closed by {{template "user" .Event.GetSender}}.
`))

	template.Must(masterTemplate.New("issueLabelled").Funcs(funcMap).Parse(`
#### {{.Event.GetIssue.GetTitle}}
##### {{template "eventRepoIssue" .Event}}
#issue-labeled ` + "`{{.Event.GetLabel.GetName}}`" + ` by {{template "user" .Event.GetSender}}.
`))

	template.Must(masterTemplate.New("reopenedIssue").Funcs(funcMap).Parse(`
{{template "repo" .Event.GetRepo}} Issue {{template "issue" .Event.GetIssue}} reopened by {{template "user" .Event.GetSender}}.
`))

	template.Must(masterTemplate.New("pushedCommits").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} {{if .GetForced}}force-{{end}}pushed [{{len .Commits}} new commit{{if ne (len .Commits) 1}}s{{end}}]({{.GetCompare}}) to [\[{{.GetRepo.GetFullName}}:{{.GetRef | trimRef}}\]]({{.GetRepo.GetHTMLURL}}/tree/{{.GetRef | trimRef}}):
{{range .Commits -}}
[` + "`{{.GetID | substr 0 6}}`" + `]({{.GetURL}}) {{.GetMessage}} - {{.GetCommitter.GetName}}
{{end -}}
`))

	template.Must(masterTemplate.New("newCreateMessage").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{.GetRefType}} [{{.GetRef}}]({{.GetRepo.GetHTMLURL}}/tree/{{.GetRef}}) created by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("newDeleteMessage").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{.GetRefType}} {{.GetRef}} deleted by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("issueComment").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} New comment by {{template "user" .GetSender}} on {{template "issue" .Issue}}:

{{.GetComment.GetBody | trimBody | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("pullRequestReviewEvent").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{template "user" .GetSender}}
{{- if eq .GetReview.GetState "APPROVED"}} approved
{{- else if eq .GetReview.GetState "COMMENTED"}} commented on
{{- else if eq .GetReview.GetState "CHANGES_REQUESTED"}} requested changes on
{{- end }} {{template "pullRequest" .GetPullRequest}}:

{{.Review.GetBody | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("newReviewComment").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} New review comment by {{template "user" .GetSender}} on {{template "pullRequest" .GetPullRequest}}:

{{.GetComment.GetDiffHunk}}
{{.GetComment.GetBody | trimBody | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentMentionNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} mentioned you on [{{.GetRepo.GetFullName}}#{{.Issue.GetNumber}}]({{.GetComment.GetHTMLURL}}) - {{.Issue.GetTitle}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAuthorPullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on your pull request {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneePullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on pull request you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeIssueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on issue you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeSelfMentionPullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} mentioned you on a pull request that you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeSelfMentionIssueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} mentioned you on a issue that you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("commentAuthorIssueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on your issue {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.GetComment.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
`))

	template.Must(masterTemplate.New("pullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetAction "review_requested" }} requested your review on
{{- else if eq .GetAction "closed" }}
    {{- if .GetPullRequest.GetMerged }} merged your pull request
    {{- else }} closed your pull request
    {{- end }}
{{- else if eq .GetAction "reopened" }} reopened your pull request
{{- else if eq .GetAction "assigned" }} assigned you to pull request
{{- end }} {{template "eventRepoPullRequestWithTitle" .}}
`))

	template.Must(masterTemplate.New("issueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetAction "closed" }} closed your issue
{{- else if eq .GetAction "reopened" }} reopened your issue
{{- else if eq .GetAction "assigned" }} assigned you to issue
{{- end }} {{template "eventRepoIssueWithTitle" .}}
`))

	template.Must(masterTemplate.New("pullRequestReviewNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetReview.GetState "approved" }} approved your pull request
{{- else if eq .GetReview.GetState "changes_requested" }} requested changes on your pull request
{{- else if eq .GetReview.GetState "commented" }} commented on your pull request
{{- end }} {{template "reviewRepoPullRequestWithTitle" .}}
{{if .GetReview.GetBody}}{{.Review.GetBody | trimBody | quote | replaceAllGitHubUsernames}}
{{else}}{{end}}`))

	template.Must(masterTemplate.New("helpText").Parse("" +
		"* `/github connect{{if .EnablePrivateRepo}}{{if not .ConnectToPrivateByDefault}} [private]{{end}}{{end}}` - Connect your Mattermost account to your GitHub account.\n" +
		"{{if .EnablePrivateRepo}}{{if not .ConnectToPrivateByDefault}}" +
		"  * `private` is optional. If used, read access to your private repositories will be requested." +
		"If these repositories send webhook events to this Mattermost server, you'll be notified of changes to those repositories.\n" +
		"{{else}}" +
		"  * Read access to your private repositories will be requested." +
		"If these repositories send webhook events to this Mattermost server, you'll be notified of changes to those repositories.\n" +
		"{{end}}{{end}}" +
		"* `/github disconnect` - Disconnect your Mattermost account from your GitHub account\n" +
		"* `/github help` - Display Slash Command help text\n" +
		"* `/github todo` - Get a list of unread messages and pull requests awaiting your review\n" +
		"* `/github subscriptions list` - Will list the current channel subscriptions\n" +
		"* `/github subscriptions add owner[/repo] [flags]` - Subscribe the current channel to receive notifications about opened pull requests and issues for an organization or repository\n" +
		"  * `flags` currently supported:\n" +
		"	 * `--features` - a comma-delimited list of one or more of the following:\n" +
		"    	* `issues` - includes new and closed issues\n" +
		"    	* `pulls` - includes new and closed pull requests\n" +
		"    	* `pulls_merged` - includes merged pull requests only\n" +
		"    	* `pushes` - includes pushes\n" +
		"    	* `creates` - includes branch and tag creations\n" +
		"    	* `deletes` - includes branch and tag deletions\n" +
		"    	* `issue_comments` - includes new issue comments\n" +
		"    	* `issue_creations` - includes new issues only \n" +
		"    	* `pull_reviews` - includes pull request reviews\n" +
		"    	* `label:<labelname>` - limit pull request and issue events to only this label. Must include `pulls` or `issues` in feature list when using a label.\n" +
		"    	* Defaults to `pulls,issues,creates,deletes`\n\n" +
		"    * `--exclude-org-member` - events triggered by organization members will not be delivered (the GitHub organization config should be set, otherwise this flag has not effect)\n" +
		"    * `--render-style` - notifications will be delivered in the specified style (for example, the body of a pull request will not be displayed). Supported values are `collapsed`, `skip-body` or `default` (same as omitting the flag).\n" +
		"* `/github subscriptions delete owner[/repo]` - Unsubscribe the current channel from a repository\n" +
		"* `/github me` - Display the connected GitHub account\n" +
		"* `/github settings [setting] [value]` - Update your user settings\n" +
		"  * `setting` can be `notifications` or `reminders`\n" +
		"  * `value` can be `on` or `off`\n" +
		"* `/github mute` - Managed muted GitHub users. You'll not receive notifications for comments in your PRs and issues from those users.\n" +
		"  * `/github mute list` - list your muted GitHub users\n" +
		"  * `/github mute add [username]` - add a GitHub user to your muted list\n" +
		"  * `/github mute delete [username]` - remove a GitHub user from your muted list\n" +
		"  * `/github mute delete-all` - unmute all GitHub users\n"))

	template.Must(masterTemplate.New("newRepoStar").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}}
{{- if eq .GetAction "created" }} starred
{{- else }} unstarred
{{- end }} by {{template "user" .GetSender}}
It now has **{{.GetRepo.GetStargazersCount}}** stars.`))
}

func registerGitHubToUsernameMappingCallback(callback func(string) string) {
	gitHubToUsernameMappingCallback = callback
}

func lookupMattermostUsername(githubUsername string) string {
	if gitHubToUsernameMappingCallback == nil {
		return ""
	}

	return gitHubToUsernameMappingCallback(githubUsername)
}

func renderTemplate(name string, data interface{}) (string, error) {
	var output bytes.Buffer
	t := masterTemplate.Lookup(name)
	if t == nil {
		return "", errors.Errorf("no template named %s", name)
	}

	err := t.Execute(&output, data)
	if err != nil {
		return "", errors.Wrapf(err, "Could not execute template named %s", name)
	}

	return output.String(), nil
}
