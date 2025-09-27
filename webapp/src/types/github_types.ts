// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import * as CSS from 'csstype';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

export type GithubLabel = {
    id: number;
    name: string;
    color: CSS.Properties;
}

type GitHubUser = {
    login: string;
}

export type Review = {
    state: string;
    user: GitHubUser;
}

export type GithubItem = PrsDetailsData & {
    id: number;
    title: string;
    created_at: string;
    updated_at: string;
    html_url: string;
    repository_url?: string;
    user: GitHubUser;
    owner?: GitHubUser;
    milestone?: {
        title: string;
    }
    repository?: {
        full_name: string;
    }
    labels?: GithubLabel[];

    // Assignments
    pullRequest?: unknown;

    // Notifications
    subject?: {
        title: string;
    }
    reason?: string;
    additions?: number;
    deletions?: number;
    changed_files?: number;
}

export type GithubItemsProps = {
    items: GithubItem[];
    theme: Theme;
}

export type UserSettingsData = {
    sidebar_buttons: string;
    daily_reminder: boolean;
    notifications: boolean;
}

export type ConnectedData = {
    connected: boolean;
    github_username: string;
    github_client_id: string;
    enterprise_base_url: string;
    organizations: string[];
    user_settings: UserSettingsData;
    configuration: Record<string, unknown>;
}

export type ConfigurationData = {
    left_sidebar_enabled: boolean;
}

export type PrsDetailsData = {
    url: string;
    number: number;
    status?: string;
    mergeable?: boolean;
    requestedReviewers?: string[];
    reviews?: Review[];
}

export type GithubIssueData = {
    id?: number;
    number: number;
    state?: string;
    state_reason?: string;
    locked?: boolean;
    title?: string;
    body?: string;
    author_association?: string;
    user?: GitHubUser;
    labels?: GithubLabel[];
    assignee?: GitHubUser;
    comments?: number;
    closed_at?: string;
    created_at?: string;
    updated_at?: string;
    closed_by?: GitHubUser;
    url?: string;
    html_url?: string;
    comments_url?: string;
    events_url?: string;
    labels_url?: string;
    repository_url: string;
    milestone?: { title: string };
    pull_request?: unknown;
    repository?: { full_name: string };
    reactions?: unknown;
    assignees?: GitHubUser[];
    node_id?: string;
    text_matches?: unknown[];
    active_lock_reason?: string;
}

export type DefaultRepo = {
    name: string;
    full_name: string;
}

export type YourReposData = {
    defaultRepo?: DefaultRepo;
    repos: ReposData[];
}

export type ReposData = {
    name: string;
    full_name: string;
    permissions: Record<string, boolean>;
}

export type UnreadsData = {
    html_url: string;
}

export type SidebarContentData = {
    prs: GithubIssueData[];
    reviews: GithubIssueData[];
    assignments: GithubIssueData[];
    unreads: UnreadsData[];
}

export type MentionsData = {
    id: number;
}

export type GithubUsersData = {
    username?: string;
    last_try: number;
}

export type GitHubPullRequestData = {
    id?: number;
    number?: number;
    state?: string;
    locked?: boolean;
    title?: string;
    body?: string;
    created_at?: string;
    updated_at?: string;
    closed_at?: string;
    merged_at?: string;
    labels?: GithubLabel[];
    user?: GitHubUser;
    draft?: boolean;
    merged?: boolean;
    mergeable?: boolean;
    mergeable_state?: string;
    merged_by?: GitHubUser;
    merge_commit_sha?: string;
    rebaseable?: boolean;
    comments?: number;
    commits?: number;
    additions?: number;
    deletions?: number;
    changed_files?: number;
    url?: string;
    html_url?: string;
    issue_url?: string;
    statuses_url?: string;
    diff_url?: string;
    patch_url?: string;
    commits_url?: string;
    comments_url?: string;
    review_comments_url?: string;
    review_comment_url?: string;
    review_comments?: number;
    assignee?: GitHubUser;
    assignees?: GitHubUser[];
    milestone?: { title: string; number: number };
    maintainer_can_modify?: boolean;
    author_association?: string;
    node_id?: string;
    requested_reviewers?: GitHubUser[];
    auto_merge?: unknown;
    requested_teams?: { name: string; id: number }[];
    links?: unknown;
    head?: { ref: string; sha: string; repo?: { full_name: string } };
    base?: { ref: string; sha: string; repo?: { full_name: string } };
    active_lock_reason?: string;
}

export type MilestoneData = {
    number: number;
    title: string;
}

export type GitHubIssueCommentData = {
    id: number;
}

export type ShowRhsPluginActionData = {
    type: string;
    state: string;
    pluggableId: string;
}

export type CreateIssueModalData = {
    title: string;
    channelId: string;
    postId: string;
}

export type AttachCommentToIssueModalForPostIdData = {
    postId: string;
}

export type APIError = {
    id?: string;
    message: string;
    status_code: number;
}

export type SidebarData = {
    username: string;
    reviews: GithubIssueData[];
    yourPrs: GithubIssueData[];
    yourAssignments: GithubIssueData[],
    unreads: UnreadsData[]
    orgs: string[],
    rhsState?: string | null
}

export type Organization = {
    login: string;
}
export type RepositoriesByOrg = {
    name: string;
    fullName: string;
}

export type RepositoryData = {
    name: string;
    full_name: string;
    permissions: Record<string, boolean>;
};

export type ChannelRepositoriesData = {
    channel_id: string;
    repositories: RepositoryData[];
};
