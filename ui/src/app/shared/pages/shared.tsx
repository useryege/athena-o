import {useSearchParams} from 'react-router-dom';
import {StatusTag} from '../../components';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../pagination';

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

export const usePagedParams = (defaultPageSize = DEFAULT_PAGE_SIZE, pageSizeOptions = PAGE_SIZE_OPTIONS) => {
    const [params, setParams] = useSearchParams();
    const allowedPageSizes = pageSizeOptions && pageSizeOptions.length > 0 ? pageSizeOptions : undefined;
    const fallbackPageSize = allowedPageSizes && !allowedPageSizes.includes(defaultPageSize) ? allowedPageSizes[0] : defaultPageSize;
    const normalizePageSize = (value: number) => {
        const nextValue = value || fallbackPageSize;
        return allowedPageSizes && !allowedPageSizes.includes(nextValue) ? fallbackPageSize : nextValue;
    };
    const page = Number(params.get('page') || 1) || 1;
    const pageSize = normalizePageSize(Number(params.get('pageSize') || params.get('page_size') || fallbackPageSize));
    const setPage = (nextPage: number, nextPageSize: number) => {
        const next = new URLSearchParams(params);
        next.set('page', String(nextPage));
        next.set('pageSize', String(normalizePageSize(nextPageSize)));
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
