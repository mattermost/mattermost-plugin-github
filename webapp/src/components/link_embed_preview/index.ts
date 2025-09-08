// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {LinkEmbedPreview} from './link_embed_preview';
import { GlobalState } from '@/types/store';
import { getPluginState } from '@/selectors';

const mapStateToProps = (state: GlobalState) => {
    return {connected: getPluginState(state).connected};
};

export default connect(mapStateToProps, null)(LinkEmbedPreview);
