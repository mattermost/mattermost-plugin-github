const React = window.react;
const {Tooltip, OverlayTrigger} = window['react-bootstrap'];

import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

export default class SidebarButtons extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        username: PropTypes.string,
        clientId: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        mentions: PropTypes.arrayOf(PropTypes.object),
        isTeamSidebar: PropTypes.bool,
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
        const isTeamSidebar = this.props.isTeamSidebar;

        let container = style.containerHeader;
        let button = style.buttonHeader;
        let placement = 'bottom';
        if (isTeamSidebar) {
            placement = 'right';
            button = style.buttonTeam;
            container = style.containerTeam;
        }

        if (!this.props.connected) {
            if (isTeamSidebar) {
                return (
                    <OverlayTrigger
                        key='githubConnectLink'
                        placement={placement}
                        overlay={<Tooltip id="reviewTooltip">Connect to your GitHub</Tooltip>}
                    >
                        <a
                            href='/plugins/github/oauth/connect'
                            style={button}
                        >
                            <i className='fa fa-github fa-2x'/>
                        </a>
                    </OverlayTrigger>
                )
            } else {
                return (
                    <div style={container}>
                        <a
                            href='/plugins/github/oauth/connect'
                            style={button}
                        >
                            <i className='fa fa-github fa-lg'/>
                            {' Connect to GitHub'}
                        </a>
                    </div>
                )
            }
        }

        const prs = this.props.reviews || [];
        const mentions = this.props.mentions || [];
        const refreshClass = this.state.refreshing ? ' fa-spin' : '';

        return (
            <div style={container}>
                <a
                    key='githubHeader'
                    href={'https://github.com/settings/connections/applications/' + this.props.clientId}
                    target='_blank'
                    style={button}
                >
                    <i className='fa fa-github fa-lg'/>
                </a>
                <OverlayTrigger
                    key='githubReviewsLink'
                    placement={placement}
                    overlay={<Tooltip id="reviewTooltip">Pull requests needing review</Tooltip>}
                >
                    <a
                        href='https://github.com/pulls/review-requested'
                        target='_blank'
                        style={button}
                    >
                        <i className='fa fa-code-fork'/>
                        {' ' + prs.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubMentionsLink'
                    placement={placement}
                    overlay={<Tooltip id="mentionsTooltip">Issues and pull requests you've been mentioned in</Tooltip>}
                >
                    <a
                        href={'https://github.com/pulls?q=is%3Aopen+mentions%3A' + this.props.username + '+archived%3Afalse'}
                        target='_blank'
                        style={button}
                    >
                        <i className='fa fa-at'/>
                        {' ' + mentions.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubRefreshButton'
                    placement={placement}
                    overlay={<Tooltip id="refreshTooltip">Refresh</Tooltip>}
                >
                    <a
                        href='#'
                        style={button}
                        onClick={this.getData}
                    >
                        <i className={'fa fa-refresh' + refreshClass}/>
                    </a>
                </OverlayTrigger>
            </div>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        buttonTeam: {
            color: changeOpacity(theme.sidebarText, 0.6),
            display: 'block',
            marginBottom: '10px',
            width: '100%',
        },
        buttonHeader: {
            color: changeOpacity(theme.sidebarText, 0.6),
            flex: 1,
            textAlign: 'center',
            cursor: 'pointer',
        },
        containerHeader: {
            marginTop: '10px',
            marginBottom: '5px',
            display: 'flex',
            alignItems: 'center',
        },
        containerTeam: {
        },
    };
});
