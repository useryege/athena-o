export function hashCode(str: string) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        // tslint:disable-next-line:no-bitwise
        hash = ~~((hash << 5) - hash + str.charCodeAt(i));
    }
    return hash;
}

export function isValidURL(url: string): boolean {
    try {
        const parsedUrl = new URL(url);
        return parsedUrl.protocol !== 'javascript:' && parsedUrl.protocol !== 'data:' && parsedUrl.protocol !== 'vbscript:';
    } catch (TypeError) {
        try {
            const parsedUrl = new URL(url, window.location.origin);
            return parsedUrl.protocol !== 'javascript:' && parsedUrl.protocol !== 'data:' && parsedUrl.protocol !== 'vbscript:';
        } catch (TypeError) {
            return false;
        }
    }
}
