const React = window.react;
import PropTypes from 'prop-types';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

export default class PostTypeSettings extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
        actions: PropTypes.shape({
            updateSettings: PropTypes.func.isRequired
        }).isRequired
    };

    static defaultProps = {
        mentionKeys: [],
        compactDisplay: false,
        isRHS: false
    };

    constructor(props) {
        super(props);

        this.state = {};
    }

    render() {
        const style = getStyle(this.props.theme);
        const post = this.props.post;
        

        return (
            <div>
                Settings
            </div>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
        },
    };
});
