// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {UserProfile} from 'mattermost-redux/types/users';

import {getPluginState} from '../../selectors';

import {GlobalState} from '../../types/store';

import {getGitHubUser} from '../../actions';

import {UserAttribute} from './user_attribute';

type OwnProps = {
    user: UserProfile;
};

type StateProps = {
    id: string;
    username?: string;
    enterpriseURL: string;
}

function mapStateToProps(state: GlobalState, ownProps: OwnProps): StateProps {
    const mmUserId = ownProps.user ? ownProps.user.id : '';

    const pluginState = getPluginState(state);
    const githubUser = pluginState.githubUsers[mmUserId];

    return {
        id: mmUserId,
        username: githubUser?.username,
        enterpriseURL: pluginState.enterpriseURL,
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            getGitHubUser,
        }, dispatch),
    };
}

type DispatchProps = ReturnType<typeof mapDispatchToProps>;

export type Props = OwnProps & StateProps & DispatchProps;

export default connect(mapStateToProps, mapDispatchToProps)(UserAttribute);
