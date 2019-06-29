import React from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import en from 'i18n/en.json';

function GithubItems(props) {
    return props.items.length > 0 ? props.items.map((item) => {
        const style = getStyle(props.theme);

        const repoName = item.repository_url ? item.repository_url.replace(/.+\/repos\//, '') : item.repository.full_name;

        let userName = null;

        if (item.user) {
            userName = item.user.login;
        } else if (item.owner) {
            userName = item.owner.login;
        }

        let title = item.title ? item.title : item.subject.title;

        if (item.html_url) {
            title = (
                <a
                    href={item.html_url}
                    target='_blank'
                    rel='noopener noreferrer'
                    style={style.itemTitle}
                >
                    {item.title ? item.title : item.subject.title}
                </a>);
        }

        return (
            <div
                key={item.id}
                style={style.container}
            >
                <div>
                    <strong>
                        {title}
                    </strong>
                    <GithubLabels labels={item.labels}/>
                </div>
                <div
                    className='light'
                    style={style.subtitle}
                >
                    {userName ? 'Created by ' + userName + ' ' : ''}
                    {'at ' + repoName + '.'}
                    {item.reason ?
                        (<React.Fragment>
                            <br/>
                            {en[item.reason]}
                        </React.Fragment>) : null }
                </div>
            </div>
        );
    }) : 'You have no active items';
}

GithubItems.propTypes = {
    items: PropTypes.array.isRequired,
    theme: PropTypes.object.isRequired,
};

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
        },
        itemTitle: {
            color: theme.centerChannelColor,
            lineHeight: 1.7,
            fontWeight: 'bold',
        },
        subtitle: {
            margin: '5px 0 0 0',
            fontSize: '13px',
        },
    };
});

function GithubLabels(props) {
    return props.labels ? props.labels.map((label) => {
        return (
            <Badge
                key={label.id}
                style={{...itemStyle.label, ...{backgroundColor: '#' + label.color}}}
            >{label.name}</Badge>
        );
    }) : null;
}

GithubLabels.propTypes = {
    labels: PropTypes.array.isRequired,
};

const itemStyle = {
    label: {
        margin: '0 0 0 5px',
        display: 'inline',
        borderRadius: '3px',
        padding: '2px 6px 3px',
        top: '-1px',
        position: 'relative',
    },
};

export default GithubItems;
