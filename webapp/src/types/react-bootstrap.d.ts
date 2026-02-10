// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Type override to fix react-bootstrap 1.x compatibility with React 18 types
// The react-bootstrap 1.x types use an older React type definition that
// conflicts with React 18's stricter ReactNode types.

import 'react-bootstrap';

declare module 'react-bootstrap' {
    import {ComponentType, ReactNode} from 'react';

    export const Tooltip: ComponentType<{
        id: string;
        children?: ReactNode;
        [key: string]: unknown;
    }>;

    export const Badge: ComponentType<{
        variant?: string;
        children?: ReactNode;
        [key: string]: unknown;
    }>;

    export const OverlayTrigger: ComponentType<{
        placement?: string;
        overlay: ReactNode;
        children?: ReactNode;
        [key: string]: unknown;
    }>;

    export const Modal: ComponentType<{
        show?: boolean;
        onHide?: () => void;
        backdrop?: string | boolean;
        dialogClassName?: string;
        bsSize?: string;
        children?: ReactNode;
        [key: string]: unknown;
    }>;
}
