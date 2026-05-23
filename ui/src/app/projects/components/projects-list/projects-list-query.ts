const DEFAULT_PAGE = 1;

const normalizePage = (page: number) => {
    const normalizedPage = Math.floor(page);
    return normalizedPage > 0 ? normalizedPage : DEFAULT_PAGE;
};

export const parseProjectsListSearch = (search: string): {page: number} => {
    const query = new URLSearchParams(search);
    const rawPage = query.get('page');
    const parsedPage = rawPage ? parseInt(rawPage, 10) : NaN;

    return {
        page: Number.isFinite(parsedPage) ? normalizePage(parsedPage) : DEFAULT_PAGE
    };
};

export const buildProjectsListSearch = (page: number): string => {
    const query = new URLSearchParams();
    query.set('page', String(normalizePage(page)));
    return `?${query.toString()}`;
};
