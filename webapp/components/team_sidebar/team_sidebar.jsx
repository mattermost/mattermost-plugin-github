const React = window.react;

import PropTypes from 'prop-types';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

export default class TeamSidebar extends React.PureComponent {
    static propTypes = {
        /*
         * Logged in user's theme.
         */
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        pullRequests: PropTypes.arrayOf(PropTypes.object),

        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
        }).isRequired

    };

    componentDidMount() {
        if (this.props.connected) {
            this.props.actions.getReviews();
        }
    }

    componentDidUpdate(prevProps) {
        if (this.props.connected && !prevProps.connected) {
            this.props.actions.getReviews();
        }
    }

    render() {
        const style = getStyle(this.props.theme);

        if (!this.props.connected) {
            return (
                <a
                    href='/plugins/github/oauth/connect'
                    style={style.container}
                    key='githubConnectLink'
                >
                    <i className='fa fa-github'/>
                </a>
            )
        }

        const prs = this.props.pullRequests || [];

        return (
            <React.Fragment>
                <a
                    href='https://github.com/pulls/review-requested'
                    target='_blank'
                    style={style.container}
                    key='githubReviewsLink'
                >
                    <i className='fa fa-code-fork'/>
                    {' ' + prs.length}
                </a>
                <a
                    href='https://github.com/pulls/mentioned'
                    target='_blank'
                    style={style.container}
                    key='githubMentionsLink'
                >
                    <i className='fa fa-at'/>
                    {' 6'}
                </a>
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
