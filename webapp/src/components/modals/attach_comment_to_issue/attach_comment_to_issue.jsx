// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import Input from 'components/input';

import GithubIssueSelector from 'components/github_issue_selector';
import {getErrorMessage} from 'utils/user_utils';

const initialState = {
    comment: '',
    submitting: false,
    issueValue: null,
    textSearchTerms: '',
    error: null,
};

export default class AttachIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
        messageData: PropTypes.object,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.state.issueValue) {
            const {repo_owner, repo_name, issue_number} = this.props.messageData ?? {};
            const issue = {
                owner: repo_owner,
                repo: repo_name,
                number: issue_number,
                comment: this.state.comment,
                post_id: this.props.post.id,
                show_attached_message: false,
            };
            this.setState({submitting: true});

            this.props.create(issue).then((created) => {
                if (created.error) {
                    const errMessage = getErrorMessage(created.error.message);
                    this.setState({error: errMessage, submitting: false});
                    return;
                }
                this.handleClose(e);
            });
            return;
        }

        const number = this.state.issueValue.number;
        const repoUrl = this.state.issueValue.repository_url;
        const repoUrlParts = repoUrl.split('/');
        const repo = repoUrlParts.pop();
        const owner = repoUrlParts.pop();

        const issue = {
            owner,
            repo,
            number,
            comment: this.state.comment,
            post_id: this.props.post.id,
            show_attached_message: true,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                const errMessage = getErrorMessage(created.error.message);
                this.setState({error: errMessage, submitting: false});
                return;
            }

            this.handleClose(e);
        });
    };

    handleIssueCommentChange = (comment) => this.setState({comment});

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleIssueValueChange = (newValue) => {
        this.setState({
            issueValue: newValue,
        });
    };

    componentDidUpdate(prevProps) {
        if (this.props.post && !this.props.messageData && !prevProps.post) {
            this.setState({comment: this.props.post.message}); // eslint-disable-line react/no-did-update-set-state
        }
    }

    render() {
        const {error, submitting, comment, issueValue} = this.state;
        const {visible, theme, messageData} = this.props;
        const style = getStyle(theme);
        if (!visible) {
            return null;
        }

        const {issue_number} = messageData ?? {};
        const modalTitle = issue_number ? 'Create a comment to GitHub Issue' : 'Attach Message to GitHub Issue';
        const component = issue_number ? (
            <div>
                <Input
                    label='Create a comment'
                    type='textarea'
                    onChange={this.handleIssueCommentChange}
                    value={comment}
                />
            </div>
        ) : (
            <div>
                <GithubIssueSelector
                    id={'issue'}
                    onChange={this.handleIssueValueChange}
                    required={true}
                    theme={theme}
                    error={error}
                    value={issueValue}
                />
                <Input
                    label='Message Attached to GitHub Issue'
                    type='textarea'
                    value={comment}
                    disabled={false}
                    onChange={this.handleIssueCommentChange}
                />
            </div>
        );

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={true}
                onHide={this.handleClose}
                onExited={this.handleClose}
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
                    onSubmit={this.handleCreate}
                >
                    <Modal.Body
                        style={style.modal}
                        ref='modalBody'
                    >
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        <FormButton
                            type='button'
                            btnClass='btn-link'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            type='submit'
                            btnClass='btn btn-primary'
                            saving={submitting}
                            defaultMessage='Attach'
                            savingMessage='Attaching'
                        >
                            {'Attach'}
                        </FormButton>
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme) => ({
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
});
