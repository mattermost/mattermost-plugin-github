const React = window.react;
const {Dropdown} = window['react-bootstrap'];
const {formatText} = window['text-formatting'];
const {messageHtmlToComponent} = window['post-utils'];

import {formatDate} from '../../utils/date_utils';

import PropTypes from 'prop-types';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

export default class PostTypePullRequest extends React.PureComponent {
    static propTypes = {

        /*
         * The post to render the message for.
         */
        post: PropTypes.object.isRequired,

        /**
         * Set to render post body compactly.
         */
        compactDisplay: PropTypes.bool,

        /**
         * Flags if the post_message_view is for the RHS (Reply).
         */
        isRHS: PropTypes.bool,

        /**
         * Set to display times using 24 hours.
         */
        useMilitaryTime: PropTypes.bool,

        /*
         * Logged in user's theme.
         */
        theme: PropTypes.object.isRequired,

        /*
         * Creator's name.
         */
        creatorName: PropTypes.string.isRequired,

        actions: PropTypes.shape({
            requestReviewers: PropTypes.func.isRequired
        }).isRequired

    };

    static defaultProps = {
        mentionKeys: [],
        compactDisplay: false,
        isRHS: false
    };

    constructor(props) {
        super(props);

        this.state = { 
            showDropdown: false,
            reviewers: []
        };
    }

    buildAssignees = (props, style) => {
        if (props.assignees && props.assignees.length) {
             return props.assignees.map((a) => {
                return (
                    <div className='row'>
                        <div
                            style={style.reviewerName}
                        >
                            <img/>{a.name}
                        </div>
                    </div>
                );
            });
        } else {
            return (
                <div className='row'>
                    <div
                        style={style.reviewerName}
                    >
                        {'None'}
                    </div>
                </div>
            );
        }
    }

    onToggle = (showDropdown) => {
        if (!showDropdown) {
            const props = this.props.post.props || {};
            this.props.actions.requestReviewers(this.props.post.id, props.pull_request_number, this.state.reviewers || [], props.org, props.repo);
        }
        this.setState({showDropdown});
    }

    renderReviewerAction = (text) => {
        let check = '';

        if (this.state.reviewers.includes(text)) {
            check += ' Y';
        }

        return (
            <li>
                <a
                    href={'#'}
                    onClick={() => {
                        const reviewers = [...this.state.reviewers];
                        const index = reviewers.indexOf(text);
                        if (index !== -1) {
                            reviewers.splice(index, 1);
                        } else {
                            reviewers.push(text);
                        }
                        this.setState({reviewers});
                    }}
                >
                    {text + check}
                </a>
            </li>
        );
    }

    buildReviewersDropdown = (props, style) => {
        return (
            <div className='row'>
            <strong><a href='#' onClick={() => this.onToggle(!this.state.showDropdown)}>{'Reviewers'}</a></strong>
            <Dropdown
                open={this.state.showDropdown}
                onToggle={this.onToggle}
            >
                <Dropdown.Menu>
                    {this.renderReviewerAction('coreyhulen')}
                    {this.renderReviewerAction('hmhealey')}
                    {this.renderReviewerAction('grundleborg')}
                    {this.renderReviewerAction('saturninoabril')}
                    {this.renderReviewerAction('enahum')}
                    {this.renderReviewerAction('mkraft')}
                </Dropdown.Menu>
            </Dropdown>
            </div>
        );
    }

    buildReviewers = (props, style) => {
        if (props.reviewers && props.reviewers.length) {
             return props.reviewers.map((r) => {
                return (
                    <div className='row'>
                        <div
                            style={style.reviewerName}
                            className='pull-left'
                        >
                            <img/>{r.name}
                        </div>
                        <div
                            style={style.reviewerState}
                            className='pull-right'
                        >
                            {r.state}
                        </div>
                    </div>
                );
            });
        } else {
            return (
                <div>
                    <div
                        style={style.reviewerName}
                        className='row'
                    >
                        {'None'}
                    </div>
                </div>
            );
        }
    }

    buildLabels = (props, style) => {
        if (props.labels && props.labels.length) {
             return props.labels.map((l) => {
                return (
                    <div className='row'>
                        <div
                            style={{color: l.text_color, backgroundColor: l.bg_color, padding: '2px'}}
                        >
                            {l.text}
                        </div>
                    </div>
                );
            });
        } else {
            return (
                <div className='row'>
                    <div
                        style={style.reviewerName}
                    >
                        {'None'}
                    </div>
                </div>
            );
        }
    }

    buildMilestone = (props, style) => {
        return (
            <div>
                <div
                    style={style.reviewerName}
                    className='row'
                >
                    {props.milestone || 'None'}
                </div>
            </div>
        );
    }

    render() {
        const style = getStyle(this.props.theme);
        const post = {...this.props.post};
        //const props = post.props || {};
        post.message = `#### Summary
Integration of team icon upload / get mechanics. Several additions in api, model, i18n and tests. Already existing code of profile image was taken into account.

#### Ticket Link
https://github.com/mattermost/mattermost-server/issues/7616

#### Reference PR
https://github.com/mattermost/mattermost-webapp/pull/796
https://github.com/mattermost/mattermost-redux/pull/403
https://github.com/mattermost/mattermost-api-reference/pull/334`;

        const reviewers = this.state.reviewers.map((r) => ({name: r, state: 'R'}));

        const props = {
            number: 123,
            submitter_name: 'jwilander',
            title: 'PLT-7567 - Integration of team icons',
            reviewers: [
                {name: 'cpanato', state: 'A'},
                {name: 'crspeller', state: 'X'},
                ...reviewers,
            ],
            assignees: [
                {name: 'hmhealey'},
            ],
            labels: [
                {text: '2: Dev Review', text_color: 'black', bg_color: 'orange'},
            ],
            submitted_at: '3 hours ago'
        };

        const formattedText = formatText(post.message || '');

        return (
            <div>
            <div
                style={style.content}
                className='col-sm-8'
                >
                    <h2>{props.title + ' #' + props.number}</h2>
                    <span>{props.submitter_name + ' submitted ' + props.submitted_at}</span>
                    {messageHtmlToComponent(formattedText, false)}
                </div>
                <div
                    style={style.contentRight}
                    className='col-sm-2'
                >
                    <div style={style.rightSection}>
                        {this.buildReviewersDropdown(props, style)}
                        {this.buildReviewers(props, style)}
                    </div>
                    <div style={style.rightSection}>
                        <strong className='row'>{'Assignees'}</strong>
                        {this.buildAssignees(props, style)}
                    </div>
                    <div style={style.rightSection}>
                        <strong className='row'>{'Labels'}</strong>
                        {this.buildLabels(props, style)}
                    </div>
                    <div style={style.rightSection}>
                        <strong className='row'>{'Milestone'}</strong>
                        {this.buildMilestone(props, style)}
                    </div>
                </div>
            </div>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        attachment: {
            marginLeft: '-20px',
            position: 'relative'
        },
        content: {
            borderRight: '1px',
            borderColor: '#BDBDBF',
        },
        contentRight: {
        },
        rightSection: {
        },
        reviewerName: {
            width: '90%'
        },
        reviewerState: {
            width: '10%'
        },
        container: {
            borderLeftStyle: 'solid',
            borderLeftWidth: '4px',
            padding: '10px',
            borderLeftColor: '#89AECB'
        },
        body: {
            overflowX: 'auto',
            overflowY: 'hidden',
            paddingRight: '5px',
            width: '100%'
        },
        title: {
            fontSize: '16px',
            fontWeight: '600',
            height: '22px',
            lineHeight: '18px',
            margin: '5px 0 1px 0',
            padding: '0'
        },
        button: {
            fontFamily: 'Open Sans',
            fontSize: '12px',
            fontWeight: 'bold',
            letterSpacing: '1px',
            lineHeight: '19px',
            marginTop: '12px',
            borderRadius: '4px',
            color: theme.buttonColor
        },
        buttonIcon: {
            paddingRight: '8px',
            fill: theme.buttonColor
        },
        summary: {
            fontFamily: 'Open Sans',
            fontSize: '14px',
            fontWeight: '600',
            lineHeight: '26px',
            margin: '0',
            padding: '14px 0 0 0'
        },
        summaryItem: {
            fontFamily: 'Open Sans',
            fontSize: '14px',
            lineHeight: '26px'
        }
    };
});
