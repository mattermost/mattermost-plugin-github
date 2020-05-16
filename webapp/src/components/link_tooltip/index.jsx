// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {id as pluginId} from 'manifest';
import {LinkTooltip} from './link_tooltip';

const mapStateToProps = (state, ownProps) => {
    return { connected: state[`plugins-${pluginId}`].connected }
};

export default connect(mapStateToProps, null)(LinkTooltip);
