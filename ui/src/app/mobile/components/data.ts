import {Grid} from 'antd';
import * as React from 'react';

export interface AsyncState<T> {
    data?: T;
    loading: boolean;
    error?: Error;
    reload: () => void;
}

export const useAsyncData = <T,>(load: () => Promise<T> & {abort?: () => void}, deps: React.DependencyList): AsyncState<T> => {
    const [data, setData] = React.useState<T>();
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error>();
    const [version, setVersion] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        const req = load();
        setLoading(true);
        setError(undefined);
        req.then(
            next => {
                if (active) {
                    setData(next);
                    setLoading(false);
                }
            },
            err => {
                if (active) {
                    setError(err instanceof Error ? err : new Error(String(err?.message || err)));
                    setLoading(false);
                }
            }
        );
        return () => {
            active = false;
            req.abort?.();
        };
    }, [...deps, version]);

    return {data, loading, error, reload: () => setVersion(current => current + 1)};
};

export const useBreakpoint = () => {
    const screens = Grid.useBreakpoint();
    return {
        isMobile: !screens.md,
        isTablet: screens.md && !screens.lg
    };
};
