// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {ReviewThreadData} from '../../../types/github_types';

import ReviewThread from './review_thread';

type Props = {
    filePath: string;
    threads: ReviewThreadData[];
    selectedCommentIds: Set<string>;
    onToggleComment: (commentId: string) => void;
    replyToReviewComment: (owner: string, repo: string, number: number, commentId: number, body: string) => Promise<any>;
    toggleReaction: (owner: string, repo: string, commentId: number, reaction: string) => Promise<any>;
    resolveThread: (threadId: string, action: string) => Promise<any>;
    theme: Theme;
    owner: string;
    repo: string;
    prNumber: number;
};

const FileGroup: React.FC<Props> = ({
    filePath,
    threads,
    selectedCommentIds,
    onToggleComment,
    replyToReviewComment,
    toggleReaction,
    resolveThread,
    theme,
    owner,
    repo,
    prNumber,
}) => {
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div style={{...styles.container, borderBottom: `1px solid ${changeOpacity(theme.centerChannelColor, 0.1)}`}}>
            <div
                style={{
                    ...styles.header,
                    backgroundColor: changeOpacity(theme.centerChannelColor, 0.05),
                }}
                onClick={() => setCollapsed(!collapsed)}
            >
                <span style={{...styles.arrow, transform: collapsed ? 'rotate(-90deg)' : 'rotate(0deg)'}}>
                    {'\u25BE'}
                </span>
                <span style={{...styles.filePath, color: theme.centerChannelColor}}>
                    {filePath}
                </span>
                <span style={{...styles.threadCount, color: changeOpacity(theme.centerChannelColor, 0.5)}}>
                    {threads.length + (threads.length === 1 ? ' thread' : ' threads')}
                </span>
            </div>
            {!collapsed && (
                <div style={styles.threadsList}>
                    {threads.map((thread) => (
                        <ReviewThread
                            key={thread.id}
                            thread={thread}
                            selectedCommentIds={selectedCommentIds}
                            onToggleComment={onToggleComment}
                            replyToReviewComment={replyToReviewComment}
                            toggleReaction={toggleReaction}
                            resolveThread={resolveThread}
                            theme={theme}
                            owner={owner}
                            repo={repo}
                            prNumber={prNumber}
                        />
                    ))}
                </div>
            )}
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        marginBottom: '4px',
    },
    header: {
        padding: '8px 12px',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        gap: '6px',
        userSelect: 'none',
    },
    arrow: {
        fontSize: '12px',
        transition: 'transform 0.15s',
        display: 'inline-block',
    },
    filePath: {
        fontSize: '12px',
        fontWeight: 600,
        fontFamily: 'SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        flex: 1,
    },
    threadCount: {
        fontSize: '11px',
        flexShrink: 0,
    },
    threadsList: {
        padding: '4px 8px',
    },
};

export default FileGroup;
