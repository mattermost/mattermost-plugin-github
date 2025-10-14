// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect} from 'react';

export const useMount = (callback: () => void) => {
    useEffect(() => {
        callback();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
};
