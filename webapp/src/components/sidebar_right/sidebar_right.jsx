import React from 'react';
import PropTypes from 'prop-types';

import Card from 'react-bootstrap/Card';
import Button from 'react-bootstrap/Button';
import { Badge } from 'react-bootstrap';

//import {FormattedMessage} from 'react-intl';

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        username: PropTypes.string,
        org: PropTypes.string,
        clientId: PropTypes.string,
        enterpriseURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        isTeamSidebar: PropTypes.bool,
        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
            getUnreads: PropTypes.func.isRequired,
            getYourPrs: PropTypes.func.isRequired,
            getYourAssignments: PropTypes.func.isRequired,
        }).isRequired,
    };
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

        //this.setState({refreshing: true});
        await Promise.all([
            this.props.actions.getReviews(),
            this.props.actions.getUnreads(),
            this.props.actions.getYourPrs(),
            this.props.actions.getYourAssignments(),
        ]);
       // this.setState({refreshing: false});
    }

    render() {
        const content = this.props.yourPrs.map((pr) => {
            const labels = pr.labels.map((label) => {
                return (
                    <Badge style={{ ...style.label, ...{backgroundColor: '#' + label.color}}}>{label.name}</Badge>
                );
            });

            return (
                <div
                    key={pr.id}
                >
                    <div>
                        <strong>
                            <a
                                href={pr.html_url}
                                target='_blank'
                                rel='noopener noreferrer'
                            >
                                {pr.title}
                            </a>
                        </strong>
                        {labels}
                    </div>
                    <div className='mb-2 text-muted' style={style.subtitle}>
                        {'Created by ' + pr.user.login + ' at ' + pr.repository_url.replace('https://api.github.com/repos/', '')}
                    </div>
                    <hr style={style.hr}/>
                </div>

            );
        });

        return (
            <React.Fragment>
                <div
                    className='text-center'
                    style={style.divPadding}
                >
                    <strong>{'Your Open Pull Requests'}</strong>
                </div>
                <div
                    className='alert alert-transparent'
                    style={style.container}
                >
                    {content}
                </div>
            </React.Fragment>
        );
    }
}

const style = {
    divPadding: {
        padding: '10px',
    },
    container: {
        margin: '10px',
    },
    label: {
        margin: '5px',
        display: 'initial',
    },
    hr: {
        borderStyle: 'solid',
        borderWidth: '1px 0px',
        borderBottom: '0',
        margin: '.8em 0',
    },
    subtitle: {
        padding: '5px',
        fontSize: '10pt',
    },
};