// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import GithubIssueSelector from './github_issue_selector';

const mapStateToProps = () => ({});

export default connect(mapStateToProps, null, null, {withRef: true})(GithubIssueSelector);
