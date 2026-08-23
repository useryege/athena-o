const hasUnsafeReturnToCharacter = (value: string) => value.includes('\\') || Array.from(value).some(character => character.charCodeAt(0) < 32 || character.charCodeAt(0) === 127);

export const readLoginReturnTo = (search: string, fallback = '/settings') => {
    const candidate = new URLSearchParams(search).get('returnTo');
    if (!candidate || !candidate.startsWith('/') || candidate.startsWith('//') || hasUnsafeReturnToCharacter(candidate)) {
        return fallback;
    }
    try {
        const expectedOrigin = 'https://athena.local';
        const target = new URL(candidate, expectedOrigin);
        const decodedPath = decodeURIComponent(target.pathname).toLowerCase();
        if (target.origin !== expectedOrigin || decodedPath === '/login' || decodedPath.startsWith('/login/')) {
            return fallback;
        }
        return `${target.pathname}${target.search}${target.hash}`;
    } catch {
        return fallback;
    }
};

export const loginPathFor = (pathname: string, search = '', hash = '') => {
    const returnTo = `${pathname || '/'}${search}${hash}`;
    if (returnTo === '/') {
        return '/login';
    }
    return `/login?returnTo=${encodeURIComponent(returnTo)}`;
};
