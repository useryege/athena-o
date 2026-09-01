import * as agent from 'superagent';

import {Observable, Observer, Subject} from 'rxjs';
import {AccountDataModule} from '../access-modules';

type Callback = (data: any) => void;

export const ACCOUNT_MAINTENANCE_MESSAGE = '系统维护中';
export const ACCOUNT_DATA_ACCESS_DENIED_REASON = 'ACCOUNT_DATA_ACCESS_DENIED';

export interface RequestErrorDetails {
    status?: number;
    code?: number;
    message?: string;
    reason?: string;
}

declare class EventSource {
    public onopen: Callback;
    public onmessage: Callback;
    public onerror: Callback;
    public readyState: number;
    constructor(url: string);
    public close(): void;
}

enum ReadyState {
    CONNECTING = 0,
    OPEN = 1,
    CLOSED = 2,
    DONE = 4
}

let baseHRef = '/';

const onError = new Subject<agent.ResponseError>();
let requestErrorGeneration = 0;

export type AuthorizationRequestMode = 'read' | 'write';
export type AuthorizationRequestFeature = 'api-key' | 'profit-sharing' | 'self-account' | 'admin-accounts' | 'admin-service-status' | 'admin-etherscan';
export type AuthorizationRequestRealm = 'member' | 'admin';

export const APPLICATION_REALM_HEADER = 'X-Athena-Application-Realm';
export const APPLICATION_REALM_QUERY = 'athenaRealm';

export type AuthorizationRequestScope =
    | {
          session: true;
      }
    | {
          module: AccountDataModule;
          mode: AuthorizationRequestMode;
      }
    | {
          feature: AuthorizationRequestFeature;
          mode: AuthorizationRequestMode;
      };

interface ScopedAuthorizationRequest {
    realm: AuthorizationRequestRealm | 'session';
    viewerAccountId: string;
    sessionGeneration: number;
    scope: AuthorizationRequestScope;
}

interface AbortableRequest {
    abort(): void;
}

let authorizationRealm: AuthorizationRequestRealm | undefined;
let authorizationViewerAccountId = '';
let authorizationSessionGeneration = 0;
const scopedRequests = new Map<AbortableRequest, ScopedAuthorizationRequest>();

const isRecord = (value: unknown): value is Record<string, any> => Boolean(value) && typeof value === 'object';

const asNumber = (value: unknown): number | undefined => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
};

const parseErrorBody = (value: unknown): Record<string, any> | undefined => {
    if (isRecord(value)) {
        return value;
    }
    if (typeof value !== 'string' || !value.trim()) {
        return undefined;
    }
    try {
        const parsed = JSON.parse(value);
        return isRecord(parsed) ? parsed : undefined;
    } catch {
        return undefined;
    }
};

const findErrorReason = (value: unknown, depth = 0): string | undefined => {
    if (depth > 6) {
        return undefined;
    }
    if (Array.isArray(value)) {
        for (const item of value) {
            const reason = findErrorReason(item, depth + 1);
            if (reason) {
                return reason;
            }
        }
        return undefined;
    }
    if (!isRecord(value)) {
        return undefined;
    }
    if (typeof value.reason === 'string' && value.reason) {
        return value.reason;
    }
    for (const nested of Object.values(value)) {
        if (Array.isArray(nested) || isRecord(nested)) {
            const reason = findErrorReason(nested, depth + 1);
            if (reason) {
                return reason;
            }
        }
    }
    return undefined;
};

export const requestErrorDetails = (error: unknown): RequestErrorDetails => {
    const source = isRecord(error) ? error : {};
    const response = isRecord(source.response) ? source.response : {};
    const status = asNumber(source.status ?? response.status);
    const bodyCandidates = [response.body, source.body, response.text, source.text];
    const headers = isRecord(response.headers) ? response.headers : isRecord(source.headers) ? source.headers : {};
    let code: number | undefined;
    let message: string | undefined;
    let reason =
        (typeof headers['x-athena-error-reason'] === 'string' && headers['x-athena-error-reason']) ||
        (typeof headers['X-Athena-Error-Reason'] === 'string' && headers['X-Athena-Error-Reason']) ||
        undefined;

    for (const candidate of bodyCandidates) {
        const parsed = parseErrorBody(candidate);
        if (!parsed) {
            continue;
        }
        const gatewayError = isRecord(parsed.error) ? parsed.error : parsed;
        code ??= asNumber(gatewayError.code);
        message ??= typeof gatewayError.message === 'string' ? gatewayError.message : undefined;
        reason ??= findErrorReason(parsed);
    }

    return {status, code, message, reason};
};

export const requestErrorMessage = (error: unknown, fallback = 'Request failed'): string => {
    const details = requestErrorDetails(error);
    if (details.message) {
        return details.message;
    }
    if (isRecord(error) && typeof error.message === 'string' && error.message) {
        return error.message;
    }
    return fallback;
};

export const isAccountMaintenanceError = (error: unknown): boolean => {
    const details = requestErrorDetails(error);
    return details.status === 503 && details.code === 14 && details.message === ACCOUNT_MAINTENANCE_MESSAGE;
};

export const isAccountDataAccessDeniedError = (error: unknown): boolean => {
    const details = requestErrorDetails(error);
    return details.status === 403 && (details.reason === ACCOUNT_DATA_ACCESS_DENIED_REASON || (details.code === 7 && details.message === ACCOUNT_DATA_ACCESS_DENIED_REASON));
};

const normalizeRequestError = <T>(error: T): T => {
    const details = requestErrorDetails(error);
    if (details.message && isRecord(error)) {
        try {
            (error as Record<string, any>).message = details.message;
        } catch {
            // Some third-party errors expose a read-only message. Consumers can
            // still obtain the gateway message through requestErrorMessage.
        }
    }
    return error;
};

const httpError = (status: number, statusText: string, body: unknown) => {
    const error = new Error(statusText || `Request failed (${status})`) as Error & {
        status: number;
        statusText: string;
        body: unknown;
    };
    error.status = status;
    error.statusText = statusText;
    error.body = body;
    return normalizeRequestError(error);
};

function toAbsURL(val: string): string {
    const base = (baseHRef || '/').replace(/\/+$/, '');
    const next = (val || '').replace(/^\/+/, '');
    const result = `${base || ''}/${next}`;
    return result === '/' ? '/' : result;
}

const appendApplicationRealmQuery = (url: string): string => {
    if (!authorizationRealm) {
        return url;
    }
    const parsed = new URL(url, 'https://athena.local');
    parsed.searchParams.set(APPLICATION_REALM_QUERY, authorizationRealm);
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
};

// Private API resources rendered directly by the browser (for example, an
// uploaded avatar in an <img>) cannot carry the application-realm header.
// Bind only Athena-owned relative API URLs; external and browser-owned URLs
// must remain byte-for-byte unchanged.
export const realmBoundResourceURL = (url: string): string => {
    if (!url || !authorizationRealm || /^[a-z][a-z\d+.-]*:/i.test(url) || url.startsWith('//')) {
        return url;
    }
    const logicalURL = url.startsWith('/') ? url : `/${url}`;
    if (logicalURL.startsWith('/api/')) {
        return appendApplicationRealmQuery(toAbsURL(logicalURL));
    }
    const deploymentRoot = `/${(baseHRef || '/').replace(/^\/+|\/+$/g, '')}`.replace(/^\/$/, '');
    if (!deploymentRoot || !logicalURL.startsWith(`${deploymentRoot}/api/`)) {
        return url;
    }
    return appendApplicationRealmQuery(logicalURL);
};

function apiRoot(): string {
    return toAbsURL('/api/v1');
}

const trackScopedRequest = (request: AbortableRequest, scope?: AuthorizationRequestScope) => {
    if (!scope) {
        return;
    }
    const sessionScope = 'session' in scope;
    if (!sessionScope && (!authorizationRealm || !authorizationViewerAccountId)) {
        throw new Error('Authenticated request scope is unavailable before the application realm guard completes');
    }
    scopedRequests.set(request, {
        realm: sessionScope ? 'session' : authorizationRealm!,
        viewerAccountId: authorizationViewerAccountId,
        sessionGeneration: authorizationSessionGeneration,
        scope
    });
};

function initHandlers(req: agent.Request, scope?: AuthorizationRequestScope) {
    const generation = requestErrorGeneration;
    if (authorizationRealm) {
        req.set(APPLICATION_REALM_HEADER, authorizationRealm);
    }
    trackScopedRequest(req, scope);
    const removeScope = () => scopedRequests.delete(req);
    req.on('error', err => {
        removeScope();
        if (generation === requestErrorGeneration) {
            onError.next(normalizeRequestError(err));
        }
    });
    req.on('end', removeScope);
    req.on('abort', removeScope);
    return req;
}

const abortAuthorizationRequests = (module?: AccountDataModule, mode?: AuthorizationRequestMode) => {
    Array.from(scopedRequests.entries()).forEach(([request, tracked]) => {
        const scope = tracked.scope;
        if ((module === undefined || ('module' in scope && scope.module === module)) && (mode === undefined || ('mode' in scope && scope.mode === mode))) {
            scopedRequests.delete(request);
            request.abort();
        }
    });
};

const abortAuthorizationFeatureRequests = (feature: AuthorizationRequestFeature, mode?: AuthorizationRequestMode) => {
    Array.from(scopedRequests.entries()).forEach(([request, tracked]) => {
        const scope = tracked.scope;
        if ('feature' in scope && scope.feature === feature && (mode === undefined || scope.mode === mode)) {
            scopedRequests.delete(request);
            request.abort();
        }
    });
};

export default {
    setBaseHRef(val: string) {
        baseHRef = val;
    },
    toAbsURL,
    onError: onError.asObservable(),
    invalidatePendingRequestErrors() {
        requestErrorGeneration++;
    },
    configureAuthorizationRealm(realm: AuthorizationRequestRealm) {
        if (realm !== 'member' && realm !== 'admin') {
            throw new Error(`Unsupported Athena application realm: ${String(realm)}`);
        }
        if (authorizationRealm && authorizationRealm !== realm) {
            throw new Error(`Authorization requests are already configured for the ${authorizationRealm} realm`);
        }
        authorizationRealm = realm;
    },
    beginAuthorizationSession(viewerAccountId: string) {
        if (!authorizationRealm) {
            throw new Error('Authorization request realm must be configured before beginning a session');
        }
        if (authorizationViewerAccountId !== viewerAccountId) {
            abortAuthorizationRequests();
            authorizationSessionGeneration++;
            authorizationViewerAccountId = viewerAccountId;
        }
        return authorizationSessionGeneration;
    },
    endAuthorizationSession() {
        abortAuthorizationRequests();
        authorizationSessionGeneration++;
        authorizationViewerAccountId = '';
    },
    abortAuthorizationRequests,
    abortAuthorizationFeatureRequests,
    get(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.get(`${apiRoot()}${url}`), scope);
    },

    rawGet(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.get(toAbsURL(url)), scope);
    },

    post(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.post(`${apiRoot()}${url}`), scope).set('Content-Type', 'application/json');
    },

    put(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.put(`${apiRoot()}${url}`), scope).set('Content-Type', 'application/json');
    },

    rawPut(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.put(toAbsURL(url)), scope);
    },

    patch(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.patch(`${apiRoot()}${url}`), scope).set('Content-Type', 'application/json');
    },

    delete(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.del(`${apiRoot()}${url}`), scope).set('Content-Type', 'application/json');
    },

    rawDelete(url: string, scope?: AuthorizationRequestScope) {
        return initHandlers(agent.del(toAbsURL(url)), scope);
    },

    scopedFetch(url: string, init: RequestInit, scope: AuthorizationRequestScope): Promise<Response> & {abort?: () => void} {
        const controller = new AbortController();
        const tracked = {abort: () => controller.abort()};
        const upstreamSignal = init.signal;
        if (upstreamSignal?.aborted) {
            controller.abort();
        } else {
            upstreamSignal?.addEventListener('abort', tracked.abort, {once: true});
        }
        trackScopedRequest(tracked, scope);
        const cleanup = () => {
            upstreamSignal?.removeEventListener('abort', tracked.abort);
            scopedRequests.delete(tracked);
        };
        const headers = new Headers(init.headers);
        if (authorizationRealm) {
            headers.set(APPLICATION_REALM_HEADER, authorizationRealm);
        }
        const promise = fetch(url, {...init, headers, signal: controller.signal}) as Promise<Response> & {abort?: () => void};
        promise.abort = tracked.abort;
        void promise.then(cleanup, cleanup);
        return promise;
    },

    loadEventSource(url: string): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            const fullUrl = appendApplicationRealmQuery(`${apiRoot()}${url}`);
            const generation = requestErrorGeneration;

            const abortController = new AbortController();

            // If there is an error, show it beforehand
            const headers = authorizationRealm ? {[APPLICATION_REALM_HEADER]: authorizationRealm} : undefined;
            fetch(fullUrl, {headers, signal: abortController.signal})
                .then(response => {
                    if (!response.ok) {
                        return response.text().then(text => {
                            const error = httpError(response.status, response.statusText, text);
                            observer.error(error);
                            if (generation === requestErrorGeneration) {
                                onError.next(error as agent.ResponseError);
                            }
                        });
                    }
                })
                .catch(err => {
                    if (err.name === 'AbortError') {
                        return;
                    }
                    observer.error(err);
                });

            let eventSource = new EventSource(fullUrl);
            eventSource.onmessage = msg => observer.next(msg.data);
            eventSource.onerror = e => () => {
                observer.error(e);
                onError.next(e);
            };

            // EventSource does not provide easy way to get notification when connection closed.
            // check readyState periodically instead.
            const interval = setInterval(() => {
                if (eventSource && eventSource.readyState === ReadyState.CLOSED) {
                    observer.error('connection got closed unexpectedly');
                }
            }, 500);
            return () => {
                clearInterval(interval);
                eventSource.close();
                abortController.abort();
                eventSource = null;
            };
        });
    }
};
