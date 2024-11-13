// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getLabelOptions} from '../../actions';

import ForgejoLabelSelector from './forgejo_label_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getLabelOptions}, dispatch),
});

export default connect(
    null,
    mapDispatchToProps,
)(ForgejoLabelSelector);
