// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import '@testing-library/jest-dom';

global.fetch = jest.fn(() => Promise.resolve({json: () => Promise.resolve({})})) as jest.Mock;
