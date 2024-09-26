// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import GithubLabelSelector from 'components/github_label_selector';
import GithubAssigneeSelector from 'components/github_assignee_selector';
import GithubMilestoneSelector from 'components/github_milestone_selector';
import GithubRepoSelector from 'components/github_repo_selector';
import Validator from 'components/validator';
import FormButton from 'components/form_button';
import Input from 'components/input';
import {getErrorMessage} from 'utils/user_utils';

const MAX_TITLE_LENGTH = 256;

const initialState = {
    submitting: false,
    error: null,
    repo: null,
    issueTitle: '',
    issueDescription: '',
    channelId: '',
    labels: [],
    assignees: [],
    milestone: null,
    showErrors: false,
    issueTitleValid: true,
};

export default class CreateOrUpdateIssueModal extends PureComponent {
    static propTypes = {
        update: PropTypes.func.isRequired,
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        getIssueInfo: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
        messageData: PropTypes.object,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
        this.validator = new Validator();
    }

    getIssueInfo = async () => {
        const {repo_owner, repo_name, issue_number, postId} = this.props.messageData;
        const issueInfo = await this.props.getIssueInfo(repo_owner, repo_name, issue_number, postId);
        return issueInfo;
    }

    updateState(issueInfo) {
        const {channel_id, title, description, milestone_title, milestone_number, repo_full_name} = issueInfo ?? {};
        const assignees = issueInfo?.assignees ?? [];
        const labels = issueInfo?.labels ?? [];

        this.setState({milestone: {
            value: milestone_number,
            label: milestone_title,
        },
        repo: {
            name: repo_full_name,
        },
        assignees,
        labels,
        channelId: channel_id,
        issueDescription: description,
        issueTitle: title.substring(0, MAX_TITLE_LENGTH)});
    }

    /* eslint-disable react/no-did-update-set-state*/
    componentDidUpdate(prevProps) {
        if (this.props.post && !this.props.messageData && !prevProps.post) {
            this.setState({issueDescription: this.props.post.message});
        }

        if (this.props.messageData?.repo_owner && !prevProps.visible && this.props.visible) {
            this.getIssueInfo().then((issueInfo) => {
                this.updateState(issueInfo.data);
            });
        } else if (this.props.messageData?.channel_id && (this.props.messageData?.channel_id !== prevProps.messageData?.channel_id || this.props.messageData?.title !== prevProps.messageData?.title)) {
            this.updateState(this.props.messageData);
        }
    }
    /* eslint-enable */

    // handle issue creation or updation after form is populated
    handleCreateOrUpdate = async (e) => {
        const {issue_number, postId} = this.props.messageData ?? {};
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const isValidTitle = this.state.issueTitle.trim().length !== 0;
        if (!this.validator.validate() || !isValidTitle) {
            this.setState({
                issueTitleValid: isValidTitle,
                showErrors: true,
            });
            return;
        }

        const issue = {
            title: this.state.issueTitle,
            body: this.state.issueDescription,
            repo: this.state.repo && this.state.repo.name,
            labels: this.state.labels,
            assignees: this.state.assignees,
            milestone: this.state.milestone && this.state.milestone.value,
            post_id: postId,
            channel_id: this.state.channelId,
            issue_number,
        };

        if (!issue.repo) {
            issue.repo = this.props.messageData.repo_owner + this.props.messageData.repo_name;
        }
        this.setState({submitting: true});
        if (issue_number) {
            const updated = await this.props.update(issue);
            if (updated?.error) {
                const errMessage = getErrorMessage(updated.error.message);
                this.setState({
                    error: errMessage,
                    showErrors: true,
                    submitting: false,
                });
                return;
            }
        } else {
            const created = await this.props.create(issue);
            if (created.error) {
                const errMessage = getErrorMessage(created.error.message);
                this.setState({
                    error: errMessage,
                    showErrors: true,
                    submitting: false,
                });
                return;
            }
        }
        this.handleClose(e);
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.setState(initialState, this.props.close);
    };

    handleRepoChange = (repo) => this.setState({repo});

    handleLabelsChange = (labels) => this.setState({labels});

    handleAssigneesChange = (assignees) => this.setState({assignees});

    handleMilestoneChange = (milestone) => this.setState({milestone});

    handleIssueTitleChange = (issueTitle) => this.setState({issueTitle});

    handleIssueDescriptionChange = (issueDescription) => this.setState({issueDescription});

    renderIssueAttributeSelectors = () => {
        if (!this.state.repo || !this.state.repo.name || (this.state.repo.permissions && !this.state.repo.permissions.push)) {
            return null;
        }

        return (
            <>
                <GithubLabelSelector
                    repoName={this.state.repo.name}
                    theme={this.props.theme}
                    selectedLabels={this.state.labels}
                    onChange={this.handleLabelsChange}
                />

                <GithubAssigneeSelector
                    repoName={this.state.repo.name}
                    theme={this.props.theme}
                    selectedAssignees={this.state.assignees}
                    onChange={this.handleAssigneesChange}
                />

                <GithubMilestoneSelector
                    repoName={this.state.repo.name}
                    theme={this.props.theme}
                    selectedMilestone={this.state.milestone}
                    onChange={this.handleMilestoneChange}
                />
            </>
        );
    }

    render() {
        if (!this.props.visible) {
            return null;
        }

        const theme = this.props.theme;
        const {error, submitting, showErrors, issueTitle, issueDescription, repo} = this.state;
        const style = getStyle(theme);
        const {repo_name, repo_owner} = this.props.messageData ?? {};
        const modalTitle = repo_name ? 'Update GitHub Issue' : 'Create GitHub Issue';

        const requiredMsg = 'This field is required.';
        let issueTitleValidationError = null;
        if (showErrors && !issueTitle) {
            issueTitleValidationError = (
                <p
                    className='help-text error-text'
                    style={{
                        marginBottom: '15px',
                    }}
                >
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let submitError = null;
        if (error) {
            submitError = (
                <p className='help-text error-text'>
                    <span>{error}</span>
                </p>
            );
        }

        const component = repo_name ? (
            <div>
                <Input
                    label='Repository'
                    type='input'
                    required={true}
                    disabled={true}
                    value={`${repo_owner}/${repo_name}`}
                />

                <Input
                    id='title'
                    label='Title for the GitHub Issue'
                    type='input'
                    required={true}
                    maxLength={MAX_TITLE_LENGTH}
                    value={issueTitle}
                    onChange={this.handleIssueTitleChange}
                />
                {issueTitleValidationError}

                {this.renderIssueAttributeSelectors()}

                <Input
                    label='Description for the GitHub Issue'
                    type='textarea'
                    value={issueDescription}
                    onChange={this.handleIssueDescriptionChange}
                />
            </div>
        ) : (
            <div>
                <GithubRepoSelector
                    onChange={this.handleRepoChange}
                    value={repo && repo.name}
                    required={true}
                    theme={theme}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />

                <Input
                    id='title'
                    label='Title for the GitHub Issue'
                    type='input'
                    required={true}
                    disabled={false}
                    maxLength={MAX_TITLE_LENGTH}
                    value={issueTitle}
                    onChange={this.handleIssueTitleChange}
                />
                {issueTitleValidationError}

                {this.renderIssueAttributeSelectors()}

                <Input
                    label='Description for the GitHub Issue'
                    type='textarea'
                    value={issueDescription}
                    onChange={this.handleIssueDescriptionChange}
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
                    onSubmit={this.handleCreateOrUpdate}
                >
                    <Modal.Body
                        style={style.modal}
                        ref='modalBody'
                    >
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        {submitError}
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
                            defaultMessage='Submit'
                            savingMessage='Submitting'
                        >
                            {'Submit'}
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
