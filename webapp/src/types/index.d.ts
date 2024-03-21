type Label = {
    id: number;
    name: string;
    color: CSS.Properties;
}

type User = {
    login: string;
}

type Review = {
    state: string;
    user: User;
}

type Item = PrsDetailsData & {
    id: number;
    title: string;
    created_at: string;
    updated_at: string;
    html_url: string;
    repository_url?: string;
    user: User;
    owner?: User;
    milestone?: {
        title: string;
    }
    repository?: {
        full_name: string;
    }
    labels?: Label[];

    // Assignments
    pullRequest?: unknown;

    // Notifications
    subject?: {
        title: string;
    }
    reason?: string;
}

type GithubItemsProps = {
    items: Item[];
    theme: Theme;
}

type UserSettingsData = {
    sidebar_buttons: string;
    daily_reminder: boolean;
    notifications: boolean;
}

type ConnectedData = {
    connected: boolean;
    github_username: string;
    github_client_id: string;
    enterprise_base_url: string;
    organization: string;
    user_settings: UserSettingsData;
    configuration: Record<string, unknown>;
}

type ConfigurationData = {
    left_sidebar_enabled: boolean;
}

type PrsDetailsData = {
    url: string;
    number: number;
    status?: string;
    mergeable?: boolean;
    requestedReviewers?: string[];
    reviews?: Review[];
}

type GithubIssueData = {
    number: number;
    repository_url: string;
}

type YourReposData = {
    name: string;
    full_name: string;
}

type UnreadsData = {
    html_url: string;
}

type SidebarContentData = {
    prs: GithubIssueData[];
    reviews: GithubIssueData[];
    assignments: GithubIssueData[];
    unreads: UnreadsData[];
}

type MentionsData = {
    id: number;
}

type GithubUsersData = {
    username: string;
    last_try: number;
}

type ShowRhsPluginActionData = {
    type: string;
    state: string;
    pluggableId: string;
}

type CreateIssueModalData = {
    title: string;
    channelId: string;
    postId: string;
}

type AttachCommentToIssueModalForPostIdData = {
    postId: string;
}

type APIError = {
    id?: string;
    message: string;
    status_code: number;
}
