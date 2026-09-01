const normalizeBasePath = (value: string | null | undefined): string => {
    const candidate = (value || '/').trim() || '/';
    let pathname = candidate;
    try {
        pathname = new URL(candidate, window.location.origin).pathname;
    } catch {
        // The server writes path-only values. Retain that value if a browser
        // URL implementation rejects malformed external input.
    }
    const segments = pathname.split('/').filter(Boolean);
    return segments.length === 0 ? '/' : `/${segments.join('/')}/`;
};

export const readApplicationBaseHRef = (): string => normalizeBasePath(document.querySelector('base')?.getAttribute('href'));

export const readDeploymentBaseHRef = (): string =>
    normalizeBasePath(document.querySelector('meta[name="athena-deployment-base-href"]')?.getAttribute('content') || readApplicationBaseHRef());

export const deploymentPath = (value: string): string => {
    const base = readDeploymentBaseHRef();
    const suffix = value.replace(/^\/+/, '');
    return suffix ? `${base}${suffix}` : base;
};
