// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import GithubRepoSelector from 'components/github_repo_selector';
import Input from 'components/input';

const initialState = {
    submitting: false,
    error: null,
    repoValue: '',
    issueTitle: '',
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
    }

    // handle issue creation after form is populated
    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.state.issueTitle) {
            return;
        }

        const issue = {
            title: this.state.issueTitle,
            body: this.props.post.message,
            repo: this.state.repoValue.name,
            post_id: this.props.post.id,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                this.setState({error: created.error.response.body.message, submitting: false});
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

    handleRepoValueChange = (newValue) => {
        this.setState({
            repoValue: newValue,
        });
    };

    handleIssueTitleChange = (newValue) => {
        this.setState({
            issueTitle: newValue,
        });
    };

    render() {
        const {visible, theme} = this.props;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
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
                    id={'repo'}
                    onChange={this.handleRepoValueChange}
                    required={true}
                    theme={theme}
                    value={this.state.repoValue}
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
                <Input
                    label='Description for the GitHub Issue'
                    type='textarea'
                    value={this.props.post.message}
                    disabled={true}
                />
            </div>
        );

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={visible}
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
