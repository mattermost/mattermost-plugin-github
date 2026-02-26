// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';

type Props = {
    diffHunk: string;
};

const MAX_VISIBLE_LINES = 8;

const DiffHunkDisplay: React.FC<Props> = ({diffHunk}) => {
    const [expanded, setExpanded] = useState(false);

    if (!diffHunk) {
        return null;
    }

    const lines = diffHunk.split('\n');
    const isLong = lines.length > MAX_VISIBLE_LINES;
    const visibleLines = expanded ? lines : lines.slice(0, MAX_VISIBLE_LINES);

    const getLineStyle = (line: string): React.CSSProperties => {
        if (line.startsWith('@@')) {
            return {backgroundColor: 'rgba(0, 90, 160, 0.15)', color: '#555'};
        }
        if (line.startsWith('+')) {
            return {backgroundColor: 'rgba(40, 167, 69, 0.15)'};
        }
        if (line.startsWith('-')) {
            return {backgroundColor: 'rgba(220, 53, 69, 0.15)'};
        }
        return {};
    };

    return (
        <div style={styles.container}>
            <pre style={styles.pre}>
                {visibleLines.map((line, idx) => (
                    <div
                        key={idx}
                        style={{...styles.line, ...getLineStyle(line)}}
                    >
                        {line}
                    </div>
                ))}
            </pre>
            {isLong && !expanded && (
                <button
                    style={styles.expandButton}
                    onClick={() => setExpanded(true)}
                >
                    {'Show more (' + (lines.length - MAX_VISIBLE_LINES) + ' more lines)'}
                </button>
            )}
            {isLong && expanded && (
                <button
                    style={styles.expandButton}
                    onClick={() => setExpanded(false)}
                >
                    {'Show less'}
                </button>
            )}
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        borderRadius: '4px',
        overflow: 'hidden',
        marginBottom: '8px',
        border: '1px solid rgba(0, 0, 0, 0.1)',
    },
    pre: {
        margin: 0,
        padding: '4px 0',
        fontFamily: 'SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace',
        fontSize: '11px',
        lineHeight: '1.4',
        overflowX: 'auto',
    },
    line: {
        padding: '0 8px',
        whiteSpace: 'pre',
    },
    expandButton: {
        display: 'block',
        width: '100%',
        padding: '4px',
        border: 'none',
        background: 'rgba(0, 0, 0, 0.03)',
        cursor: 'pointer',
        fontSize: '11px',
        color: '#555',
        textAlign: 'center',
    },
};

export default DiffHunkDisplay;
