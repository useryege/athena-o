import * as agent from 'superagent';

import {Observable, Observer, Subject} from 'rxjs';

type Callback = (data: any) => void;

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
            onError.next(err);
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

            const abortController = new AbortController();

            // If there is an error, show it beforehand
            fetch(fullUrl, {signal: abortController.signal})
                .then(response => {
                    if (!response.ok) {
                        return response.text().then(text => {
                            observer.error({status: response.status, statusText: response.statusText, body: text});
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
