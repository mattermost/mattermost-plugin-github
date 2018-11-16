import React from 'react';
import PropTypes from 'prop-types';

export default class ToDoItem extends React.PureComponent {
    static propTypes = {
        url: PropTypes.string.isRequired,
        title: PropTypes.string.isRequired,
        repo: PropTypes.string.isRequired,
        type: PropTypes.string.isRequired,
        number: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
    }

    constructor(props) {
        super(props);

        this.state = {
            hover: false,
        };
    }

    handleMouseEnter = () => {
        this.setState({hover: true});
    }

    handleMouseLeave = () => {
        this.setState({hover: false});
    }

    render() {
        const {url, type, number, repo, title} = this.props;
        const displayType = type === 'Issue' ? 'Issue' : 'Pull Request';
        const icon = type === 'Issue' ? 'fa-exclamation-circle' : 'fa-code-fork';

        const {
            container,
            title: titleStyle,
            titleHover,
            subtext,
            subtextHover,
            iconContainer,
        } = getStyle(this.props.theme);

        return (
            <div
                style={container}
                onClick={() => window.open(url, '_blank')}
                onMouseEnter={this.handleMouseEnter}
                onMouseLeave={this.handleMouseLeave}
            >
                <div>
                    <span style={iconContainer}>
                        <i className={'fa ' + icon}/>
                    </span>
                    <span
                        style={this.state.hover ? titleHover : titleStyle}
                    >
                        {title}
                    </span>
                </div>
                <div
                    style={this.state.hover ? subtextHover : subtext}
                >
                    {repo + '#' + number + ' - ' + displayType}
                </div>
            </div>
        );
    }
}

const getStyle = (theme) => {
    const title = {
        fontWeight: 'bold',
        marginLeft: '5px',
        color: theme.linkColor,
    };

    const subtext = {
        fontSize: 'small',
        marginLeft: '18px',
    };

    return {
        container: {
            marginBottom: '5px',
            cursor: 'pointer',
        },
        iconContainer: {
            display: 'inline-block',
            textAlign: 'center',
            width: '12px',
        },
        title,
        titleHover: {
            ...title,
            color: theme.linkColor,
            textDecoration: 'underline',
        },
        subtext,
        subtextHover: {
            ...subtext,
            textDecoration: 'underline',
        },
    };
};
