import {Context, type ModalHandle} from './context';
import * as React from 'react';
import {Button, Result} from 'antd';
import {clearAsyncDataCache} from '../components/data';
import requests, {accountModuleForAccess, requestErrorDetails} from './services/requests';
import {isModuleAccessReason, isModuleKey, moduleAccessDefinitions, moduleAccessService, type ModuleKey, type ModuleAccessState} from './module-access-service';
import type {AbortablePromise} from './use-visible-query';

type Access = 'unknown' | 'open' | 'closed';
type Snapshot = Record<ModuleKey, {state: Access; epoch: number}>;
const initialSnapshot = (): Snapshot => Object.fromEntries(moduleAccessDefinitions.map(({key}) => [key, {state: 'unknown', epoch: 0}])) as Snapshot;
const foreground = () => document.visibilityState !== 'hidden' && navigator.onLine;
const callbackFields = ['wormCredentialReason', 'wormTradingReason', 'wormExecutionReason', 'wormPositionCashOutReason', 'wormPositionCashOutBatchReason'];
const callbackFailure = (search: string) => {
    const query = new URLSearchParams(search);
    return callbackFields.some(key => isModuleAccessReason(query.get(key)));
};

/** One authenticated application owns one controller; no access decision is persisted or shared across realms. */
class ModuleAccessController {
    snapshot = initialSnapshot();
    wormResumeRequiresAction = false;
    private listeners = new Set<() => void>();
    private sequence = 0;
    private active = false;
    constructor(private onInvalidate?: (key: ModuleKey) => void) {}
    private pending?: AbortablePromise<ModuleAccessState[]>;
    private expiry?: number;
    private expiresAt = 0;
    subscribe = (listener: () => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };
    getSnapshot = () => this.snapshot;
    private publish(states: Partial<Record<ModuleKey, Access>>) {
        const next = {...this.snapshot};
        const invalidated: ModuleKey[] = [];
        for (const {key} of moduleAccessDefinitions) {
            const state = states[key] ?? next[key].state;
            if (state === next[key].state) continue;
            next[key] = {state, epoch: next[key].epoch + 1};
            if (state !== 'open') invalidated.push(key);
        }
        // Withdraw access before abort handlers or cache listeners can run.
        this.snapshot = next;
        for (const key of invalidated) {
            requests.abortModuleAccessRequests(key);
            this.onInvalidate?.(key);
            if (key === 'worm') window.sessionStorage.removeItem('athena.member.worm-trading.pending-connection-action');
            const module = accountModuleForAccess[key];
            if (module !== undefined) clearAsyncDataCache(module);
        }
        this.listeners.forEach(listener => listener());
    }
    invalidate = (key?: ModuleKey) => {
        if (!key || key === 'worm') this.wormResumeRequiresAction = true;
        this.sequence++;
        this.pending?.abort?.();
        this.pending = undefined;
        if (key) this.publish({[key]: 'unknown'});
        else {
            window.clearTimeout(this.expiry);
            this.publish(Object.fromEntries(moduleAccessDefinitions.map(item => [item.key, 'unknown'])));
        }
    };
    refresh = () => {
        if (!this.active || !foreground() || this.pending) return;
        const sequence = ++this.sequence;
        const started = performance.now();
        const request = moduleAccessService.states();
        this.pending = request;
        void request
            .then(
                rows => {
                    if (!this.active || sequence !== this.sequence || !foreground()) return;
                    const remaining = 5000 - (performance.now() - started);
                    if (remaining <= 0) {
                        this.invalidate();
                        return;
                    }
                    this.expiresAt = started + 5000;
                    this.publish(Object.fromEntries(rows.map(row => [row.module_key, row.state === 1 ? 'open' : 'closed'])));
                    window.clearTimeout(this.expiry);
                    this.expiry = window.setTimeout(() => this.invalidate(), remaining);
                },
                () => {
                    if (this.active && sequence === this.sequence) this.invalidate();
                }
            )
            .finally(() => {
                if (sequence === this.sequence) this.pending = undefined;
            });
    };
    start(realm: 'member' | 'admin') {
        this.active = true;
        const unregisterGuard = requests.registerModuleAccessGuard(realm, key => this.capture(key)());
        const reset = () => {
            this.invalidate();
            this.refresh();
        };
        document.addEventListener('visibilitychange', reset);
        for (const event of ['offline', 'online', 'focus', 'pageshow']) window.addEventListener(event, reset);
        const errors = requests.onError.subscribe(error => {
            const details = requestErrorDetails(error);
            if (!isModuleAccessReason(details.reason)) return;
            this.invalidate(isModuleKey(details.moduleKey) ? details.moduleKey : undefined);
            // The regular poll retries; a failing state endpoint must not create an immediate retry loop.
        });
        const timer = window.setInterval(this.refresh, 2000);
        this.refresh();
        return () => {
            this.active = false;
            this.invalidate();
            unregisterGuard();
            errors.unsubscribe();
            window.clearInterval(timer);
            document.removeEventListener('visibilitychange', reset);
            for (const event of ['offline', 'online', 'focus', 'pageshow']) window.removeEventListener(event, reset);
        };
    }
    capture(key: ModuleKey) {
        const epoch = this.snapshot[key].epoch;
        return () => this.active && foreground() && performance.now() < this.expiresAt && this.snapshot[key].state === 'open' && this.snapshot[key].epoch === epoch;
    }
}
const PendingWormReturnContext = React.createContext(false);
const ModuleAccessContext = React.createContext<ModuleAccessController | null>(null);
const ControllerProvider = ({
    children,
    returnSearch = '',
    onInvalidate,
    realm
}: {
    realm: 'member' | 'admin';
    children: React.ReactNode;
    returnSearch?: string;
    onInvalidate?: (key: ModuleKey) => void;
}) => {
    const [controller] = React.useState(() => new ModuleAccessController(onInvalidate));
    const [processedReturn, setProcessedReturn] = React.useState(returnSearch);
    const callbackPending = returnSearch !== processedReturn && callbackFailure(returnSearch);
    React.useLayoutEffect(() => {
        if (callbackFailure(returnSearch)) {
            controller.invalidate('worm');
            window.sessionStorage.removeItem('athena.member.worm-trading.pending-connection-action');
            window.sessionStorage.removeItem('athena.member.worm-execution.authorization');
            const url = new URL(window.location.href);
            for (const field of callbackFields) if (isModuleAccessReason(url.searchParams.get(field))) url.searchParams.delete(field);
            window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
            controller.refresh();
        }
        setProcessedReturn(returnSearch);
    }, [controller, returnSearch]);
    React.useLayoutEffect(() => controller.start(realm), [controller, realm]);
    return (
        <ModuleAccessContext.Provider value={controller}>
            <PendingWormReturnContext.Provider value={callbackPending}>{children}</PendingWormReturnContext.Provider>
        </ModuleAccessContext.Provider>
    );
};
export const ModuleAccessProvider = (props: {
    realm: 'member' | 'admin';
    identity: string;
    returnSearch?: string;
    onInvalidate?: (key: ModuleKey) => void;
    children: React.ReactNode;
}) => (
    <ControllerProvider realm={props.realm} key={`${props.realm}:${props.identity}`} returnSearch={props.returnSearch} onInvalidate={props.onInvalidate}>
        {props.children}
    </ControllerProvider>
);
export const useModuleAccessReopened = (key: ModuleKey) => {
    const controller = React.useContext(ModuleAccessContext);
    return Boolean(controller && (controller.snapshot[key].epoch > 1 || (key === 'worm' && controller.wormResumeRequiresAction)));
};
export const useModuleAccessLease = (key: ModuleKey) => {
    const controller = React.useContext(ModuleAccessContext);
    return React.useCallback(() => (controller ? controller.capture(key) : () => true), [controller, key]);
};
/** Imperative modals are rendered outside the route tree; own their handles and callbacks here. */
const ModuleBusinessScope = ({moduleKey, children}: {moduleKey: ModuleKey; children: React.ReactNode}) => {
    const context = React.useContext(Context);
    const capture = useModuleAccessLease(moduleKey);
    const lifetime = React.useMemo(() => ({active: true, current: capture(), handles: new Set<ModalHandle>()}), [capture]);
    React.useLayoutEffect(() => {
        lifetime.active = true;
        return () => {
            lifetime.active = false;
            lifetime.handles.forEach(handle => handle.destroy());
            lifetime.handles.clear();
        };
    }, [lifetime]);
    const value = React.useMemo(
        () =>
            context && {
                ...context,
                modal: {
                    ...context.modal,
                    confirm: (options: Parameters<typeof context.modal.confirm>[0]) => {
                        const current = () => lifetime.active && lifetime.current();
                        if (!current()) return {destroy: () => undefined};
                        const handle = context.modal.confirm({
                            ...options,
                            onOk: () => {
                                if (current()) return options.onOk?.();
                            },
                            onCancel: () => {
                                if (current()) options.onCancel?.();
                            }
                        });
                        lifetime.handles.add(handle);
                        return handle;
                    }
                }
            },
        [context, lifetime]
    );
    return value ? <Context.Provider value={value}>{children}</Context.Provider> : <>{children}</>;
};

export const ModuleAccessBoundary = ({moduleKey, children}: {moduleKey: ModuleKey; children: React.ReactNode}) => {
    const controller = React.useContext(ModuleAccessContext);
    if (!controller) throw new Error('Module access provider is unavailable');
    const snapshot = React.useSyncExternalStore(controller.subscribe, controller.getSnapshot);
    const callbackPending = React.useContext(PendingWormReturnContext);
    const access = callbackPending && moduleKey === 'worm' ? {state: 'unknown', epoch: snapshot[moduleKey].epoch} : snapshot[moduleKey];
    if (access.state === 'open')
        return (
            <ModuleBusinessScope key={access.epoch} moduleKey={moduleKey}>
                {children}
            </ModuleBusinessScope>
        );
    return (
        <div role='status'>
            <Result
                status='info'
                title={access.state === 'closed' ? 'This module is not open yet' : 'Module access could not be confirmed'}
                subTitle='Your account and navigation remain available. Background tasks and notifications continue.'
                extra={<Button onClick={controller.refresh}>Check access</Button>}
            />
        </div>
    );
};
