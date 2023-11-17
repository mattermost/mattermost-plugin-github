export type GithubPluginSettings = {
    connecttoprivatebydefault: string | null;
    enablecodepreview: string;
    enableleftsidebar: boolean;
    enableprivaterepo: boolean | null;
    enablewebhookeventlogging: boolean;
    encryptionkey: string;
    enterprisebaseurl: string;
    enterpriseuploadurl: string;
    githuboauthclientid: string;
    githuboauthclientsecret: string;
    githuborg: string | null;
    usepreregisteredapplication: boolean;
    webhooksecret: string;
}

export const githubConfig: GithubPluginSettings = {
    githuboauthclientid: '',
    githuboauthclientsecret: '',

    connecttoprivatebydefault: null,
    enablecodepreview: 'public',
    enableleftsidebar: true,
    enableprivaterepo: null,
    enablewebhookeventlogging: false,
    encryptionkey: 'S9YasItflsENXnrnKUhMJkdosXTsr6Tc',
    enterprisebaseurl: '',
    enterpriseuploadurl: '',
    githuborg: null,
    usepreregisteredapplication: false,
    webhooksecret: 'w7HfrdZ+mtJKnWnsmHMh8eKzWpQH7xET',
};
