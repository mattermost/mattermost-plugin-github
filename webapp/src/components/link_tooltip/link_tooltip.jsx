import React, {useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import './tooltip.css';
import Octicon, {GitMerge, GitPullRequest, IssueClosed, IssueOpened} from '@primer/octicons-react';
import ReactMarkdown from 'react-markdown';

import Client from 'client';
import {getLabelFontColor} from '../../utils/styles';

export const LinkTooltip = ({href}) => {
    const [data, setData] = useState(null);

    useEffect(() => {
        const init = async () => {
            if (href.includes('github.com/')) {
                const [owner, repo, type, number] = href.split('github.com/')[1].split('/');
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
        init();
    }, []);

    const getIconElement = () => {
        let icon;
        let color;
        let iconType;
        switch (data.type) {
        case 'pull':
            color = '#28a745';
            iconType = GitPullRequest;
            if (data.state === 'closed') {
                if (data.merged) {
                    color = '#6f42c1';
                    iconType = GitMerge;
                } else {
                    color = '#cb2431';
                }
            }
            icon = (
                <span style={{color}}>
                    <Octicon
                        icon={iconType}
                        size='small'
                        verticalAlign='middle'
                    />
                </span>
            );
            break;
        case 'issues':
            color = data.state === 'open' ? '#28a745' : '#cb2431';
            iconType = data.state === 'open' ? IssueOpened : IssueClosed;
            icon = (
                <span style={{color}}>
                    <Octicon
                        icon={iconType}
                        size='small'
                        verticalAlign='middle'
                    />
                </span>
            );
            break;
        }
        return icon;
    };

    if (data) {
        let date = new Date(data.created_at);
        date = date.toDateString();

        return (
            <div className='github-tooltip'>
                <div className='github-tooltip box github-tooltip--large github-tooltip--bottom-left p-4'>
                    <div className='header mb-1'>
                        <a
                            title={data.repo}
                            href={href}
                        >{data.repo}</a>&nbsp;on&nbsp;<span>{date}</span>
                    </div>

                    <div className='body d-flex mt-2'>
                        <span className='pt-1 pb-1 pr-2'>
                            { getIconElement() }
                        </span>

                        {/* info */}
                        <div className='tooltip-info mt-1'>
                            <a href={href}>
                                <h5 className='mr-1'>{data.title}</h5>
                                <span>#{data.number}</span>
                            </a>
                            <div className='markdown-text mt-1 mb-1'>
                                <ReactMarkdown
                                    source={data.body}
                                    disallowedTypes={['heading']}
                                    linkTarget='_blank'
                                />
                            </div>

                            {/* base <- head */}
                            {data.type === 'pull' && (
                                <div className='base-head mt-1 mr-3'>
                                    <span
                                        title={data.base.ref}
                                        className='commit-ref'
                                        style={{maxWidth: '140px'}}
                                    >{data.base.ref}
                                    </span>
                                    <span className='mx-1'>←</span>
                                    <span
                                        title={data.head.ref}
                                        className='commit-ref'
                                    >{data.head.ref}
                                    </span>
                                </div>
                            )}

                            <div className='see-more mt-1'>
                                <a
                                    href={href}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                >See more</a>
                            </div>

                            {/* Labels */}
                            <div className='labels mt-3'>
                                {data.labels && data.labels.map((label, idx) => {
                                    return (
                                        <span
                                            key={idx}
                                            className='label mr-1'
                                            title={label.description}
                                            style={{backgroundColor: '#' + label.color}}
                                        >
                                            <span style={{color: getLabelFontColor(label.color)}}>{label.name}</span>
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
};
