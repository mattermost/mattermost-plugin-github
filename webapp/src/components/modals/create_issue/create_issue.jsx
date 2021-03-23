// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import GithubRepoSelector from 'components/github_repo_selector';
import Validator from 'components/validator';
import Input from 'components/input';
import {getErrorMessage} from 'utils/user_utils';

const initialState = {
    submitting: false,
    error: null,
    repoValue: '',
    issueTitle: '',
    issueDescription: '',
    showErrors: false,
    issueTitleValid: true,
};

export default class CreateIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
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
        }
    }

    // handle issue creation after form is populated
    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate() || !this.state.issueTitle) {
            this.setState({
                issueTitleValid: Boolean(this.state.issueTitle),
                showErrors: true,
            });
            return;
        }

        const issue = {
            title: this.state.issueTitle,
            body: this.state.issueDescription,
            repo: this.state.repoValue,
            post_id: this.props.post.id,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                const errMessage = getErrorMessage(created.error.message);

                this.setState({
                    error: errMessage,
                    showErrors: true,
                    submitting: false});
                return;
            }
            this.handleClose(e);
        });
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.setState(initialState, this.props.close);
    };

    handleRepoValueChange = (name) => {
        this.setState({
            repoValue: name,
        });
    };

    handleIssueTitleChange = (newValue) => {
        this.setState({
            issueTitle: newValue,
        });
    };

    handleIssueDescriptionChange = (newValue) => {
        this.setState({
            issueDescription: newValue,
        });
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
                <p className='help-text error-text'>
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
                    onChange={this.handleRepoValueChange}
                    value={this.state.repoValue}
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
                    maxLength={65}
                    value={this.state.issueTitle}
                    onChange={this.handleIssueTitleChange}
                />
                {issueTitleValidationError}
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
                    <Modal.Title>
                        {'Create GitHub Issue'}
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
