import * as CSS from 'csstype';

import {Theme} from 'mattermost-redux/types/preferences';

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
    number: number;
    repository_url: string;
}

export type YourReposData = {
    name: string;
    full_name: string;
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
    username: string;
    last_try: number;
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
