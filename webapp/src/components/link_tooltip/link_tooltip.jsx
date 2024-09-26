import React, {useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import './tooltip.css';
import {GitMergeIcon, GitPullRequestIcon, IssueClosedIcon, IssueOpenedIcon} from '@primer/octicons-react';
import ReactMarkdown from 'react-markdown';

import Client from 'client';
import {getLabelFontColor, hexToRGB} from '../../utils/styles';

const maxTicketDescriptionLength = 160;

export const LinkTooltip = ({href, connected, show, theme}) => {
    const [data, setData] = useState(null);
    useEffect(() => {
        const initData = async () => {
            if (href.includes('github.com/')) {
                const [owner, repo, type, number] = href.split('github.com/')[1].split('/');
                if (!owner | !repo | !type | !number) {
                    return;
                }

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
        if (!connected || data || ((typeof (show) !== 'undefined' || show != null) && !show)) {
            return;
        }

        initData();
    }, [connected, data, href, show]);

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
            <div className='github-tooltip'>
                <div
                    className='github-tooltip box github-tooltip--large github-tooltip--bottom-left p-4'
                    style={{backgroundColor: theme.centerChannelBg, border: `1px solid ${hexToRGB(theme.centerChannelColor, '0.16')}`}}
                >
                    <div className='header mb-1'>
                        <span style={{color: theme.centerChannelColor}}>
                            {data.repo}
                        </span>
                        {' on '}
                        <span>{date}</span>
                    </div>

                    <div className='body d-flex mt-2'>
                        <span className='pt-1 pb-1 pr-2'>
                            { getIconElement() }
                        </span>

                        {/* info */}
                        <div className='tooltip-info mt-1'>
                            <a
                                href={href}
                                target='_blank'
                                rel='noopener noreferrer'
                                style={{color: theme.centerChannelColor}}
                            >
                                <h5 className='mr-1'>{data.title}</h5>
                                <span>{'#' + data.number}</span>
                            </a>
                            {data?.user?.login && (
                                <p className='opened-by'>
                                    {'Opened by '}
                                    <a href={`https://github.com/${data.user.login}`}>{data.user.login}</a>
                                </p>
                            )}
                            <div className='markdown-text mt-1 mb-1'>
                                <ReactMarkdown linkTarget='_blank'>{description}</ReactMarkdown>
                            </div>

                            {/* base <- head */}
                            {data.type === 'pull' && (
                                <div className='base-head mt-1 mr-3'>
                                    <span
                                        title={data.base.ref}
                                        className='commit-ref'
                                    >{data.base.ref}
                                    </span>
                                    <span className='mx-1'>{'←'}</span>
                                    <span
                                        title={data.head.ref}
                                        className='commit-ref'
                                    >{data.head.ref}
                                    </span>
                                </div>
                            )}

                            {/* Labels */}
                            <div className='labels mt-3'>
                                {data.labels && data.labels.map((label, idx) => {
                                    return (
                                        <span
                                            key={idx}
                                            className='label mr-1'
                                            title={label.description}
                                            style={{backgroundColor: '#' + label.color, color: getLabelFontColor(label.color)}}
                                        >
                                            <span>{label.name}</span>
                                        </span>
                                    );
                                })}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }
    return null;
};

LinkTooltip.propTypes = {
    href: PropTypes.string.isRequired,
    connected: PropTypes.bool.isRequired,
    theme: PropTypes.object.isRequired,
    show: PropTypes.bool,
};
