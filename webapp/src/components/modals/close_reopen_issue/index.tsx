// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {Modal} from 'react-bootstrap';

import {Theme} from 'mattermost-redux/types/preferences';

import {useDispatch, useSelector} from 'react-redux';

import {closeCloseOrReOpenIssueModal, closeOrReopenIssue} from '../../../actions';

import {getCloseOrReopenIssueModalData} from '../../../selectors';

import FormButton from '../../form_button';
import Input from '../../input';

const CloseOrReopenIssueModal = ({theme}: {theme: Theme}) => {
    const dispatch = useDispatch();
    const {messageData, visible} = useSelector(getCloseOrReopenIssueModalData);
    const [statusReason, setStatusReason] = useState('completed');
    const [submitting, setSubmitting] = useState(false);
    const [comment, setComment] = useState('');
    if (!visible) {
        return null;
    }

    const handleCloseOrReopenIssue = async (e: React.SyntheticEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const issue = {
            channel_id: messageData.channel_id,
            issue_comment: comment,
            status_reason: messageData?.status === 'open' ? statusReason : 'reopened', // Sending the reason for the issue edit API call
            repo: messageData.repo_name,
            number: messageData.issue_number,
            owner: messageData.repo_owner,
            status: messageData.status === 'open' ? 'Close' : 'Reopen', // Sending the state of the issue which we want it to be after the edit API call
            postId: messageData.postId,
        };
        setSubmitting(true);
        await dispatch(closeOrReopenIssue(issue));
        setSubmitting(false);
        handleClose(e);
    };

    const handleClose = (e: React.SyntheticEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        dispatch(closeCloseOrReOpenIssueModal());
    };

    const handleStatusChange = (e: React.ChangeEvent<HTMLInputElement>) => setStatusReason(e.target.value);

    const handleIssueCommentChange = (updatedComment: string) => setComment(updatedComment);

    const style = getStyle(theme);
    const issueAction = messageData.status === 'open' ? 'Close Issue' : 'Open Issue';
    const modalTitle = issueAction;
    const status = issueAction;
    const savingMessage = messageData.status === 'open' ? 'Closing' : 'Reopening';
    const submitError = null;

    const component = (messageData.status === 'open') ? (
        <div>
            <Input
                label='Leave a comment (optional)'
                type='textarea'
                onChange={handleIssueCommentChange}
                value={comment}
            />
            <div>
                <input
                    type='radio'
                    id='completed'
                    name='close_issue'
                    value='completed'
                defaultChecked // eslint-disable-line
                    onChange={handleStatusChange}
                />
                <label
                    style={style.radioButtons}
                    htmlFor='completed'
                >
                    {'Mark issue as completed'}
                </label>
                <br/>
                <input
                    type='radio'
                    id='not_planned'
                    name='close_issue'
                    value='not_planned'
                    onChange={handleStatusChange}
                />
                <label
                    style={style.radioButtons}
                    htmlFor='not_planned'
                >
                    {'Mark issue as not planned'}
                </label>
            </div>
        </div>
    ) : (
        <div>
            <Input
                label='Leave a comment (optional)'
                type='textarea'
                onChange={handleIssueCommentChange}
                value={comment}
            />
        </div>
    );

    return (
        <Modal
            dialogClassName='modal--scroll'
            show={true}
            onHide={handleClose}
            onExited={handleClose}
            bsSize='large'
            backdrop='static'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>
                    {modalTitle}
                </Modal.Title>
            </Modal.Header>
            <form
                role='form'
                onSubmit={handleCloseOrReopenIssue}
            >
                <Modal.Body
                    style={style.modal}
                >
                    {component}
                </Modal.Body>
                <Modal.Footer>
                    {submitError}
                    <FormButton
                        type='button'
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={handleClose}
                    />
                    <FormButton
                        type='submit'
                        btnClass='btn btn-primary'
                        saving={submitting}
                        defaultMessage={modalTitle}
                        savingMessage={savingMessage}
                    >
                        {status}
                    </FormButton>
                </Modal.Footer>
            </form>
        </Modal>
    );
};

const getStyle = (theme: Theme) => ({
    modal: {
        padding: '2em 2em 3em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
    radioButtons: {
        margin: '0.4em 0.6em',
    },
});

export default CloseOrReopenIssueModal;
