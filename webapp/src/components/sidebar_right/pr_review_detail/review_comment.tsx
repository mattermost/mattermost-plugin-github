// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {ReviewCommentData} from '../../../types/github_types';
import {formatTimeSince} from '../../../utils/date_utils';

import ReactionBar from './reaction_bar';

type Props = {
    comment: ReviewCommentData;
    toggleReaction: (owner: string, repo: string, commentId: number, reaction: string) => Promise<any>;
    owner: string;
    repo: string;
    theme: Theme;
};

const ReviewComment: React.FC<Props> = ({comment, toggleReaction, owner, repo, theme}) => {
    const handleToggleReaction = useCallback((content: string) => {
        return toggleReaction(owner, repo, comment.database_id, content);
    }, [toggleReaction, owner, repo, comment.database_id]);

    const timeSince = formatTimeSince(comment.created_at);

    return (
        <div style={{...styles.container, borderBottom: `1px solid ${changeOpacity(theme.centerChannelColor, 0.1)}`}}>
            <div style={styles.header}>
                {comment.author?.avatar_url && (
                    <img
                        src={comment.author.avatar_url}
                        alt={comment.author.login}
                        style={styles.avatar}
                    />
                )}
                <strong style={{fontSize: '13px', color: theme.centerChannelColor}}>
                    {comment.author?.login || 'unknown'}
                </strong>
                <span style={{...styles.timestamp, color: changeOpacity(theme.centerChannelColor, 0.6)}}>
                    {timeSince + ' ago'}
                </span>
            </div>
            <div style={{...styles.body, color: theme.centerChannelColor}}>
                {comment.body}
            </div>
            {comment.reactions && comment.reactions.length > 0 && (
                <ReactionBar
                    reactions={comment.reactions}
                    onToggleReaction={handleToggleReaction}
                    theme={theme}
                />
            )}
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        padding: '8px 0',
    },
    header: {
        display: 'flex',
        alignItems: 'center',
        gap: '6px',
        marginBottom: '4px',
    },
    avatar: {
        width: '20px',
        height: '20px',
        borderRadius: '50%',
    },
    timestamp: {
        fontSize: '12px',
        marginLeft: 'auto',
    },
    body: {
        fontSize: '13px',
        lineHeight: '1.5',
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-word',
    },
};

export default ReviewComment;
