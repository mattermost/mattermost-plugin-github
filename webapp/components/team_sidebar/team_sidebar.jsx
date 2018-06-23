const React = window.react;
const {Tooltip, OverlayTrigger} = window['react-bootstrap'];

import PropTypes from 'prop-types';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

export default class TeamSidebar extends React.PureComponent {
    static propTypes = {
        /*
         * Logged in user's theme.
         */
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        username: PropTypes.string,
        clientId: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        mentions: PropTypes.arrayOf(PropTypes.object),

        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
            getMentions: PropTypes.func.isRequired,
        }).isRequired

    };

    constructor(props) {
        super(props);

        this.state = {
            refreshing: false,
        };
    }

    componentDidMount() {
        if (this.props.connected) {
            this.getData();
        }
    }

    componentDidUpdate(prevProps) {
        if (this.props.connected && !prevProps.connected) {
            this.getData();
        }
    }

    getData = async (e) => {
        if (this.state.refreshing) {
            return;
        }

        if (e) {
            e.preventDefault();
        }

        this.setState({refreshing: true});
        await Promise.all([
            this.props.actions.getReviews(),
            this.props.actions.getMentions(),
        ]);
        this.setState({refreshing: false});
    }

    render() {
        const style = getStyle(this.props.theme);

        if (!this.props.connected) {
            return (
                <OverlayTrigger
                    key='githubConnectLink'
                    placement='right'
                    overlay={<Tooltip id="reviewTooltip">Connect your Mattermost to GitHub</Tooltip>}
                >
                    <a
                        href='/plugins/github/oauth/connect'
                        style={style.container}
                    >
                        <i className='fa fa-github fa-2x'/>
                    </a>
                </OverlayTrigger>
            )
        }

        const prs = this.props.reviews || [];
        const mentions = this.props.mentions || [];
        const refreshClass = this.state.refreshing ? ' fa-spin' : '';

        return (
            <React.Fragment>
                <a
                    key='githubHeader'
                    href={'https://github.com/settings/connections/applications/' + this.props.clientId}
                    target='_blank'
                    style={style.container}
                >
                    <i className='fa fa-github fa-lg'/>
                </a>
                <OverlayTrigger
                    key='githubReviewsLink'
                    placement='right'
                    overlay={<Tooltip id="reviewTooltip">Pull requests needing review</Tooltip>}
                >
                    <a
                        href='https://github.com/pulls/review-requested'
                        target='_blank'
                        style={style.container}
                    >
                        <i className='fa fa-code-fork'/>
                        {' ' + prs.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubMentionsLink'
                    placement='right'
                    overlay={<Tooltip id="mentionsTooltip">Issues and pull requests you've been mentioned in</Tooltip>}
                >
                    <a
                        href={'https://github.com/pulls?q=is%3Aopen+mentions%3A' + this.props.username + '+archived%3Afalse'}
                        target='_blank'
                        style={style.container}
                    >
                        <i className='fa fa-at'/>
                        {' ' + mentions.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubRefreshButton'
                    placement='right'
                    overlay={<Tooltip id="refreshTooltip">Refresh</Tooltip>}
                >
                    <a
                        href='#'
                        style={style.container}
                        onClick={this.getData}
                    >
                        <i className={'fa fa-refresh' + refreshClass}/>
                    </a>
                </OverlayTrigger>
            </React.Fragment>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            color: theme.sidebarText,
            display: 'block',
            marginBottom: '10px',
            width: '100%',
        },
    };
});
