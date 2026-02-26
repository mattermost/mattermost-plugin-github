// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {REACTIONS} from '../../../constants';

type ReactionData = {
    content: string;
    count: number;
    reacted: boolean;
};

type Props = {
    reactions: ReactionData[];
    onToggleReaction: (content: string) => Promise<any>;
    theme: Theme;
};

const EMOJI_MAP: Record<string, string> = {
    '+1': '\uD83D\uDC4D',
    '-1': '\uD83D\uDC4E',
    laugh: '\uD83D\uDE04',
    confused: '\uD83D\uDE15',
    heart: '\u2764\uFE0F',
    hooray: '\uD83C\uDF89',
};

const ReactionBar: React.FC<Props> = ({reactions, onToggleReaction, theme}) => {
    const buildInitialState = useCallback(() => {
        const state: Record<string, {count: number; reacted: boolean}> = {};
        for (const r of REACTIONS) {
            const found = reactions.find((rx) => rx.content === r);
            state[r] = {
                count: found ? found.count : 0,
                reacted: found ? found.reacted : false,
            };
        }
        return state;
    }, [reactions]);

    const [localReactions, setLocalReactions] = useState(buildInitialState);

    // Sync with props when reactions change from outside
    React.useEffect(() => {
        setLocalReactions(buildInitialState());
    }, [reactions, buildInitialState]);

    const handleToggle = useCallback(async (content: string) => {
        const current = localReactions[content];
        const newReacted = !current.reacted;
        const newCount = newReacted ? current.count + 1 : Math.max(0, current.count - 1);

        // Optimistic update
        setLocalReactions((prev) => ({
            ...prev,
            [content]: {count: newCount, reacted: newReacted},
        }));

        try {
            await onToggleReaction(content);
        } catch {
            // Revert on error
            setLocalReactions((prev) => ({
                ...prev,
                [content]: {count: current.count, reacted: current.reacted},
            }));
        }
    }, [localReactions, onToggleReaction]);

    return (
        <div style={styles.container}>
            {REACTIONS.map((content) => {
                const data = localReactions[content];
                const isActive = data?.reacted;
                const count = data?.count || 0;
                const emoji = EMOJI_MAP[content] || content;

                return (
                    <button
                        key={content}
                        style={{
                            ...styles.button,
                            backgroundColor: isActive ? changeOpacity(theme.buttonBg, 0.15) : changeOpacity(theme.centerChannelColor, 0.05),
                            border: isActive ? `1px solid ${changeOpacity(theme.buttonBg, 0.4)}` : '1px solid transparent',
                        }}
                        onClick={() => handleToggle(content)}
                        title={content}
                    >
                        <span style={styles.emoji}>{emoji}</span>
                        {count > 0 && <span style={{...styles.count, color: isActive ? theme.buttonBg : theme.centerChannelColor}}>{count}</span>}
                    </button>
                );
            })}
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        display: 'flex',
        flexWrap: 'wrap',
        gap: '4px',
        marginTop: '4px',
    },
    button: {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '2px',
        padding: '2px 6px',
        borderRadius: '10px',
        cursor: 'pointer',
        fontSize: '12px',
        lineHeight: '1.4',
    },
    emoji: {
        fontSize: '13px',
    },
    count: {
        fontSize: '11px',
        fontWeight: 500,
    },
};

export default ReactionBar;
