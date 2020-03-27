// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export default class Validator {
    constructor() {
        // Our list of components we have to validate before allowing a submit action.
        this.components = new Map();
    }

    addComponent = (key, validateField) => {
        this.components.set(key, validateField);
    };

    removeComponent = (key) => {
        this.components.delete(key);
    };

    validate = () => {
        return Array.from(this.components.values()).reduce((accum, validateField) => {
            return validateField() && accum;
        }, true);
    };
}
