// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {openCreateIssueModalWithoutPost} from '../actions';

const createIssueCommand = '/github issue create';

export default class Hooks {
    constructor(store) {
        this.store = store;
    }

    slashCommandWillBePostedHook = (rawMessage, contextArgs) => {
        let message;
        if (rawMessage) {
            message = rawMessage.trim();
        }

        if (!message) {
            return Promise.resolve({message, args: contextArgs});
        }

        const shouldEnableCreate = true;

        if (message.startsWith(createIssueCommand) && shouldEnableCreate) {
            return this.handleCreateIssueSlashCommand(message, contextArgs);
        }

        return Promise.resolve({message, args: contextArgs});
    }

    handleCreateIssueSlashCommand = (message, contextArgs) => {
        const description = message.slice(createIssueCommand.length).trim();
        this.store.dispatch(openCreateIssueModalWithoutPost(description, contextArgs.channel_id));
        return Promise.resolve({});
    }
}
