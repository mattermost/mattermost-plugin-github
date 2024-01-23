// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import manifest from 'manifest';

import {LinkTooltip} from './link_tooltip';

const mapStateToProps = (state) => {
    return {connected: state[`plugins-${manifest.id}`].connected};
};

export default connect(mapStateToProps, null)(LinkTooltip);
