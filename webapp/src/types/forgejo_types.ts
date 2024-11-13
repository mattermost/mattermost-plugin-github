import * as CSS from 'csstype';

import {Theme} from 'mattermost-redux/types/preferences';

export type ForgejoLabel = {
    id: number;
    name: string;
    color: CSS.Properties;
}

type ForgejoUser = {
    login: string;
}

export type Review = {
    state: string;
    user: ForgejoUser;
}

export type ForgejoItem = PrsDetailsData & {
    id: number;
    title: string;
    created_at: string;
    updated_at: string;
    html_url: string;
    repository_url?: string;
    user: ForgejoUser;
    owner?: ForgejoUser;
    milestone?: {
        title: string;
    }
    repository?: {
        full_name: string;
    }
    labels?: ForgejoLabel[];

    // Assignments
    pullRequest?: unknown;

    // Notifications
    subject?: {
        title: string;
    }
    reason?: string;
}

export type ForgejoItemsProps = {
    items: ForgejoItem[];
    theme: Theme;
}

export type UserSettingsData = {
    sidebar_buttons: string;
    daily_reminder: boolean;
    notifications: boolean;
}

export type ConnectedData = {
    connected: boolean;
    forgejo_username: string;
    forgejo_client_id: string;
    base_url: string;
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

export type ForgejoIssueData = {
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
    prs: ForgejoIssueData[];
    reviews: ForgejoIssueData[];
    assignments: ForgejoIssueData[];
    unreads: UnreadsData[];
}

export type MentionsData = {
    id: number;
}

export type ForgejoUsersData = {
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
    reviews: ForgejoIssueData[];
    yourPrs: ForgejoIssueData[];
    yourAssignments: ForgejoIssueData[],
    unreads: UnreadsData[]
    orgs: string[],
    rhsState?: string | null
}
