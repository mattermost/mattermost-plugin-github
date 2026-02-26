// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {ReviewThreadData} from '../../../types/github_types';

import DiffHunkDisplay from './diff_hunk_display';
import ReviewComment from './review_comment';
import ReplyBox from './reply_box';

type Props = {
    thread: ReviewThreadData;
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

const ReviewThread: React.FC<Props> = ({
    thread,
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
    const [expandedResolved, setExpandedResolved] = useState(false);

    const isResolved = thread.is_resolved;
    const isCollapsed = isResolved && !expandedResolved;

    const firstComment = thread.comments?.[0];
    const threadCommentId = firstComment?.id || thread.id;
    const isSelected = selectedCommentIds.has(threadCommentId);

    const handleReply = useCallback(async (body: string) => {
        if (!firstComment) {
            return Promise.resolve();
        }
        return replyToReviewComment(owner, repo, prNumber, firstComment.database_id, body);
    }, [replyToReviewComment, owner, repo, prNumber, firstComment]);

    const handleResolveToggle = useCallback(async () => {
        const action = isResolved ? 'unresolve' : 'resolve';
        return resolveThread(thread.id, action);
    }, [resolveThread, thread.id, isResolved]);

    const handleCheckboxChange = useCallback(() => {
        onToggleComment(threadCommentId);
    }, [onToggleComment, threadCommentId]);

    if (isCollapsed) {
        return (
            <div
                style={{
                    ...styles.resolvedCollapsed,
                    backgroundColor: changeOpacity(theme.centerChannelColor, 0.03),
                    borderLeft: `3px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
                }}
                onClick={() => setExpandedResolved(true)}
            >
                <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                    <input
                        type='checkbox'
                        checked={isSelected}
                        onChange={(e) => {
                            e.stopPropagation();
                            handleCheckboxChange();
                        }}
                        onClick={(e) => e.stopPropagation()}
                        style={styles.checkbox}
                    />
                    <span style={{...styles.resolvedLabel, color: changeOpacity(theme.centerChannelColor, 0.5)}}>
                        {'Resolved thread'}
                        {thread.resolved_by ? ` by ${thread.resolved_by.login}` : ''}
                        {' - '}
                        {firstComment?.body ? firstComment.body.substring(0, 80) + (firstComment.body.length > 80 ? '...' : '') : ''}
                    </span>
                </div>
                <span style={{...styles.expandHint, color: changeOpacity(theme.centerChannelColor, 0.4)}}>
                    {'Click to expand'}
                </span>
            </div>
        );
    }

    return (
        <div
            style={{
                ...styles.container,
                backgroundColor: isResolved ? changeOpacity(theme.centerChannelColor, 0.03) : 'transparent',
                borderLeft: `3px solid ${isResolved ? changeOpacity(theme.centerChannelColor, 0.2) : changeOpacity(theme.buttonBg, 0.5)}`,
            }}
        >
            <div style={styles.threadHeader}>
                <input
                    type='checkbox'
                    checked={isSelected}
                    onChange={handleCheckboxChange}
                    style={styles.checkbox}
                />
                {isResolved && (
                    <span style={{...styles.resolvedBadge, color: changeOpacity(theme.centerChannelColor, 0.5)}}>
                        {'Resolved'}
                    </span>
                )}
            </div>

            {firstComment?.diff_hunk && (
                <DiffHunkDisplay diffHunk={firstComment.diff_hunk}/>
            )}

            {thread.comments.map((comment) => (
                <ReviewComment
                    key={comment.id}
                    comment={comment}
                    toggleReaction={toggleReaction}
                    owner={owner}
                    repo={repo}
                    theme={theme}
                />
            ))}

            <div style={styles.threadActions}>
                <button
                    style={{
                        ...styles.resolveButton,
                        color: isResolved ? theme.dndIndicator : theme.onlineIndicator,
                        border: `1px solid ${isResolved ? changeOpacity(theme.dndIndicator, 0.3) : changeOpacity(theme.onlineIndicator, 0.3)}`,
                    }}
                    onClick={handleResolveToggle}
                >
                    {isResolved ? 'Unresolve' : 'Resolve'}
                </button>
                {isResolved && (
                    <button
                        style={{...styles.collapseButton, color: changeOpacity(theme.centerChannelColor, 0.5)}}
                        onClick={() => setExpandedResolved(false)}
                    >
                        {'Collapse'}
                    </button>
                )}
            </div>

            <ReplyBox
                onSubmit={handleReply}
                theme={theme}
            />
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        padding: '10px 12px',
        marginBottom: '8px',
        borderRadius: '4px',
    },
    resolvedCollapsed: {
        padding: '8px 12px',
        marginBottom: '8px',
        borderRadius: '4px',
        cursor: 'pointer',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    resolvedLabel: {
        fontSize: '12px',
        fontStyle: 'italic',
    },
    expandHint: {
        fontSize: '11px',
        flexShrink: 0,
    },
    threadHeader: {
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        marginBottom: '8px',
    },
    checkbox: {
        cursor: 'pointer',
        margin: 0,
    },
    resolvedBadge: {
        fontSize: '11px',
        fontWeight: 600,
        textTransform: 'uppercase' as const,
    },
    threadActions: {
        display: 'flex',
        gap: '8px',
        marginTop: '8px',
    },
    resolveButton: {
        padding: '3px 10px',
        borderRadius: '4px',
        background: 'transparent',
        fontSize: '12px',
        fontWeight: 600,
        cursor: 'pointer',
    },
    collapseButton: {
        padding: '3px 10px',
        borderRadius: '4px',
        background: 'transparent',
        border: 'none',
        fontSize: '12px',
        cursor: 'pointer',
    },
};

export default ReviewThread;
