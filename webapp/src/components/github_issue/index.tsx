import * as React from 'react';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';
import {Theme} from 'mattermost-redux/types/preferences';
import {Post} from 'mattermost-redux/types/posts';
import {useDispatch} from 'react-redux';

import {openCreateCommentOnIssueModal, openCreateOrUpdateIssueModal, openCloseOrReopenIssueModal} from '../../actions';

type GithubIssueProps = {
    theme: Theme,
    post: Post,
}

const GithubIssue = ({theme, post}: GithubIssueProps) => {
    const style = getStyle(theme);
    const postProps = post.props || {};
    let assignees;
    let labels;
    const dispatch = useDispatch();

    const issue = {
        repo_owner: postProps.repo_owner,
        repo_name: postProps.repo_name,
        issue_number: postProps.issue_number,
        postId: post.id,
        status: postProps.status,
        channel_id: post.channel_id,
    };

    const content = (
        <div style={style.button_container}>
            <button
                className='btn btn-primary'
                onClick={() => dispatch(openCreateCommentOnIssueModal(issue))}
            >{'Comment'}</button>
            <button
                className='btn btn-tertiary'
                onClick={() => dispatch(openCreateOrUpdateIssueModal(issue))}
            >{'Edit'}</button>
            <button
                className='btn btn-tertiary'
                onClick={() => dispatch(openCloseOrReopenIssueModal(issue))}
            >{postProps.status}</button>
        </div>
    );

    if (postProps.assignees?.length) {
        assignees = (
            <div style={style.assignee}>
                <b>{'Assignees'}</b>
                <div>
                    {postProps.assignees.map((assignee: string, index: number) => (
                        <span key={assignee}>{(index ? ', ' : '') + assignee} </span>
                    ))}
                </div>
            </div>
        );
    }

    if (postProps.labels?.length) {
        labels = (
            <div>
                <b>{'Labels'}</b>
                <div>
                    {postProps.labels.map((label: string, index: number) => (
                        <span key={label}>{(index ? ', ' : '') + label} </span>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div>
            <h4>
                <a
                    href={postProps.issue_url}
                    target='_blank'
                    rel='noopener noreferrer'
                    style={style.issue_title}
                >
                    {'#' + postProps.issue_number + ' ' + postProps.title}
                </a>
            </h4>
            <p style={style.issue_description}>{postProps.description}</p>
            <div className='d-flex display-flex'>
                {assignees}
                {labels}
            </div>
            {content}
        </div>
    );
};

const getStyle = makeStyleFromTheme((theme) => ({
    button_container: {
        margin: '16px 0 8px 0',
    },
    issue_description: {
        marginBottom: '10px',
    },
    assignee: {
        marginRight: '20px',
    },
    issue_title: {
        fontFamily: 'Metropolis',
        fontWeight: 600,
    },
    assignees_and_labels: {
        display: 'inline-block',
        verticalAlign: 'top',
        width: '30%',
    },
}));

export default GithubIssue;
