import {GitMergeIcon, GitPullRequestIcon, IssueClosedIcon, IssueOpenedIcon} from '@primer/octicons-react';
import PropTypes from 'prop-types';
import React, {useEffect, useState} from 'react';
import ReactMarkdown from 'react-markdown';
import './preview.css';

import Client from 'client';
import {getLabelFontColor} from '../../utils/styles';
import {isUrlCanPreview} from 'src/utils/github_utils';

const maxTicketDescriptionLength = 160;

export const LinkPreview = ({embed: {url}, connected}) => {
    const [data, setData] = useState(null);
    useEffect(() => {
        const initData = async () => {
            if (isUrlCanPreview(url)) {
                const [owner, repo, type, number] = url.split('github.com/')[1].split('/');

                let res;
                switch (type) {
                case 'issues':
                    res = await Client.getIssue(owner, repo, number);
                    break;
                case 'pull':
                    res = await Client.getPullRequest(owner, repo, number);
                    break;
                }
                if (res) {
                    res.owner = owner;
                    res.repo = repo;
                    res.type = type;
                }
                setData(res);
            }
        };

        // show is not provided for Mattermost Server < 5.28
        if (!connected || data) {
            return;
        }

        initData();
    }, [connected, data, url]);

    const getIconElement = () => {
        const iconProps = {
            size: 'small',
            verticalAlign: 'text-bottom',
        };

        let icon;
        let color;
        switch (data.type) {
        case 'pull':
            icon = <GitPullRequestIcon {...iconProps}/>;

            color = '#28a745';
            if (data.state === 'closed') {
                if (data.merged) {
                    color = '#6f42c1';
                    icon = <GitMergeIcon {...iconProps}/>;
                } else {
                    color = '#cb2431';
                }
            }

            break;
        case 'issues':
            color = data.state === 'open' ? '#28a745' : '#cb2431';

            if (data.state === 'open') {
                icon = <IssueOpenedIcon {...iconProps}/>;
            } else {
                icon = <IssueClosedIcon {...iconProps}/>;
            }
            break;
        }
        return (
            <span style={{color}}>
                {icon}
            </span>
        );
    };

    if (data) {
        let date = new Date(data.created_at);
        date = date.toDateString();

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
                    <span>{date}</span>
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
                                <span className='pr-2'>
                                    { getIconElement() }
                                </span>
                                {data.title}
                            </h5>
                            <span>{'#' + data.number}</span>
                        </a>
                        <div className='markdown-text mt-1 mb-1'>
                            <ReactMarkdown linkTarget='_blank'>{description}</ReactMarkdown>
                        </div>

                        <div className='sub-info mt-1'>
                            {/* base <- head */}
                            {data.type === 'pull' && (
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
    }
    return null;
};

LinkPreview.propTypes = {
    embed: {
        url: PropTypes.string.isRequired,
    },
    connected: PropTypes.bool.isRequired,
};
