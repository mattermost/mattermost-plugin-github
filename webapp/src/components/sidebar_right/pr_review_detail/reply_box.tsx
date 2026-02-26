// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

type Props = {
    onSubmit: (body: string) => Promise<any>;
    theme: Theme;
};

const ReplyBox: React.FC<Props> = ({onSubmit, theme}) => {
    const [text, setText] = useState('');
    const [expanded, setExpanded] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    const handleFocus = useCallback(() => {
        setExpanded(true);
    }, []);

    const handleBlur = useCallback(() => {
        if (!text.trim()) {
            setExpanded(false);
        }
    }, [text]);

    const handleSubmit = useCallback(async () => {
        const body = text.trim();
        if (!body || submitting) {
            return;
        }

        setSubmitting(true);
        try {
            await onSubmit(body);
            setText('');
            setExpanded(false);
        } finally {
            setSubmitting(false);
        }
    }, [text, submitting, onSubmit]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            handleSubmit();
        }
    }, [handleSubmit]);

    return (
        <div style={styles.container}>
            <textarea
                style={{
                    ...styles.textarea,
                    backgroundColor: changeOpacity(theme.centerChannelColor, 0.05),
                    color: theme.centerChannelColor,
                    border: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
                }}
                rows={expanded ? 4 : 1}
                value={text}
                onChange={(e) => setText(e.target.value)}
                onFocus={handleFocus}
                onBlur={handleBlur}
                onKeyDown={handleKeyDown}
                placeholder='Reply...'
            />
            {expanded && (
                <div style={styles.actions}>
                    <button
                        style={{
                            ...styles.submitButton,
                            backgroundColor: text.trim() ? theme.buttonBg : changeOpacity(theme.centerChannelColor, 0.3),
                            color: text.trim() ? theme.buttonColor : changeOpacity(theme.centerChannelColor, 0.5),
                            cursor: text.trim() && !submitting ? 'pointer' : 'default',
                        }}
                        onClick={handleSubmit}
                        disabled={!text.trim() || submitting}
                    >
                        {submitting ? 'Sending...' : 'Reply'}
                    </button>
                </div>
            )}
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        marginTop: '8px',
    },
    textarea: {
        width: '100%',
        padding: '6px 8px',
        borderRadius: '4px',
        fontSize: '13px',
        resize: 'none',
        outline: 'none',
        fontFamily: 'inherit',
        boxSizing: 'border-box',
    },
    actions: {
        display: 'flex',
        justifyContent: 'flex-end',
        marginTop: '4px',
    },
    submitButton: {
        padding: '4px 12px',
        borderRadius: '4px',
        border: 'none',
        fontSize: '12px',
        fontWeight: 600,
    },
};

export default ReplyBox;
