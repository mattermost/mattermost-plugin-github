// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {getLabelOptions} from '../../actions';

import GithubLabelSelector from './github_label_selector';

const mapDispatchToProps = (dispatch: Dispatch) => ({
    actions: bindActionCreators({getLabelOptions}, dispatch),
}) as unknown as Actions;

type Actions = {
    getLabelOptions: (repoName: string) => ReturnType<ReturnType<typeof getLabelOptions>>;
};

export type GitHubLabelSelectorDispatchProps = {
    actions: Actions;
};

export default connect(
    null,
    mapDispatchToProps,
)(GithubLabelSelector);
