import {PROJECT_SCOPE, ProjectScope} from '../../../shared/services/athena-application-service';

const QUERY_SCOPE_ACTIVE = 'active';
const QUERY_SCOPE_ARCHIVED = 'archived';
const DEFAULT_PAGE = 1;

const normalizePage = (page: number) => {
    const normalizedPage = Math.floor(page);
    return normalizedPage > 0 ? normalizedPage : DEFAULT_PAGE;
};

export const parseProjectListScopeQuery = (scope: string | null): ProjectScope => {
    if ((scope || '').toLowerCase() === QUERY_SCOPE_ARCHIVED) {
        return PROJECT_SCOPE.ARCHIVED;
    }
    return PROJECT_SCOPE.ACTIVE;
};

export const serializeProjectListScopeQuery = (scope: ProjectScope): string => (scope === PROJECT_SCOPE.ARCHIVED ? QUERY_SCOPE_ARCHIVED : QUERY_SCOPE_ACTIVE);

export const parseProjectsListSearch = (search: string): {page: number; scope: ProjectScope} => {
    const query = new URLSearchParams(search);
    const rawPage = query.get('page');
    const parsedPage = rawPage ? parseInt(rawPage, 10) : NaN;

    return {
        page: Number.isFinite(parsedPage) ? normalizePage(parsedPage) : DEFAULT_PAGE,
        scope: parseProjectListScopeQuery(query.get('scope'))
    };
};

export const buildProjectsListSearch = (page: number, scope: ProjectScope): string => {
    const query = new URLSearchParams();
    query.set('page', String(normalizePage(page)));
    query.set('scope', serializeProjectListScopeQuery(scope));
    return `?${query.toString()}`;
};
