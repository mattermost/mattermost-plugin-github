// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';
import {Overlay} from 'react-bootstrap';
import {ChevronDownIcon, ChevronUpIcon, CheckIcon} from '@primer/octicons-react';

import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

export default class SortDropdown extends React.PureComponent {
    static propTypes = {
        value: PropTypes.shape({
            value: PropTypes.string.isRequired,
            label: PropTypes.string.isRequired,
        }).isRequired,
        onChange: PropTypes.func.isRequired,
        options: PropTypes.arrayOf(PropTypes.shape({
            value: PropTypes.string.isRequired,
            label: PropTypes.string.isRequired,
        })).isRequired,
        theme: PropTypes.object.isRequired,
        ariaLabel: PropTypes.string,
        className: PropTypes.string,
    };

    static defaultProps = {
        ariaLabel: 'Sort items',
        className: '',
    };

    constructor(props) {
        super(props);
        this.state = {isOpen: false, hoveredOption: null, isTriggerFocused: false};
        this.toggleRef = React.createRef();
    }

    handleTriggerFocus = () => {
        this.setState({isTriggerFocused: true});
    };

    handleTriggerBlur = () => {
        this.setState({isTriggerFocused: false});
    };

    handleToggle = () => {
        this.setState(({isOpen}) => ({isOpen: !isOpen}));
    };

    handleClose = () => {
        this.setState({isOpen: false, hoveredOption: null});
    };

    handleOptionClick = (option) => {
        this.props.onChange(option);
        this.handleClose();
    };

    handleKeyDown = (event) => {
        const {options, value} = this.props;
        const currentIndex = options.findIndex((opt) => opt.value === value.value);
        let newIndex = currentIndex;

        switch (event.key) {
        case 'ArrowDown':
            event.preventDefault();
            newIndex = (currentIndex + 1) % options.length;
            break;
        case 'ArrowUp':
            event.preventDefault();
            newIndex = ((currentIndex - 1) + options.length) % options.length;
            break;
        case 'Enter':
        case ' ':
            event.preventDefault();
            this.handleToggle();
            return;
        case 'Escape':
            this.handleClose();
            return;
        default:
            return;
        }

        if (newIndex !== currentIndex) {
            this.props.onChange(options[newIndex]);
        }
    };

    render() {
        const {value, options, theme, ariaLabel, className} = this.props;
        const {isOpen, hoveredOption, isTriggerFocused} = this.state;
        const style = getStyle(theme);

        const trigger = (
            <button
                ref={this.toggleRef}
                type='button'
                style={{
                    ...style.trigger,
                    ...(isTriggerFocused ? style.triggerFocused : {}),
                }}
                onClick={this.handleToggle}
                onKeyDown={this.handleKeyDown}
                onFocus={this.handleTriggerFocus}
                onBlur={this.handleTriggerBlur}
                aria-haspopup='listbox'
                aria-expanded={isOpen}
                aria-label={`Sort by: ${value.label}`}
                className={className}
            >
                <span style={style.triggerText}>
                    {'Sort by: '}
                    {value.label}
                </span>
                <span style={style.chevronContainer}>
                    {isOpen ? (
                        <ChevronUpIcon
                            size={12}
                            style={style.chevron}
                            aria-hidden='true'
                        />
                    ) : (
                        <ChevronDownIcon
                            size={12}
                            style={style.chevron}
                            aria-hidden='true'
                        />
                    )}
                </span>
            </button>
        );

        const overlay = (
            <Overlay
                show={isOpen}
                onHide={this.handleClose}
                target={this.toggleRef}
                placement='bottom-start'
                rootClose={true}
            >
                <div style={style.popover}>
                    <ul
                        style={style.optionsList}
                        role='listbox'
                        aria-label={ariaLabel}
                    >
                        {options.map((option) => {
                            const isSelected = option.value === value.value;
                            const isHovered = option.value === hoveredOption;
                            return (
                                <li
                                    key={option.value}
                                    role='option'
                                    aria-selected={isSelected}
                                    style={{
                                        ...style.option,
                                        ...(isSelected ? style.optionSelected : {}),
                                        ...(isHovered && !isSelected ? style.optionHover : {}),
                                    }}
                                    onClick={() => this.handleOptionClick(option)}
                                    onMouseEnter={() => this.setState({hoveredOption: option.value})}
                                    onMouseLeave={() => this.setState({hoveredOption: null})}
                                    onFocus={() => this.setState({hoveredOption: option.value})}
                                    onBlur={() => this.setState({hoveredOption: null})}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter' || e.key === ' ') {
                                            e.preventDefault();
                                            this.handleOptionClick(option);
                                        }
                                    }}
                                    tabIndex={isSelected ? 0 : -1}
                                >
                                    <span style={style.optionLabel}>
                                        <CheckIcon
                                            size={16}
                                            style={isSelected ? style.optionIconSelected : style.optionIcon}
                                            aria-hidden='true'
                                        />
                                        <span style={isSelected ? style.optionTextSelected : style.optionText}>
                                            {option.label}
                                        </span>
                                    </span>
                                </li>
                            );
                        })}
                    </ul>
                </div>
            </Overlay>
        );

        return (
            <div style={style.container}>
                {trigger}
                {overlay}
            </div>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    const textColor = theme.centerChannelColor;
    const bgColor = theme.centerChannelBg;

    const text56 = changeOpacity(textColor, 0.56);
    const text8 = changeOpacity(textColor, 0.08);

    return {
        container: {
            display: 'inline-flex',
            alignItems: 'center',
        },

        // Select trigger — "Sort by: Recently Created"
        // font: Open Sans 12px/16px semibold, color rgba(61,60,64,0.56)
        trigger: {
            display: 'inline-flex',
            alignItems: 'center',
            gap: '4px',
            padding: 0,
            border: 'none',
            borderRadius: 0,
            background: 'transparent',
            color: text56,
            fontFamily: '"Open Sans", sans-serif',
            fontSize: '12px',
            fontStyle: 'normal',
            fontWeight: 600,
            lineHeight: '16px',
            textAlign: 'center',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
        },
        triggerFocused: {
            outline: `2px solid ${textColor}`,
            outlineOffset: '2px',
        },
        triggerText: {
            display: 'inline',
        },
        chevronContainer: {
            display: 'inline-flex',
            alignItems: 'center',
            flexShrink: 0,
        },
        chevron: {
            color: text56,
        },

        // Popover — display:inline-flex, padding:8px 0, flex-direction:column, align-items:flex-start
        // border-radius:4px, border:1px solid rgba(61,60,64,0.08), background:#FFF
        // box-shadow: Elevation 4 = 0 8px 24px 0 rgba(0,0,0,0.12)
        popover: {
            position: 'relative',
            display: 'inline-flex',
            padding: '8px 0',
            flexDirection: 'column',
            alignItems: 'flex-start',
            background: bgColor,
            border: `1px solid ${text8}`,
            borderRadius: '4px',
            boxShadow: '0 8px 24px 0 rgba(0, 0, 0, 0.12)',
            minWidth: '200px',
            zIndex: 9999,
            marginTop: '4px',
        },
        optionsList: {
            listStyle: 'none',
            margin: 0,
            padding: 0,
            maxHeight: '300px',
            overflowY: 'auto',
        },
        option: {
            height: '32px',
            display: 'flex',
            alignItems: 'center',
            padding: '0 20px',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            transition: 'background 0.1s ease',
        },
        optionHover: {
            background: text8,
            outline: 'none',
        },
        optionSelected: {
            background: textColor,
        },
        optionLabel: {
            display: 'inline-flex',
            alignItems: 'center',
            gap: '12px',
            padding: '0 4px',
        },
        optionIcon: {
            color: 'transparent',
            flexShrink: 0,
        },
        optionIconSelected: {
            color: bgColor,
            flexShrink: 0,
        },
        optionText: {
            fontSize: '12px',
            fontWeight: 400,
            color: textColor,
        },
        optionTextSelected: {
            fontSize: '12px',
            fontWeight: 400,
            color: bgColor,
        },
    };
});
