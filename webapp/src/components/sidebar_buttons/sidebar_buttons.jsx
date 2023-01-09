// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';
import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {RHSStates} from '../../constants';

export default class SidebarButtons extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        clientId: PropTypes.string,
        enterpriseURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        isTeamSidebar: PropTypes.bool,
        showRHSPlugin: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getConnected: PropTypes.func.isRequired,
            getSidebarContent: PropTypes.func.isRequired,
            updateRhsState: PropTypes.func.isRequired,
        }).isRequired,
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
            return;
        }

        this.props.actions.getConnected(true);
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
        await this.props.actions.getSidebarContent();
        this.setState({refreshing: false});
    }

    openConnectWindow = (e) => {
        e.preventDefault();
        window.open('/plugins/github/oauth/connect', 'Connect Mattermost to GitHub', 'height=570,width=520');
    }

    openRHS = (rhsState) => {
        this.props.actions.updateRhsState(rhsState);
        this.props.showRHSPlugin();
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
                        overlay={<Tooltip id='reviewTooltip'>{'Connect to your GitHub'}</Tooltip>}
                    >
                        <a
                            href='/plugins/github/oauth/connect'
                            onClick={this.openConnectWindow}
                            style={button}
                        >
                            <i className='fa fa-github fa-2x'/>
                        </a>
                    </OverlayTrigger>
                );
            }
            return null;
        }

        const reviews = this.props.reviews || [];
        const yourPrs = this.props.yourPrs || [];
        const unreads = this.props.unreads || [];
        const yourAssignments = this.props.yourAssignments || [];
        const refreshClass = this.state.refreshing ? ' fa-spin' : '';

        let baseURL = 'https://github.com';
        if (this.props.enterpriseURL) {
            baseURL = this.props.enterpriseURL;
        }

        return (
            <div style={container}>
                <a
                    key='githubHeader'
                    href={baseURL + '/settings/connections/applications/' + this.props.clientId}
                    target='_blank'
                    rel='noopener noreferrer'
                    style={button}
                >
                    <i className='fa fa-github fa-lg'/>
                </a>
                <OverlayTrigger
                    key='githubYourPrsLink'
                    placement={placement}
                    overlay={<Tooltip id='yourPrsTooltip'>{'Your open pull requests'}</Tooltip>}
                >
                    <a
                        style={button}
                        onClick={() => this.openRHS(RHSStates.PRS)}
                    >
                        <i className='fa fa-compress'/>
                        {' ' + yourPrs.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubReviewsLink'
                    placement={placement}
                    overlay={<Tooltip id='reviewTooltip'>{'Pull requests needing review'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.REVIEWS)}
                        style={button}
                    >
                        <i className='fa fa-code-fork'/>
                        {' ' + reviews.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubAssignmentsLink'
                    placement={placement}
                    overlay={<Tooltip id='reviewTooltip'>{'Your assignments'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.ASSIGNMENTS)}
                        style={button}
                    >
                        <i className='fa fa-list-ol'/>
                        {' ' + yourAssignments.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubUnreadsLink'
                    placement={placement}
                    overlay={<Tooltip id='unreadsTooltip'>{'Unread messages'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.UNREADS)}
                        style={button}
                    >
                        <i className='fa fa-envelope'/>
                        {' ' + unreads.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='githubRefreshButton'
                    placement={placement}
                    overlay={<Tooltip id='refreshTooltip'>{'Refresh'}</Tooltip>}
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
            textAlign: 'center',
            cursor: 'pointer',
        },
        containerHeader: {
            marginTop: '10px',
            marginBottom: '5px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-around',
            padding: '0 10px',
        },
        containerTeam: {
        },
    };
});
