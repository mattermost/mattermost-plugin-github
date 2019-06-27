import React from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';

function GithubItems(props) {
    return props.items.length > 0 ? props.items.map((item) => {
        const repoName = item.repository_url.replace(/.+\/repos\//, '');

        return (
            <div key={item.id}>
                <div>
                    <strong>
                        <a
                            href={item.html_url}
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            {item.title ? item.title : item.subject.title}
                        </a>
                    </strong>
                    <GithubLabels labels={item.labels}/>
                </div>
                <div
                    className='mb-3 text-muted'
                    style={style.subtitle}
                >
                    {'Created by ' + item.user.login + ' at ' + repoName}
                </div>
                <hr style={style.hr}/>
            </div>
        );
    }) : 'You have no active items';
}

GithubItems.propTypes = {
    items: PropTypes.array.isRequired,
};

function GithubLabels(props) {
    return props.labels ? props.labels.map((label) => {
        return (
            <Badge
                key={label.id}
                style={{...style.label, ...{backgroundColor: '#' + label.color}}}
            >{label.name}</Badge>
        );
    }) : null;
}

GithubLabels.propTypes = {
    labels: PropTypes.array.isRequired,
};

const style = {
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

export default GithubItems;
