import React from 'react';
import PropTypes from 'prop-types';

import ToDoItem from './todo_item';

export default class ToDoPostType extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
    }

    buildNotification = (n) => {
        return (
            <ToDoItem
                key={'githubnotification' + n.repo + n.number}
                url={n.url}
                title={n.title}
                repo={n.repo}
                number={n.number}
                type={n.type}
                theme={this.props.theme}
            />
        );
    }

    buildNotifications = (notifications) => {
        return (
            <div>{notifications.map(this.buildNotification)}</div>
        );
    }

    openAll = (notifications) => {
        notifications.forEach((n) => {
            window.open(n.url, '_blank');
        });
    }

    render() {
        const post = {...this.props.post};
        const {props} = post;
        const {button} = getStyle();

        const notifications = JSON.parse(props.notifications);

        return (
            <div>
                <h5>
                    <strong>{'Unread Messages (' + notifications.length + ')'}</strong>
                    <button
                        className='btn btn-xs btn-primary'
                        style={button}
                        onClick={() => this.openAll(notifications)}
                    >
                        {'Open All'}
                    </button>
                </h5>
                {this.buildNotifications(notifications)}
            </div>
        );
    }
}

const getStyle = () => ({
    button: {
        marginLeft: '10px',
    },
});
