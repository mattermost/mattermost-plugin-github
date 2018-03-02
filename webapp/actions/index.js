import {PostTypes} from 'mattermost-redux/action_types';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import Client from '../client';

export function requestReviewers(postId, prId, reviewers, org, repo) {
    return async (dispatch, getState) => {
        const post = getPost(getState(), postId);

        try {
            await Client.requestReviewers(prId, reviewers, org, repo);
        } catch (error) {
            return {error};
        }

        if (!post) {
            return {data: true};
        }

        const props = {...(post.props || {}), reviewers};

        dispatch({
            type: PostTypes.RECEIVED_POSTS,
            data: {
                order: [],
                posts: {
                    [post.id]: {...post, props}
                }
            },
            channelId: post.channel_id
        });

        return {data: true};
    };
}
