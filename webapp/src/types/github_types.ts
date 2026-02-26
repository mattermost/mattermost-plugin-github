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
    onSelectPR?: (prData: SelectedPRData) => void;
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
    number: number;
    repository_url: string;
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
    id: number;
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

export interface ReviewCommentReaction {
    id: number;
    content: string;
    user: {login: string};
}

export interface ReviewCommentData {
    id: string;
    database_id: number;
    body: string;
    author: {login: string; avatar_url: string};
    created_at: string;
    updated_at: string;
    url: string;
    diff_hunk: string;
    path: string;
    line: number;
    start_line: number;
    reactions: {content: string; count: number; reacted: boolean}[];
}

export interface ReviewThreadData {
    id: string;
    is_resolved: boolean;
    resolved_by: {login: string} | null;
    path: string;
    line: number;
    start_line: number;
    diff_hunk: string;
    comments: ReviewCommentData[];
}

export interface PRReviewSummary {
    approved: number;
    changes_requested: number;
    unresolved_threads: number;
    total_threads: number;
}

export interface PRReviewThreadsData {
    pr_title: string;
    pr_number: number;
    pr_url: string;
    summary: PRReviewSummary;
    threads: ReviewThreadData[];
}

export interface SelectedPRData {
    owner: string;
    repo: string;
    number: number;
    title: string;
    url: string;
}

export interface AIAgent {
    name: string;
    mention: string;
    is_default: boolean;
}

export interface AIAgentsData {
    agents: AIAgent[];
}

export interface ResolveThreadResponse {
    status: string;
    is_resolved: boolean;
}

export interface ReactionToggleResponse {
    id: number;
    content: string;
    toggled: boolean;
}
