import {GitMergeIcon, GitPullRequestIcon, IssueClosedIcon, IssueOpenedIcon, SkipIcon, IconProps} from '@primer/octicons-react';
import PropTypes from 'prop-types';
import React, {useEffect, useState} from 'react';
import ReactMarkdown from 'react-markdown';
import './embed_preview.scss';

import {getLabelFontColor} from '../../utils/styles';
import {isUrlCanPreview} from '../../utils/github_utils';
import Client from '@/client';

const maxTicketDescriptionLength = 160;

interface Label {
    name: string;
    color: string;
    description?: string;
}

interface RepoRef {
    ref: string;
}

interface GitHubData {
    owner: string;
    repo: string;
    type: 'issues' | 'pull';
    state: string;
    merged?: boolean;
    state_reason?: string;
    created_at: string;
    title: string;
    body?: string;
    number: number;
    labels?: Label[];
    base?: RepoRef;
    head?: RepoRef;
}

type LinkEmbedProps = {
    embed: {
        url: string;
    };
    connected: boolean;
};

export const LinkEmbedPreview = ({embed: {url}, connected}: LinkEmbedProps) => {
    const [data, setData] = useState<GitHubData | null>(null);
    useEffect(() => {
        const initData = async () => {
            if (isUrlCanPreview(url)) {
                const [owner, repo, type, number] = url.split('github.com/')[1].split('/');

                try {
                    let issueOrPR: any;

                    if (type === 'issues') {
                        issueOrPR = await Client.getIssue(owner, repo, Number(number));
                    } else if (type === 'pull') {
                        issueOrPR = await Client.getPullRequest(owner, repo, Number(number));
                    } else {
                        return;
                    }

                    if (issueOrPR && !('error' in issueOrPR)) {
                        const githubData: GitHubData = {
                            owner,
                            repo,
                            type: type as 'issues' | 'pull',
                            state: issueOrPR.state || '',
                            created_at: issueOrPR.created_at || '',
                            title: issueOrPR.title || '',
                            number: issueOrPR.number || 0,
                            merged: type === 'pull' ? issueOrPR.merged : undefined,
                            state_reason: issueOrPR.state_reason,
                            body: issueOrPR.body,
                            labels: Array.isArray(issueOrPR.labels) ? issueOrPR.labels : [],
                            base: type === 'pull' && issueOrPR.base ? issueOrPR.base : undefined,
                            head: type === 'pull' && issueOrPR.head ? issueOrPR.head : undefined,
                        };
                        setData(githubData);
                    }
                } catch (error) {
                    console.error('Error fetching GitHub data:', error);
                }
            }
        };

        if (!connected || data) {
            return;
        }

        initData();
    }, [connected, data, url]);

    const getIconElement = () => {
        if (!data) return null;
        
        const iconProps = {
            size: 16, // Use a number instead of 'small'
            verticalAlign: 'text-bottom' as const,
        };

        let icon;
        let colorClass;
        switch (data.type) {
        case 'pull':
            icon = <GitPullRequestIcon {...iconProps}/>;

            colorClass = 'github-preview-icon-open';
            if (data.state === 'closed') {
                if (data.merged) {
                    colorClass = 'github-preview-icon-merged';
                    icon = <GitMergeIcon {...iconProps}/>;
                } else {
                    colorClass = 'github-preview-icon-closed';
                }
            }

            break;
        case 'issues':
            if (data.state === 'open') {
                colorClass = 'github-preview-icon-open';
                icon = <IssueOpenedIcon {...iconProps}/>;
            } else if (data.state_reason === 'not_planned') {
                colorClass = 'github-preview-icon-not-planned';
                icon = <SkipIcon {...iconProps}/>;
            } else {
                colorClass = 'github-preview-icon-merged';
                icon = <IssueClosedIcon {...iconProps}/>;
            }
            break;
        }
        return (
            <span className={`pr-2 ${colorClass}`}>
                {icon}
            </span>
        );
    };

    if (!data) {
        return null;
    }
    const dateObj = new Date(data.created_at);
    const dateStr = dateObj.toDateString();

    let description = '';
    if (data.body) {
        description = data.body.substring(0, maxTicketDescriptionLength).trim();
        if (data.body.length > maxTicketDescriptionLength) {
            description += '...';
        }
    }

    return (
        <div className='github-preview github-preview--large p-4 mt-1 mb-1'>
            <div className='header'>
                <span className='repo'>
                    {data.repo}
                </span>
                {' on '}
                <span>{dateStr}</span>
            </div>

            <div className='body d-flex'>

                {/* info */}
                <div className='preview-info mt-1'>
                    <a
                        href={url}
                        target='_blank'
                        rel='noopener noreferrer'
                    >
                        <h5 className='mr-1'>
                            { getIconElement() }
                            {data.title}
                        </h5>
                        <span>{'#' + data.number}</span>
                    </a>
                    <div className='markdown-text mt-1 mb-1'>
                        <ReactMarkdown linkTarget='_blank'>{description}</ReactMarkdown>
                    </div>

                    <div className='sub-info mt-1'>
                        {/* base <- head */}
                        {data.type === 'pull' && data.base && data.head && (
                            <div className='sub-info-block'>
                                <h6 className='mt-0 mb-1'>{'Base ← Head'}</h6>
                                <div className='base-head'>
                                    <span
                                        title={data.base.ref}
                                        className='commit-ref'
                                    >{data.base.ref}
                                    </span> <span className='mx-1'>{'←'}</span>{' '}
                                    <span
                                        title={data.head.ref}
                                        className='commit-ref'
                                    >{data.head.ref}
                                    </span>
                                </div>
                            </div>
                        )}

                        {/* Labels */}
                        {data.labels && data.labels.length > 0 && (
                            <div className='sub-info-block'>
                                <h6 className='mt-0 mb-1'>{'Labels'}</h6>
                                <div className='labels'>
                                    {data.labels.map((label, idx) => {
                                        return (
                                            <span
                                                key={`${label.name}-${idx}`}
                                                className='label'
                                                title={label.description}
                                                style={{backgroundColor: '#' + label.color, color: getLabelFontColor(label.color)}}
                                            >
                                                <span>{label.name}</span>
                                            </span>
                                        );
                                    })}
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

LinkEmbedPreview.propTypes = {
    embed: {
        url: PropTypes.string.isRequired,
    },
    connected: PropTypes.bool.isRequired,
};
