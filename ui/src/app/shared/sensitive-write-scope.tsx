import * as React from 'react';
import {AccountDataModule} from './access-modules';
import {useAuthorization} from './context';

export type AbortableSensitiveTask<T> = Promise<T> & {abort?: () => void};

export type SensitiveTaskResult<T> = {status: 'fulfilled'; value: T} | {status: 'rejected'; error: unknown} | {status: 'discarded'};

export interface SensitiveWriteLease {
    runTask<T>(start: () => AbortableSensitiveTask<T>): Promise<SensitiveTaskResult<T>>;
}

interface TrackedTask {
    abort?: () => void;
}

const isAbortError = (error: unknown) =>
    Boolean(error) && typeof error === 'object' && (('code' in error && error.code === 'ABORTED') || ('name' in error && error.name === 'AbortError'));

class SensitiveWriteLeaseState implements SensitiveWriteLease {
    private active = true;
    private generation = 1;
    private readonly tasks = new Set<TrackedTask>();

    public activate() {
        if (this.active) {
            return;
        }
        this.generation++;
        this.active = true;
    }

    public deactivate() {
        if (!this.active && this.tasks.size === 0) {
            return;
        }
        this.active = false;
        this.generation++;
        const tasks = Array.from(this.tasks);
        this.tasks.clear();
        tasks.forEach(task => {
            try {
                task.abort?.();
            } catch {
                // An abort failure cannot make a task current again.
            }
        });
    }

    public async runTask<T>(start: () => AbortableSensitiveTask<T>): Promise<SensitiveTaskResult<T>> {
        if (!this.active) {
            return {status: 'discarded'};
        }

        let promise: AbortableSensitiveTask<T>;
        try {
            promise = start();
        } catch (error) {
            return this.active ? {status: 'rejected', error} : {status: 'discarded'};
        }

        const generation = this.generation;
        const task: TrackedTask = {abort: promise.abort ? () => promise.abort?.() : undefined};
        this.tasks.add(task);
        const isCurrent = () => this.active && this.generation === generation && this.tasks.has(task);

        try {
            const value = await promise;
            return isCurrent() ? {status: 'fulfilled', value} : {status: 'discarded'};
        } catch (error) {
            return isCurrent() && !isAbortError(error) ? {status: 'rejected', error} : {status: 'discarded'};
        } finally {
            this.tasks.delete(task);
        }
    }
}

const SensitiveWriteLeaseContext = React.createContext<SensitiveWriteLease>(null);

const SensitiveWriteLeaseProvider = (props: {children: React.ReactNode}) => {
    const leaseRef = React.useRef<SensitiveWriteLeaseState>(null);
    if (!leaseRef.current) {
        leaseRef.current = new SensitiveWriteLeaseState();
    }
    const lease = leaseRef.current;

    React.useLayoutEffect(() => {
        lease.activate();
        return () => lease.deactivate();
    }, [lease]);

    return <SensitiveWriteLeaseContext.Provider value={lease}>{props.children}</SensitiveWriteLeaseContext.Provider>;
};

export const SensitiveWriteScope = (props: {module: AccountDataModule; children: React.ReactNode}) => {
    const authorization = useAuthorization();
    if (!authorization.canWrite(props.module)) {
        return null;
    }

    const identity = JSON.stringify([authorization.user.iss || '', authorization.user.username, authorization.user.administrator, props.module]);
    return <SensitiveWriteLeaseProvider key={identity}>{props.children}</SensitiveWriteLeaseProvider>;
};

export const useSensitiveWriteLease = (): SensitiveWriteLease => {
    const lease = React.useContext(SensitiveWriteLeaseContext);
    if (!lease) {
        throw new Error('Sensitive write lease is unavailable');
    }
    return lease;
};
