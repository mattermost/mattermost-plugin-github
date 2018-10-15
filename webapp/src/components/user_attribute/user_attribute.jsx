import React from 'react';
import PropTypes from 'prop-types';

export default class UserAttribute extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        username: PropTypes.string,
    };

    render() {
        const style = getStyle(this.props.theme);

        const username = this.props.username;

        if (!username) {
            return null;
        }

        return (
            <div style={style.container}>
                <a
                    href={'https://github.com/' + username}
                    target='_blank'
                    rel='noopener noreferrer'
                >
                    <i className='fa fa-github'/>{' ' + username}
                </a>
            </div>
        );
    }
}

const getStyle = {
    container: {
        margin: '5px 0',
    },
};
