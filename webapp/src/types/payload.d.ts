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
