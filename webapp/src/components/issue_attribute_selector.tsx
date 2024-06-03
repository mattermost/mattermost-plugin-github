// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect from 'react-select';

import {getStyleForReactSelect} from '@/utils/styles';
import Setting from '@/components/setting';

import {ReactSelectOption} from './react_select_setting';
import {Theme} from 'mattermost-redux/types/preferences';

export type Props = {
    isMulti: boolean;
    repoName: string;
    theme: Theme;
    onChange: (selection: ReactSelectOption | ReactSelectOption[] | null) => void;
    loadOptions: () => Promise<ReactSelectOption[]>;
    selection: ReactSelectOption | string[] | null;
};

type State = {
    options: ReactSelectOption[];
    isLoading: boolean;
    error: Error | null;
};

export default class IssueAttributeSelector extends PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);

        this.state = {
            options: [],
            isLoading: false,
            error: null,
        };
    }

    componentDidMount() {
        if (this.props.repoName) {
            this.loadOptions();
        }
    }

    componentDidUpdate(prevProps: Props) {
        if (this.props.repoName && prevProps.repoName !== this.props.repoName) {
            this.loadOptions();
        }
    }

    loadOptions = async () => {
        this.setState({isLoading: true});

        try {
            const options = await this.props.loadOptions();
            this.filterSelection(options);
            this.setState({
                options,
                isLoading: false,
                error: null,
            });
        } catch (err) {
            this.filterSelection([]);
            this.setState({
                error: err,
                isLoading: false,
            });
        }
    };

    filterSelection = (options: ReactSelectOption[]) => {
        if (this.props.isMulti || Array.isArray(this.props.selection)) {
            const selection = this.props.selection as string[] | null;
            const filtered = options.filter((option) => selection?.includes(option.value));
            this.props.onChange(filtered);
            return;
        }

        if (!this.props.selection) {
            this.props.onChange(null);
            return;
        }

        for (const option of options) {
            if (option.value === this.props.selection.value) {
                this.props.onChange(option);
                return;
            }
        }

        this.props.onChange(null);
    }

    onChange = (selection: ReactSelectOption | ReactSelectOption[] | null) => {
        if (this.props.isMulti) {
            this.props.onChange(selection || []);
            return;
        }

        this.props.onChange(selection);
    };

    render() {
        let selection: ReactSelectOption | ReactSelectOption[] | null;
        if (Array.isArray(this.props.selection)) {
            selection = this.props.selection.map((s) => ({label: s, value: s}));
        } else {
            selection = this.props.selection;
        }

        const noOptionsMessage = this.props.repoName ? 'No options' : 'Please select a repository first';

        const {theme, ...props} = this.props;

        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...props}
                    isClearable={true}
                    placeholder={'Select...'}
                    noOptionsMessage={() => noOptionsMessage}
                    closeMenuOnSelect={!this.props.isMulti}
                    hideSelectedOptions={this.props.isMulti}
                    onChange={this.onChange as any}
                    options={this.state.options}
                    value={selection}
                    isLoading={this.state.isLoading}
                    styles={getStyleForReactSelect(theme) as any}
                />

                {this.state.error && (
                    <p className='alert alert-danger'>
                        <i
                            className='fa fa-warning'
                            title='Warning Icon'
                        />
                        <span> {this.state.error.message}</span>
                    </p>
                )}
            </Setting>
        );
    }
}
