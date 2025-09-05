// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Action} from 'redux';
import type {ThunkAction} from 'redux-thunk';

import {GlobalState} from './store';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type ActionResult<Data = any, Error = any> = {
    data?: Data;
    error?: Error;
};

export type ActionFuncAsync<Data = unknown, Error = unknown> = ThunkAction<Promise<ActionResult<Data, Error>>, GlobalState, unknown, Action>;
