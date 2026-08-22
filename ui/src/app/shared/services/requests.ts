import * as agent from 'superagent';

import {Observable, Observer, Subject} from 'rxjs';

type Callback = (data: any) => void;

export const ACCOUNT_MAINTENANCE_MESSAGE = '系统维护中';

export interface RequestErrorDetails {
    status?: number;
    code?: number;
    message?: string;
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

export const requestErrorDetails = (error: unknown): RequestErrorDetails => {
    const source = isRecord(error) ? error : {};
    const response = isRecord(source.response) ? source.response : {};
    const status = asNumber(source.status ?? response.status);
    const bodyCandidates = [response.body, source.body, response.text, source.text];

    for (const candidate of bodyCandidates) {
        const parsed = parseErrorBody(candidate);
        if (!parsed) {
            continue;
        }
        const gatewayError = isRecord(parsed.error) ? parsed.error : parsed;
        const code = asNumber(gatewayError.code);
        const message = typeof gatewayError.message === 'string' ? gatewayError.message : undefined;
        if (code !== undefined || message !== undefined) {
            return {status, code, message};
        }
    }

    return {status};
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

const normalizeRequestError = <T,>(error: T): T => {
    const details = requestErrorDetails(error);
    if (details.message && isRecord(error)) {
        try {
            error.message = details.message;
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

function apiRoot(): string {
    return toAbsURL('/api/v1');
}

function initHandlers(req: agent.Request) {
    const generation = requestErrorGeneration;
    req.on('error', err => {
        if (generation === requestErrorGeneration) {
            onError.next(normalizeRequestError(err));
        }
    });
    return req;
}

export default {
    setBaseHRef(val: string) {
        baseHRef = val;
    },
    agent,
    toAbsURL,
    onError: onError.asObservable(),
    invalidatePendingRequestErrors() {
        requestErrorGeneration++;
    },
    get(url: string) {
        return initHandlers(agent.get(`${apiRoot()}${url}`));
    },

    post(url: string) {
        return initHandlers(agent.post(`${apiRoot()}${url}`)).set('Content-Type', 'application/json');
    },

    put(url: string) {
        return initHandlers(agent.put(`${apiRoot()}${url}`)).set('Content-Type', 'application/json');
    },

    patch(url: string) {
        return initHandlers(agent.patch(`${apiRoot()}${url}`)).set('Content-Type', 'application/json');
    },

    delete(url: string) {
        return initHandlers(agent.del(`${apiRoot()}${url}`)).set('Content-Type', 'application/json');
    },

    loadEventSource(url: string): Observable<string> {
        return Observable.create((observer: Observer<any>) => {
            const fullUrl = `${apiRoot()}${url}`;
            const generation = requestErrorGeneration;

            const abortController = new AbortController();

            // If there is an error, show it beforehand
            fetch(fullUrl, {signal: abortController.signal})
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
