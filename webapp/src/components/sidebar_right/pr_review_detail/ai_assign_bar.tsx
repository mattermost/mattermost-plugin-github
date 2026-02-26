// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {AIAgent} from '../../../types/github_types';

type Props = {
    selectedCount: number;
    agents: AIAgent[];
    onAssign: (agentMention: string) => void;
    theme: Theme;
};

const AIAssignBar: React.FC<Props> = ({selectedCount, agents, onAssign, theme}) => {
    const defaultMention = agents.length > 0 ? (agents.find((a) => a.is_default)?.mention || agents[0].mention) : '';
    const [selectedAgent, setSelectedAgent] = useState(defaultMention);

    const handleAssign = useCallback(() => {
        if (selectedAgent) {
            onAssign(selectedAgent);
        }
    }, [selectedAgent, onAssign]);

    if (selectedCount === 0 || agents.length === 0) {
        return null;
    }

    return (
        <div
            style={{
                ...styles.container,
                backgroundColor: theme.centerChannelBg,
                borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
                boxShadow: `0 -2px 6px ${changeOpacity(theme.centerChannelColor, 0.1)}`,
            }}
        >
            <span style={{...styles.countText, color: theme.centerChannelColor}}>
                {selectedCount + (selectedCount === 1 ? ' comment selected' : ' comments selected')}
            </span>
            <div style={styles.actions}>
                <select
                    style={{
                        ...styles.select,
                        backgroundColor: changeOpacity(theme.centerChannelColor, 0.05),
                        color: theme.centerChannelColor,
                        border: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
                    }}
                    value={selectedAgent}
                    onChange={(e) => setSelectedAgent(e.target.value)}
                >
                    {agents.map((agent) => (
                        <option
                            key={agent.mention}
                            value={agent.mention}
                        >
                            {agent.name}
                        </option>
                    ))}
                </select>
                <button
                    style={{
                        ...styles.assignButton,
                        backgroundColor: theme.buttonBg,
                        color: theme.buttonColor,
                    }}
                    onClick={handleAssign}
                >
                    {'Assign'}
                </button>
            </div>
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    container: {
        position: 'sticky',
        bottom: 0,
        padding: '10px 12px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        zIndex: 10,
    },
    countText: {
        fontSize: '13px',
        fontWeight: 600,
    },
    actions: {
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
    },
    select: {
        padding: '4px 8px',
        borderRadius: '4px',
        fontSize: '12px',
        outline: 'none',
    },
    assignButton: {
        padding: '5px 14px',
        borderRadius: '4px',
        border: 'none',
        fontSize: '12px',
        fontWeight: 600,
        cursor: 'pointer',
    },
};

export default AIAssignBar;
