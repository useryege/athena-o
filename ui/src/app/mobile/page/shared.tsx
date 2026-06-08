import {useSearchParams} from 'react-router-dom';
import {StatusTag, useAsyncData} from '../components';
import {services} from '../../shared/services';

export const fmt = (value: unknown) => {
    if (value === undefined || value === null || value === '') {
        return '-';
    }
    if (typeof value === 'boolean') {
        return value ? 'Yes' : 'No';
    }
    return String(value);
};

export const fmtNumber = (value?: number) => (value === undefined ? '-' : new Intl.NumberFormat().format(value));
export const short = (value?: string, head = 10, tail = 8) => (value && value.length > head + tail ? `${value.slice(0, head)}...${value.slice(-tail)}` : value || '-');
export const boolTag = (value?: boolean) => <StatusTag value={fmt(value)} positive={value === true} negative={value === false} />;

export const rbacResources = {
    notifications: 'notifications',
    tokenapi: 'tokenapi',
    wallets: 'wallets'
};

export const rbacActions = {
    update: 'update',
    invoke: 'invoke'
};

export const useCanI = (resource: string, action: string, subresource = '*') =>
    useAsyncData<boolean>(() => services.accounts.canI(resource, action, subresource) as any, [resource, action, subresource]);

export const usePagedParams = (defaultPageSize = 20) => {
    const [params, setParams] = useSearchParams();
    const page = Number(params.get('page') || 1) || 1;
    const pageSize = Number(params.get('pageSize') || params.get('page_size') || defaultPageSize) || defaultPageSize;
    const setPage = (nextPage: number, nextPageSize: number) => {
        const next = new URLSearchParams(params);
        next.set('page', String(nextPage));
        next.set('pageSize', String(nextPageSize));
        setParams(next);
    };
    return {params, setParams, page, pageSize, setPage};
};

export const useKeywordParam = (key = 'q') => {
    const [params, setParams] = useSearchParams();
    const value = params.get(key) || '';
    const setValue = (nextValue: string) => {
        const next = new URLSearchParams(params);
        if (nextValue) {
            next.set(key, nextValue);
        } else {
            next.delete(key);
        }
        next.set('page', '1');
        setParams(next);
    };
    return [value, setValue] as const;
};
