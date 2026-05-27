import React from 'react';

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

export const colorSchemes = {
    light: '(prefers-color-scheme: light)',
    dark: '(prefers-color-scheme: dark)'
};

export function getTheme(theme: string) {
    if (theme !== 'auto') {
        return theme;
    }

    const dark = window.matchMedia(colorSchemes.dark);

    return dark.matches ? 'dark' : 'light';
}

export const useSystemTheme = (cb: (theme: string) => void) => {
    const dark = window.matchMedia(colorSchemes.dark);
    const light = window.matchMedia(colorSchemes.light);

    const listener = () => {
        cb(dark.matches ? 'dark' : 'light');
    };

    dark.addEventListener('change', listener);
    light.addEventListener('change', listener);

    return () => {
        dark.removeEventListener('change', listener);
        light.removeEventListener('change', listener);
    };
};

export const useTheme = (props: {theme: string}) => {
    const [theme, setTheme] = React.useState(getTheme(props.theme));

    React.useEffect(() => {
        let destroyListener: (() => void) | undefined;

        if (props.theme === 'auto') {
            destroyListener = useSystemTheme(systemTheme => {
                setTheme(systemTheme);
            });
        }

        if (props.theme !== theme) {
            setTheme(getTheme(props.theme));
        }

        return () => {
            destroyListener?.();
        };
    }, [props.theme]);

    return [theme];
};
