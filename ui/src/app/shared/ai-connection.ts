export interface AIConnectionDetails {
    baseUrl: string;
    llmsUrl: string;
    swaggerUrl: string;
    verifyUrl: string;
    expectedAccountId: string;
    authorizationHeader: string;
    instructions: string;
}

export type AIConnectionVerification = {status: 'ready'; message: string} | {status: 'failed'; message: string};

export const createAIConnectionID = (): string => {
    const now = new Date();
    const timestamp = [
        now.getUTCFullYear().toString().padStart(4, '0'),
        (now.getUTCMonth() + 1).toString().padStart(2, '0'),
        now.getUTCDate().toString().padStart(2, '0'),
        '-',
        now.getUTCHours().toString().padStart(2, '0'),
        now.getUTCMinutes().toString().padStart(2, '0'),
        now.getUTCSeconds().toString().padStart(2, '0')
    ].join('');
    const random = new Uint8Array(2);
    globalThis.crypto.getRandomValues(random);
    const suffix = Array.from(random, value => value.toString(16).padStart(2, '0')).join('');
    return `ai-${timestamp}-${suffix}`;
};

export const buildAIConnectionDetails = (input: {baseURI: string; accountId: string; secret: string}): AIConnectionDetails => {
    const base = new URL(input.baseURI);
    base.hash = '';
    base.search = '';
    if (!base.pathname.endsWith('/')) {
        base.pathname = `${base.pathname}/`;
    }

    const baseUrl = base.href.replace(/\/$/, '');
    const llmsUrl = new URL('llms.txt', base).href;
    const swaggerUrl = new URL('swagger.json', base).href;
    const verifyUrl = new URL('api/v1/session/userinfo', base).href;
    const authorizationHeader = `Bearer ${input.secret}`;
    const instructions = [
        'Connect to this Athena instance and act as my account.',
        '',
        `Athena Base URL: ${baseUrl}`,
        `AI discovery document: ${llmsUrl}`,
        `API contract (Swagger 2.0): ${swaggerUrl}`,
        `Credential verification endpoint: ${verifyUrl}`,
        `Expected account ID: ${input.accountId}`,
        'Authorization header:',
        `Authorization: ${authorizationHeader}`,
        '',
        'Connection procedure:',
        `1. Read ${llmsUrl} before selecting or calling Athena operations.`,
        `2. Treat the Swagger 2.0 document at ${swaggerUrl} as the source of truth for paths, methods, parameters, request bodies, response schemas, and enum values.`,
        `3. Send a GET request to ${verifyUrl} with the Authorization header above and confirm that loggedIn is true and accountId equals ${input.accountId}.`,
        '4. After verification, act with the full current authority of this ordinary Athena account, including all read and write operations the account is currently allowed to perform.',
        '',
        'Authorization model:',
        '- This bearer credential represents the complete current API authority of the account.',
        '- Athena does not use OAuth delegation, API Key scopes, a read-only mode, or per-request approval for this connection.',
        '- Athena evaluates account permissions in real time. Permission changes, account suspension, API Key access changes, expiration, or revocation take effect on subsequent requests.'
    ].join('\n');

    return {
        baseUrl,
        llmsUrl,
        swaggerUrl,
        verifyUrl,
        expectedAccountId: input.accountId,
        authorizationHeader,
        instructions
    };
};

export const verifyAIConnectionCredential = async (
    details: Pick<AIConnectionDetails, 'verifyUrl' | 'expectedAccountId' | 'authorizationHeader'>,
    signal?: AbortSignal
): Promise<AIConnectionVerification> => {
    try {
        const response = await fetch(details.verifyUrl, {
            method: 'GET',
            credentials: 'omit',
            cache: 'no-store',
            signal,
            headers: {
                Authorization: details.authorizationHeader,
                Accept: 'application/json'
            }
        });
        if (!response.ok) {
            return {
                status: 'failed',
                message: `Credential verification failed with HTTP ${response.status}. You can retry the check or copy the connection instructions now.`
            };
        }

        const payload: unknown = await response.json();
        if (typeof payload !== 'object' || payload === null || !('loggedIn' in payload) || payload.loggedIn !== true) {
            return {
                status: 'failed',
                message: 'Credential verification did not return a signed-in Athena account. You can retry the check or copy the connection instructions now.'
            };
        }
        if (!('accountId' in payload) || payload.accountId !== details.expectedAccountId) {
            return {
                status: 'failed',
                message: 'Credential verification returned a different Athena account. You can retry the check or copy the connection instructions now.'
            };
        }

        return {
            status: 'ready',
            message: 'Credential ready. Athena accepted the API key for the expected account. This check does not confirm that an external AI is connected.'
        };
    } catch {
        return {
            status: 'failed',
            message: 'Credential verification could not be completed. You can retry the check or copy the connection instructions now.'
        };
    }
};
