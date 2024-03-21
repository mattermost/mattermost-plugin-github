import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

export type GlobalState = ReduxGlobalState & {
    'plugins-github': {
        connected: boolean,
        enterpriseURL: string,
        organization: string,
        username: string,
        userSettings: UserSettingsData,
        configuration: ConfigurationData | Record<string, unknown>,
        clientId: string,
        reviewDetails: PrsDetailsData[],
        sidebarContent: SidebarContentData,
        yourRepos: YourReposData[],
        yourPrDetails: PrsDetailsData[],
        mentions: MentionsData[],
        githubUsers: Record<string, GithubUsersData>,
        rhsPluginAction: ShowRhsPluginActionData | null,
        rhsState: string | null,
        isCreateIssueModalVisible: boolean,
        attachCommentToIssueModalVisible: boolean,
        createIssueModal: CreateIssueModalData,
        attachCommentToIssueModalForPostId: string,
    }
};
