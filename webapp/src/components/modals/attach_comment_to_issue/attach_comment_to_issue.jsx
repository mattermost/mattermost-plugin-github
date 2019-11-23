// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import Input from 'components/input';

import GithubIssueSelector from 'components/github_issue_selector';

const initialState = {
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
        currentTeam: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
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
            comment: this.props.post.message,
            post_id: this.props.post.id,
            current_team: this.props.currentTeam.name,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                this.setState({error: created.error.message, submitting: false});
                return;
            }
            
            this.handleClose(e);
        });
    };

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

    render() {
        const {visible, theme} = this.props;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
        }

        const component = (
            <div>
                <GithubIssueSelector
                    id={'issue'}
                    onChange={this.handleIssueValueChange}
                    required={true}
                    theme={theme}
                    error={error}
                    value={this.state.issueValue}
                />
                <Input
                    label='Message Attached to Github Issue'
                    type='textarea'
                    isDisabled={true}
                    value={this.props.post.message}
                    disabled={false}
                    readOnly={true}
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
                        {'Attach Message to Github Issue'}
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
