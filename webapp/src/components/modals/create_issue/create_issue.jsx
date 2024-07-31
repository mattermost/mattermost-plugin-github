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
    labels: [],
    assignees: [],
    milestone: null,
    showErrors: false,
    issueTitleValid: true,
};

export default class CreateIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        title: PropTypes.string,
        channelId: PropTypes.string,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
        this.validator = new Validator();
    }

    componentDidUpdate(prevProps) {
        if (this.props.post && !prevProps.post) {
            this.setState({issueDescription: this.props.post.message}); //eslint-disable-line react/no-did-update-set-state
        } else if (this.props.channelId && (this.props.channelId !== prevProps.channelId || this.props.title !== prevProps.title)) {
            const title = this.props.title.substring(0, MAX_TITLE_LENGTH);
            this.setState({issueTitle: title}); // eslint-disable-line react/no-did-update-set-state
        }
    }

    // handle issue creation after form is populated
    handleCreate = async (e) => {
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

        const {post} = this.props;
        const postId = post ? post.id : '';

        const issue = {
            title: this.state.issueTitle,
            body: this.state.issueDescription,
            repo: this.state.repo && this.state.repo.name,
            labels: this.state.labels,
            assignees: this.state.assignees,
            milestone: this.state.milestone && this.state.milestone.value,
            post_id: postId,
            channel_id: this.props.channelId,
        };

        this.setState({submitting: true});

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

    handleIssueDescriptionChange = (issueDescription) =>
        this.setState({issueDescription});

    renderIssueAttributeSelectors = () => {
        if (!this.state.repo || (this.state.repo.permissions && !this.state.repo.permissions.push)) {
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
    };

    render() {
        if (!this.props.visible) {
            return null;
        }

        const theme = this.props.theme;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        const requiredMsg = 'This field is required.';
        let issueTitleValidationError = null;
        if (this.state.showErrors && !this.state.issueTitleValid) {
            issueTitleValidationError = (
                <p
                    className='help-text error-text'
                    style={{marginTop: '8px', marginBottom: '24px'}}
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

        const component = (
            <div>
                <GithubRepoSelector
                    onChange={this.handleRepoChange}
                    value={this.state.repo && this.state.repo.name}
                    required={true}
                    theme={theme}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />

                <Input
                    id={'title'}
                    label='Title for the GitHub Issue'
                    type='input'
                    required={true}
                    disabled={false}
                    maxLength={MAX_TITLE_LENGTH}
                    value={this.state.issueTitle}
                    onChange={this.handleIssueTitleChange}
                />
                {issueTitleValidationError}

                {this.renderIssueAttributeSelectors()}

                <Input
                    label='Description for the GitHub Issue'
                    type='textarea'
                    value={this.state.issueDescription}
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
                    <Modal.Title>{'Create GitHub Issue'}</Modal.Title>
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
