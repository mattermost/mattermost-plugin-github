import React from 'react';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars';

import {RHSStates} from '../../constants';

import GithubItems from './github_items';

export function renderView(props) {
    return (
        <div
            {...props}
            className='scrollbar--view'
        />);
}

export function renderThumbHorizontal(props) {
    return (
        <div
            {...props}
            className='scrollbar--horizontal'
        />);
}

export function renderThumbVertical(props) {
    return (
        <div
            {...props}
            className='scrollbar--vertical'
        />);
}

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        rhsState: PropTypes.string,
        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
            getUnreads: PropTypes.func.isRequired,
            getYourPrs: PropTypes.func.isRequired,
            getYourAssignments: PropTypes.func.isRequired,
        }).isRequired,
    };

    render() {
        let title = '';
        let githubItems = [];

        switch (this.props.rhsState) {
        case RHSStates.PRS:

            githubItems = this.props.yourPrs;
            title = 'Your Open Pull Requests';
            break;
        case RHSStates.REVIEWS:

            githubItems = this.props.reviews;
            title = 'Pull Requests Needing Review';
            break;
        case RHSStates.UNREADS:

            githubItems = this.props.unreads;
            title = 'Unread Messages';
            break;
        case RHSStates.ASSIGNMENTS:

            githubItems = this.props.yourAssignments;
            title = 'Your Assignments';
            break;
        default:
            break;
        }

        return (
            <React.Fragment>
                <Scrollbars
                    autoHide={true}
                    autoHideTimeout={500}
                    autoHideDuration={500}
                    renderThumbHorizontal={renderThumbHorizontal}
                    renderThumbVertical={renderThumbVertical}
                    renderView={renderView}
                >
                    <div
                        className='text-center'
                        style={style.divPadding}
                    >
                        <strong>{title}</strong>
                    </div>
                    <div
                        className='alert alert-transparent'
                        style={style.container}
                    >
                        <GithubItems items={githubItems}/>
                    </div>
                </Scrollbars>
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
};