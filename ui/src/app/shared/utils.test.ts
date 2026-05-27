import {hashCode, isValidURL} from './utils';

describe('utils', () => {
    test('hashCode', () => {
        expect(hashCode('test')).toBe(hashCode('test'));
        expect(hashCode('a')).not.toBe(hashCode('b'));
    });

    test('isValidURL rejects javascript protocol', () => {
        expect(isValidURL('javascript:alert(1)')).toBe(false);
    });

    test('isValidURL accepts https URL', () => {
        expect(isValidURL('https://example.com')).toBe(true);
    });
});
