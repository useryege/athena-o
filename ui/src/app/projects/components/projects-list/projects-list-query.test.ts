import {PROJECT_SCOPE} from '../../../shared/services/athena-application-service';
import {buildProjectsListSearch, parseProjectsListSearch} from './projects-list-query';

test('parseProjectsListSearch parses valid page and scope', () => {
    expect(parseProjectsListSearch('?page=6&scope=archived')).toEqual({
        page: 6,
        scope: PROJECT_SCOPE.ARCHIVED
    });
});

test('parseProjectsListSearch falls back to defaults for missing params', () => {
    expect(parseProjectsListSearch('')).toEqual({
        page: 1,
        scope: PROJECT_SCOPE.ACTIVE
    });
});

test('parseProjectsListSearch falls back to defaults for invalid params', () => {
    expect(parseProjectsListSearch('?page=abc&scope=unknown')).toEqual({
        page: 1,
        scope: PROJECT_SCOPE.ACTIVE
    });
});

test('buildProjectsListSearch serializes query with canonical values', () => {
    expect(buildProjectsListSearch(5, PROJECT_SCOPE.ARCHIVED)).toBe('?page=5&scope=archived');
});
