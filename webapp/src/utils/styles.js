// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

export const getStyleForReactSelect = (theme) => {
    if (!theme) {
        return null;
    }

    return {
        menuPortal: (provided) => ({
            ...provided,
            zIndex: 9999,
        }),
        control: (provided, state) => ({
            ...provided,
            color: theme.centerChannelColor,
            background: theme.centerChannelBg,

            // Overwrittes the different states of border
            borderColor: state.isFocused ? changeOpacity(theme.centerChannelColor, 0.25) : changeOpacity(theme.centerChannelColor, 0.12),

            // Removes weird border around container
            boxShadow: 'inset 0 1px 1px ' + changeOpacity(theme.centerChannelColor, 0.075),
            borderRadius: '2px',

            '&:hover': {
                borderColor: changeOpacity(theme.centerChannelColor, 0.25),
            },
        }),
        option: (provided, state) => ({
            ...provided,
            background: state.isFocused ? changeOpacity(theme.centerChannelColor, 0.12) : theme.centerChannelBg,
            cursor: state.isDisabled ? 'not-allowed' : 'pointer',
            color: theme.centerChannelColor,
            '&:hover': state.isDisabled ? {} : {
                background: changeOpacity(theme.centerChannelColor, 0.12),
            },
        }),
        clearIndicator: (provided) => ({
            ...provided,
            width: '34px',
            color: changeOpacity(theme.centerChannelColor, 0.4),
            transform: 'scaleX(1.15)',
            marginRight: '-10px',
            '&:hover': {
                color: theme.centerChannelColor,
            },
        }),
        multiValue: (provided) => ({
            ...provided,
            background: changeOpacity(theme.centerChannelColor, 0.15),
        }),
        multiValueLabel: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
            paddingBottom: '4px',
            paddingLeft: '8px',
            fontSize: '90%',
        }),
        multiValueRemove: (provided) => ({
            ...provided,
            transform: 'translateX(-2px) scaleX(1.15)',
            color: changeOpacity(theme.centerChannelColor, 0.4),
            '&:hover': {
                background: 'transparent',
            },
        }),
        menu: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
            background: theme.centerChannelBg,
            border: '1px solid ' + changeOpacity(theme.centerChannelColor, 0.2),
            borderRadius: '0 0 2px 2px',
            boxShadow: changeOpacity(theme.centerChannelColor, 0.2) + ' 1px 3px 12px',
            marginTop: '4px',
        }),
        input: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
        }),
        placeholder: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
        }),
        dropdownIndicator: (provided) => ({
            ...provided,

            '&:hover': {
                color: theme.centerChannelColor,
            },
        }),
        singleValue: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
        }),
        indicatorSeparator: (provided) => ({
            ...provided,
            display: 'none',
        }),
    };
};

export const getLabelFontColor = (hexcolor) => {
    const r = parseInt(hexcolor.substr(0, 2), 16);
    const g = parseInt(hexcolor.substr(2, 2), 16);
    const b = parseInt(hexcolor.substr(4, 2), 16);
    const yiq = ((r * 299) + (g * 587) + (b * 114)) / 1000;
    return (yiq >= 128) ? 'black' : 'white';
};

export const hexToRGB = (hex, alpha) => {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    if (alpha) {
        return 'rgba(' + r + ', ' + g + ', ' + b + ', ' + alpha + ')';
    }
    return 'rgb(' + r + ', ' + g + ', ' + b + ')';
};
