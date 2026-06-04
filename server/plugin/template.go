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

// There is no public documentation of what constitutes a Forgejo username, but
// according to the error messages returned in https://github.com/join, it must:
//  1. be between 1 and 39 characters long.
//  2. contain only alphanumeric characters or non-adjacent hyphens.
//  3. not begin or end with a hyphen.
//
// When matching a valid Forgejo username in the body of messages, it must:
//  4. not be preceded by an underscore, a backtick (that cryptic \x60) or an
//     alphanumeric character.
//
// Ensuring the maximum length is not trivial without lookaheads, so this
// regexp ensures only the minimum length, besides points 2, 3 and 4.
// Note that the username, with the @ sign, is in the second capturing group.
const forgejoUsernameRegexPattern string = `(^|[^_\x60[:alnum:]])(@[[:alnum:]](-?[[:alnum:]]+)*)`

var mdCommentRegex = regexp.MustCompile(mdCommentRegexPattern)
var forgejoUsernameRegex = regexp.MustCompile(forgejoUsernameRegexPattern)
var masterTemplate *template.Template
var forgejoToUsernameMappingCallback func(string) string
var showAuthorInCommitNotification bool

func init() {
	var funcMap = sprig.TxtFuncMap()

	// Try to parse out email footer junk
	funcMap["trimBody"] = func(body string) string {
		if strings.Contains(body, "notifications@forgejo.pyn.ru") {
			return strings.Split(body, "\n\nOn")[0]
		}

		return body
	}

	// Trim space
	funcMap["trimSpace"] = strings.TrimSpace

	// Trim a ref to use in constructing a link.
	funcMap["trimRef"] = func(ref string) string {
		return strings.Replace(ref, "refs/heads/", "", 1)
	}

	// Resolve a Forgejo username to the corresponding Mattermost username, if linked.
	funcMap["lookupMattermostUsername"] = lookupMattermostUsername

	// Trim away markdown comments in the text
	funcMap["removeComments"] = func(body string) string {
		if len(strings.TrimSpace(body)) == 0 {
			return ""
		}
		return mdCommentRegex.ReplaceAllString(body, "")
	}

	// Replace any Forgejo username with its corresponding Mattermost username, if any
	funcMap["replaceAllForgejoUsernames"] = func(body string) string {
		return forgejoUsernameRegex.ReplaceAllStringFunc(body, func(matched string) string {
			// The matched string contains the @ sign, and may contain a single
			// character prepending the whole thing.
			forgejoUsernameFirstCharIndex := strings.LastIndex(matched, "@") + 1
			prefix := matched[:forgejoUsernameFirstCharIndex]
			forgejoUsername := matched[forgejoUsernameFirstCharIndex:]

			username := lookupMattermostUsername(forgejoUsername)
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

	funcMap["commitAuthor"] = func(commit *FHeadCommit) *FCommitAuthor {
		if showAuthorInCommitNotification {
			return commit.Author
		}

		return commit.Committer
	}

	funcMap["workflowJobFailedStep"] = func(steps []*FTaskStep) string {
		for _, step := range steps {
			if step.GetConclusion() == workflowJobFail {
				return step.GetName()
			}
		}

		return ""
	}

	masterTemplate = template.Must(template.New("master").Funcs(funcMap).Parse(""))

	// The user template links to the corresponding Forgejo user. If the Forgejo user is a known
	// Mattermost user, their Mattermost handle is referenced as an at-mention instead.
	template.Must(masterTemplate.New("user").Parse(`
{{- $mattermostUsername := .GetLogin | lookupMattermostUsername}}
{{- if $mattermostUsername }}@{{$mattermostUsername}}
{{- else}}[{{.GetLogin}}]({{.GetHTMLURL}})
{{- end -}}
	`))

	template.Must(masterTemplate.New("FUser").Parse(`
{{- $mattermostUsername := .Login | lookupMattermostUsername}}
{{- if $mattermostUsername }}@{{$mattermostUsername}}
{{- else}}[{{.Login}}]({{.HTMLURL}})
{{- end -}}
	`))

	// The repo template links to the corresponding repository.
	template.Must(masterTemplate.New("repo").Parse(
		`[\[{{.GetFullName}}\]]({{.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("FRepo").Parse(
		`[\[{{.FullName}}\]]({{.HTMLURL}})`,
	))

	// The eventRepoPullRequest links to the corresponding pull request, anchored at the repo.
	template.Must(masterTemplate.New("eventRepoPullRequest").Parse(
		`[{{.Repo.FullName}}#{{.PullRequest.Number}}]({{.PullRequest.HTMLURL}})`,
	))

	template.Must(masterTemplate.New("eventRepoPullRequestWithTitle").Parse(
		`{{template "eventRepoPullRequest" .}} - {{.PullRequest.Title}}`,
	))

	// The reviewRepoPullRequest links to the corresponding pull request, anchored at the repo.
	template.Must(masterTemplate.New("reviewRepoPullRequest").Parse(
		`[{{.Repo.FullName}}#{{.PullRequest.Number}}]({{.PullRequest.HTMLURL}})`,
	))

	// this reviewRepoPullRequestWithTitle just adds title
	template.Must(masterTemplate.New("reviewRepoPullRequestWithTitle").Parse(
		`{{template "reviewRepoPullRequest" .}} - {{.PullRequest.Title}}`,
	))

	// The pullRequest links to the corresponding pull request, skipping the repo title.
	template.Must(masterTemplate.New("pullRequest").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("FPullRequest").Parse(
		`[#{{.Number}} {{.Title}}]({{.HTMLURL}})`,
	))

	// The issue links to the corresponding issue.
	template.Must(masterTemplate.New("issue").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("FIssue").Parse(
		`[#{{.Number}} {{.Title}}]({{.HTMLURL}})`,
	))

	// The workflow job links to the corresponding workflow.
	template.Must(masterTemplate.New("workflowJob").Parse(
		`[{{.GetName}}]({{.GetHTMLURL}})`,
	))

	// The release links to the corresponding release.
	template.Must(masterTemplate.New("release").Parse(
		`[{{.GetTagName}}]({{.GetHTMLURL}})`,
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
		`[{{.Repo.FullName}}#{{.Issue.Number}}]({{.Comment.HTMLURL}})`,
	))

	// eventRepoIssueFullLinkWithTitle template is sibling of eventRepoIssueWithTitle
	// this one refers to the comment instead of the issue itself
	template.Must(masterTemplate.New("eventRepoIssueFullLinkWithTitle").Parse(
		`{{template "eventRepoIssueFullLink" .}} - {{.Issue.Title}}`,
	))

	template.Must(masterTemplate.New("labels").Funcs(funcMap).Parse(`
{{- if .Labels }}
Labels: {{range $i, $el := .Labels -}}` + "{{- if $i}}, {{end}}[`{{ $el.Name }}`]({{ $.RepositoryURL }}/labels/{{ $el.Name | pathEscape }})" + `{{end -}}
{{ end -}}
`))

	template.Must(masterTemplate.New("FLabels").Funcs(funcMap).Parse(`
{{- if .Labels }}
Labels: {{range $i, $el := .Labels -}}` + "{{- if $i}}, {{end}}[`{{ $el.Name }}`]({{ $.RepositoryURL }}/labels/{{ $el.Name | pathEscape }})" + `{{end -}}
{{ end -}}
`))

	template.Must(masterTemplate.New("subscriptionLabel").Funcs(funcMap).Parse(`
{{- if . }}
{{- if ne . "" }} with the label ` + "`{{.}}`" + `{{- end }}
{{- end -}}
`))

	template.Must(masterTemplate.New("assignee").Funcs(funcMap).Parse(`
{{- if .Assignees }}
Assignees: {{range $i, $el := .Assignees -}} {{- if $i}}, {{end}}{{template "user" $el}}{{end -}}
{{- end -}}
`))

	template.Must(masterTemplate.New("FAssignee").Funcs(funcMap).Parse(`
{{- if .Assignees }}
Assignees: {{range $i, $el := .Assignees -}} {{- if $i}}, {{end}}{{template "FUser" $el}}{{end -}}
{{- end -}}
`))

	template.Must(masterTemplate.New("newDraftPR").Funcs(funcMap).Parse(`
{{template "repo" .Event.GetRepo}} New draft pull request {{template "pullRequest" .Event.GetPullRequest}} was opened by {{template "user" .Event.GetSender}}.
`))

	template.Must(masterTemplate.New("newPR").Funcs(funcMap).Parse(`
{{ if eq .Config.Style "collapsed" -}}
{{template "FRepo" .Event.Repo}} New pull request {{template "FPullRequest" .Event.PullRequest}} was opened by {{template "FUser" .Event.Sender}}{{template "subscriptionLabel" .Label}}.
{{- else -}}
#### {{.Event.PullRequest.Title}}
##### {{template "eventRepoPullRequest" .Event}}
#new-pull-request by {{template "FUser" .Event.Sender}}{{template "subscriptionLabel" .Label}}
{{- if ne .Config.Style "skip-body" -}}
{{- template "FLabels" dict "Labels" .Event.PullRequest.Labels "RepositoryURL" .Event.Repo.HTMLURL  }}
{{- template "FAssignee" .Event.PullRequest }}

{{.Event.PullRequest.Body | removeComments | replaceAllForgejoUsernames}}
{{- end -}}
{{- end }}
`))

	template.Must(masterTemplate.New("markedReadyToReviewPR").Funcs(funcMap).Parse(`
{{ if eq .Config.Style "collapsed" -}}
{{template "repo" .Event.GetRepo}} Pull request {{template "pullRequest" .Event.GetPullRequest}} was marked ready for review by {{template "user" .Event.GetSender}}{{template "subscriptionLabel" (dict "Label" .Label)}}.
{{- else -}}
#### {{.Event.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .Event}}
#new-pull-request by {{template "user" .Event.PullRequest.User}}{{template "subscriptionLabel" .Label}}
{{- if ne .Config.Style "skip-body" -}}
{{- template "labels" dict "Labels" .Event.GetPullRequest.Labels "RepositoryURL" .Event.GetRepo.GetHTMLURL  }}
{{- template "assignee" .Event.GetPullRequest }}

{{.Event.GetPullRequest.GetBody | removeComments | replaceAllForgejoUsernames}}
{{- end -}}
{{- end }}
`))

	template.Must(masterTemplate.New("closedPR").Funcs(funcMap).Parse(`
{{template "FRepo" .Repo}} Pull request {{template "FPullRequest" .PullRequest}} was
{{- if .GetPullRequest.GetMerged }} merged
{{- else }} closed
{{- end }} by {{template "FUser" .Sender}}.
`))

	template.Must(masterTemplate.New("reopenedPR").Funcs(funcMap).Parse(`
{{template "FRepo" .Repo}} Pull request {{template "FPullRequest" .PullRequest}} was reopened by {{template "FUser" .Sender}}.
`))

	template.Must(masterTemplate.New("pullRequestLabelled").Funcs(funcMap).Parse(`
#### {{.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .}}
#pull-request-labeled ` + "`{{.GetLabel.GetName}}`" + ` by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("pullRequestMentionNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} mentioned you on [{{.Repo.FullName}}#{{.PullRequest.Number}}]({{.PullRequest.HTMLURL}}) - {{.PullRequest.Title}}:
{{.PullRequest.Body | trimBody | quote | replaceAllForgejoUsernames}}`))

	template.Must(masterTemplate.New("newIssue").Funcs(funcMap).Parse(`
{{ if eq .Config.Style "collapsed" -}}
{{template "repo" .Event.GetRepo}} New issue {{template "issue" .Event.GetIssue}} opened by {{template "user" .Event.GetSender}}{{template "subscriptionLabel" .Label}}.
{{- else -}}
#### {{.Event.GetIssue.GetTitle}}
##### {{template "eventRepoIssue" .Event}}
#new-issue by {{template "user" .Event.GetSender}}{{template "subscriptionLabel" .Label}}
{{- if ne .Config.Style "skip-body" -}}
{{- template "labels" dict "Labels" .Event.GetIssue.Labels "RepositoryURL" .Event.GetRepo.GetHTMLURL  }}
{{- template "assignee" .Event.GetIssue }}

{{.Event.GetIssue.GetBody | removeComments | replaceAllForgejoUsernames}}
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
{{template "FUser" .Sender}} {{if .Forced}}force-{{end}}pushed [{{len .Commits}} new commit{{if ne (len .Commits) 1}}s{{end}}]({{.Compare}}) to [{{.Repo.FullName}}:{{.Ref | trimRef}}]({{.Repo.HTMLURL}}/src/branch/{{.Ref | trimRef}}):
{{range .Commits -}}
[` + "`{{.ID | substr 0 6}}`" + `]({{.URL}}) {{.Message | trimSpace}} - {{with . | commitAuthor}}{{.Name}}{{end}}
{{end -}}
`))

	template.Must(masterTemplate.New("newCreateMessage").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{.GetRefType}} [{{.GetRef | trimRef}}]({{.GetRepo.GetHTMLURL}}/src/branch/{{.GetRef | trimRef}}) created by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("newDeleteMessage").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{.GetRefType}} {{.GetRef}} deleted by {{template "user" .GetSender}}
`))

	template.Must(masterTemplate.New("issueComment").Funcs(funcMap).Parse(`
{{template "FRepo" .Repo}} New comment by {{template "FUser" .Sender}} on {{template "FIssue" .Issue}}:

{{.Comment.Body | trimBody | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("pullRequestReviewEvent").Funcs(funcMap).Parse(`
{{template "FRepo" .Repo}} {{template "FUser" .Sender}}
{{- if eq .GetReview.GetType "pull_request_review_approved"}} approved
{{- else if eq .GetReview.GetType "commented"}} commented on
{{- else if eq .GetReview.GetType "pull_request_review_rejected"}} requested changes on
{{- end }} {{template "FPullRequest" .PullRequest}}:

{{.Review.Content | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("newReviewComment").Funcs(funcMap).Parse(`
{{template "FRepo" .Repo}} New review comment by {{template "FUser" .Sender}} on {{template "FPullRequest" .PullRequest}}:

{{.Review.Content | trimBody | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentMentionNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} mentioned you on [{{.Repo.FullName}}#{{.Issue.Number}}]({{.Comment.HTMLURL}}) - {{.Issue.Title}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAuthorPullRequestNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} commented on your pull request {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneePullRequestNotification").Funcs(funcMap).Parse(`
{{template "FUser" .GetSender}} commented on pull request you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeIssueNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} commented on an issue you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeSelfMentionPullRequestNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} mentioned you on a pull request that you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAssigneeSelfMentionIssueNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} mentioned you on an issue that you are assigned to {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("commentAuthorIssueNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}} commented on your issue {{template "eventRepoIssueFullLinkWithTitle" .}}:
{{.Comment.Body | trimBody | quote | replaceAllForgejoUsernames}}
`))

	template.Must(masterTemplate.New("pullRequestNotification").Funcs(funcMap).Parse(`
{{template "FUser" .Sender}}
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
{{template "FUser" .Sender}}
{{- if eq .GetReview.GetType "pull_request_review_approved" }} approved your pull request
{{- else if eq .GetReview.GetType "pull_request_review_rejected" }} requested changes on your pull request
{{- else if eq .GetReview.GetType "commented" }} commented on your pull request
{{- end }} {{template "reviewRepoPullRequestWithTitle" .}}
{{if ne .GetReview.GetContent ""}}{{.Review.Content | trimBody | trimSpace | quote | replaceAllForgejoUsernames}}
{{else}}{{end}}`))

	template.Must(masterTemplate.New("helpText").Parse("" +
		"* `/forgejo connect{{if .EnablePrivateRepo}}{{if not .ConnectToPrivateByDefault}} [private]{{end}}{{end}}` - Connect your Mattermost account to your Forgejo account.\n" +
		"{{if .EnablePrivateRepo}}{{if not .ConnectToPrivateByDefault}}" +
		"  * `private` is optional. If used, read access to your private repositories will be requested." +
		"If these repositories send webhook events to this Mattermost server, you'll be notified of changes to those repositories.\n" +
		"{{else}}" +
		"  * Read access to your private repositories will be requested." +
		"If these repositories send webhook events to this Mattermost server, you'll be notified of changes to those repositories.\n" +
		"{{end}}{{end}}" +
		"* `/forgejo disconnect` - Disconnect your Mattermost account from your Forgejo account\n" +
		"* `/forgejo help` - Display Slash Command help text\n" +
		"* `/forgejo todo` - Get a list of unread messages and pull requests awaiting your review\n" +
		"* `/forgejo subscriptions list` - Will list the current channel subscriptions\n" +
		"* `/forgejo subscriptions add owner[/repo] [flags]` - Subscribe the current channel to receive notifications about opened pull requests and issues for an organization or repository\n" +
		"  * `flags` currently supported:\n" +
		"	 * `--features` - a comma-delimited list of one or more of the following:\n" +
		"    	* `issues` - includes new and closed issues\n" +
		"    	* `pulls` - includes new and closed pull requests\n" +
		"    	* `pulls_merged` - includes merged pull requests only\n" +
		"    	* `pulls_created` - includes new pull requests only\n" +
		"    	* `pushes` - includes pushes\n" +
		"    	* `creates` - includes branch and tag creations\n" +
		"    	* `deletes` - includes branch and tag deletions\n" +
		"    	* `issue_comments` - includes new issue comments\n" +
		"    	* `issue_creations` - includes new issues only \n" +
		"    	* `pull_reviews` - includes pull request reviews\n" +
		"    	* `workflow_failure` - includes workflow job failure\n" +
		"    	* `workflow_success` - includes workflow job success\n" +
		"    	* `releases` - includes release created and deleted\n" +
		"    	* `label:<labelname>` - limit pull request and issue events to only this label. Must include `pulls` or `issues` in feature list when using a label.\n" +
		"    	* `discussions` - includes new discussions\n" +
		"    	* `discussion_comments` - includes new discussion comments\n" +
		"    	* Defaults to `pulls,issues,creates,deletes`\n\n" +
		"    * `--exclude-org-member` - events triggered by organization members will not be delivered (the Forgejo organization config should be set, otherwise this flag has not effect)\n" +
		"    * `--render-style` - notifications will be delivered in the specified style (for example, the body of a pull request will not be displayed). Supported values are `collapsed`, `skip-body` or `default` (same as omitting the flag).\n" +
		"* `/forgejo subscriptions delete owner[/repo]` - Unsubscribe the current channel from a repository\n" +
		"* `/forgejo me` - Display the connected Forgejo account\n" +
		"* `/forgejo settings [setting] [value]` - Update your user settings\n" +
		"  * `setting` can be `notifications` or `reminders`\n" +
		"  * `value` can be `on` or `off`\n" +
		"  * `setting` can be `team-review-notifications`\n" +
		"    * `value` can be `on` or `off`\n" +
		"    * When `on`, you can use `--exclude` flag to specify repositories to exclude from team notifications\n" +
		"    * Example: `/forgejo settings team-review-notifications on --exclude repo1,repo2`\n" +
		"* `/forgejo mute` - Managed muted Forgejo users. You'll not receive notifications for comments in your PRs and issues from those users.\n" +
		"  * `/forgejo mute list` - list your muted Forgejo users\n" +
		"  * `/forgejo mute add [username]` - add a Forgejo user to your muted list\n" +
		"  * `/forgejo mute delete [username]` - remove a Forgejo user from your muted list\n" +
		"  * `/forgejo mute delete-all` - unmute all Forgejo users\n"))

	template.Must(masterTemplate.New("newRepoStar").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}}
{{- if eq .GetAction "created" }} starred
{{- else }} unstarred
{{- end }} by {{template "user" .GetSender}}
It now has **{{.GetRepo.GetStargazersCount}}** stars.`))

	template.Must(masterTemplate.New("newWorkflowJob").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{.GetWorkflowJob.GetWorkflowName}} workflow {{if eq .GetWorkflowJob.GetConclusion "success"}}succeeded{{else}}failed{{end}} (triggered by {{template "user" .GetSender}})
{{if eq .GetWorkflowJob.GetConclusion "failure"}}Job failed: {{template "workflowJob" .GetWorkflowJob}}
Step failed: {{.GetWorkflowJob.Steps | workflowJobFailedStep}}
{{end}}Commit: {{.GetRepo.GetHTMLURL}}/commit/{{.GetWorkflowJob.GetHeadSHA}}`))
	template.Must(masterTemplate.New("newReleaseEvent").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{template "user" .GetSender}}
{{- if eq .GetAction "created" }} created a release {{template "release" .GetRelease}}
{{- else if eq .GetAction "deleted" }} deleted a release {{template "release" .GetRelease}}
{{- end -}}`))

	template.Must(masterTemplate.New("newDiscussion").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} started a new discussion [#{{.GetDiscussion.GetNumber}} {{.GetDiscussion.GetTitle}}]({{.GetDiscussion.GetHTMLURL}}) on {{template "repo" .GetRepo}}
`))

	template.Must(masterTemplate.New("newDiscussionComment").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} New comment by {{template "user" .GetSender}} on discussion [#{{.GetDiscussion.GetNumber}} {{.GetDiscussion.GetTitle}}]({{.GetDiscussion.GetHTMLURL}}):

{{.GetComment.GetBody | trimBody | replaceAllForgejoUsernames}}
`))
}

func registerForgejoToUsernameMappingCallback(callback func(string) string) {
	forgejoToUsernameMappingCallback = callback
}

func lookupMattermostUsername(forgejoUsername string) string {
	if forgejoToUsernameMappingCallback == nil {
		return ""
	}

	return forgejoToUsernameMappingCallback(forgejoUsername)
}

func setShowAuthorInCommitNotification(value bool) {
	showAuthorInCommitNotification = value
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
