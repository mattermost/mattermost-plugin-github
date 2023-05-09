package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

const (
	dailySummary                 = "_dailySummary"
	NotificationReasonSubscribed = "subscribed"
)

func (a *App) CheckIfDuplicateDailySummary(userID, text string) (bool, error) {
	previousSummary, err := a.GetDailySummaryText(userID)
	if err != nil {
		return false, err
	}
	if previousSummary == text {
		return true, nil
	}

	return false, nil
}

func (a *App) StoreDailySummaryText(userID, summaryText string) error {
	_, err := a.client.KV.Set(userID+dailySummary, []byte(summaryText))
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GetDailySummaryText(userID string) (string, error) {
	var summaryByte []byte
	err := a.client.KV.Get(userID+dailySummary, summaryByte)
	if err != nil {
		return "", err
	}

	return string(summaryByte), nil
}

func (a *App) PostToDo(info *GitHubUserInfo, userID string) error {
	ctx := context.Background()
	text, err := a.GetToDo(ctx, info.GitHubUsername, a.GithubConnectUser(ctx, info))
	if err != nil {
		return err
	}

	if info.Settings.DailyReminderOnChange {
		isSameSummary, err := a.CheckIfDuplicateDailySummary(userID, text)
		if err != nil {
			return err
		}
		if isSameSummary {
			return nil
		}
		err = a.StoreDailySummaryText(userID, text)
		if err != nil {
			return err
		}
	}
	a.CreateBotDMPost(info.UserID, text, "custom_git_todo")
	return nil
}

func (a *App) GetToDo(ctx context.Context, username string, githubClient *github.Client) (string, error) {
	config := a.config.GetConfiguration()
	baseURL := config.GetBaseURL()

	issueResults, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Error occurred while searching for reviews")
	}

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while listing notifications")
	}

	yourPrs, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while searching for PRs")
	}

	yourAssignments, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while searching for assignments")
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.GetReason() == NotificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			a.client.Log.Warn("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if a.CheckOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		notificationSubject := n.GetSubject()
		notificationType := notificationSubject.GetType()
		switch notificationType {
		case "RepositoryVulnerabilityAlert":
			message := fmt.Sprintf("[Vulnerability Alert for %v](%v)", n.GetRepository().GetFullName(), fixGithubNotificationSubjectURL(n.GetSubject().GetURL(), ""))
			notificationContent += fmt.Sprintf("* %v\n", message)
		default:
			issueURL := n.GetSubject().GetURL()
			issueNumIndex := strings.LastIndex(issueURL, "/")
			issueNum := issueURL[issueNumIndex+1:]
			subjectURL := n.GetSubject().GetURL()
			if n.GetSubject().GetLatestCommentURL() != "" {
				subjectURL = n.GetSubject().GetLatestCommentURL()
			}

			notificationTitle := notificationSubject.GetTitle()
			notificationURL := fixGithubNotificationSubjectURL(subjectURL, issueNum)
			notificationContent += getToDoDisplayText(baseURL, notificationTitle, notificationURL, notificationType)
		}

		notificationCount++
	}

	if notificationCount == 0 {
		text += "You don't have any unread messages.\n"
	} else {
		text += fmt.Sprintf("You have %v unread messages:\n", notificationCount)
		text += notificationContent
	}

	text += "##### Review Requests\n"

	if issueResults.GetTotal() == 0 {
		text += "You don't have any pull requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", issueResults.GetTotal())

		for _, pr := range issueResults.Issues {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "")
		}
	}

	text += "##### Your Open Pull Requests\n"

	if yourPrs.GetTotal() == 0 {
		text += "You don't have any open pull requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open pull requests:\n", yourPrs.GetTotal())

		for _, pr := range yourPrs.Issues {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "")
		}
	}

	text += "##### Your Assignments\n"

	if yourAssignments.GetTotal() == 0 {
		text += "You don't have any assignments.\n"
	} else {
		text += fmt.Sprintf("You have %v assignments:\n", yourAssignments.GetTotal())

		for _, assign := range yourAssignments.Issues {
			text += getToDoDisplayText(baseURL, assign.GetTitle(), assign.GetHTMLURL(), "")
		}
	}

	return text, nil
}

func (a *App) HasUnreads(info *GitHubUserInfo) bool {
	username := info.GitHubUsername
	ctx := context.Background()
	githubClient := a.GithubConnectUser(ctx, info)
	config := a.config.GetConfiguration()

	query := getReviewSearchQuery(username, config.GitHubOrg)
	issues, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		a.client.Log.Warn("Failed to search for review", "query", query, "error", err.Error())
		return false
	}

	query = getYourPrsSearchQuery(username, config.GitHubOrg)
	yourPrs, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		a.client.Log.Warn("Failed to search for PRs", "query", query, "error", "error", err.Error())
		return false
	}

	query = getYourAssigneeSearchQuery(username, config.GitHubOrg)
	yourAssignments, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		a.client.Log.Warn("Failed to search for assignments", "query", query, "error", "error", err.Error())
		return false
	}

	relevantNotifications := false
	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		a.client.Log.Warn("Failed to list notifications", "error", err.Error())
		return false
	}

	for _, n := range notifications {
		if n.GetReason() == NotificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			a.client.Log.Warn("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.app.CheckOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		relevantNotifications = true
		break
	}

	if issues.GetTotal() == 0 && !relevantNotifications && yourPrs.GetTotal() == 0 && yourAssignments.GetTotal() == 0 {
		return false
	}

	return true
}
