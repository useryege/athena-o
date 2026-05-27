import {isValidURL} from '../utils';

export class InvalidExternalLinkError extends Error {
    constructor(message: string) {
        super(message);
        Object.setPrototypeOf(this, InvalidExternalLinkError.prototype);
        this.name = 'InvalidExternalLinkError';
    }
}

export class ExternalLink {
    public title: string;
    public ref: string;

    constructor(url: string) {
        const parts = url.split('|');
        if (parts.length === 2) {
            this.title = parts[0];
            this.ref = parts[1];
        } else {
            this.title = url;
            this.ref = url;
        }
        if (!isValidURL(this.ref)) {
            throw new InvalidExternalLinkError('Invalid URL');
        }
    }
}
