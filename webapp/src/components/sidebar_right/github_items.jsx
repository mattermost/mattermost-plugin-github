import React from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';
import {FormattedMessage} from 'react-intl';

import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

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
            <div key={item.id}>
                <div>
                    <strong>
                        {title}
                    </strong>
                    <GithubLabels labels={item.labels}/>
                </div>
                <div
                    className='mb-3 text-muted'
                    style={style.subtitle}
                >
                    {userName ? 'Created by ' + userName + ' ' : ''}
                    {'at ' + repoName + '.'}
                    {item.reason ?
                        (<React.Fragment>
                            <br/>
                            <FormattedMessage
                                id={item.reason}
                                defaultMessage={item.reason}
                            />
                        </React.Fragment>) : null }
                </div>
                <hr style={style.hr}/>
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
        itemTitle: {
            color: theme.centerChannelColor,
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
        margin: '5px',
        display: 'initial',
    },
};

export default GithubItems;
