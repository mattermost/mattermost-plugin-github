// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

type AttachCommentToIssuePayload = {
    post_id: string;
    owner: string;
    repo: string;
    number: number;
    comment: string;
}

type CreateIssuePayload = {
    title: string;
    body: string;
    repo: string;
    post_id: string;
    channel_id: string;
    labels: string[];
    assignees: string[];
    milestone: number;
}
