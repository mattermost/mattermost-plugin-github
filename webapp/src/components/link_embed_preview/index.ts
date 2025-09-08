// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {GlobalState} from '@/types/store';
import {getPluginState} from '@/selectors';

import {LinkEmbedPreview} from './link_embed_preview';

const mapStateToProps = (state: GlobalState) => {
    return {connected: getPluginState(state).connected};
};

// Use a more direct approach with type assertion
// @ts-ignore - Ignoring type errors for connect function
export default connect(mapStateToProps)(LinkEmbedPreview);
