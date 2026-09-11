import {useSyncExternalStore} from 'react';
import {useAuthorization} from '../shared/context';
import type {ReadScope} from '../shared/use-visible-query';
import type {UserInfo} from '../shared/models';

let identity = '';
let generation = 0;
const listeners = new Set<() => void>();
const identityOf = (user: UserInfo) => (user.loggedIn && user.administrator && user.accountId ? JSON.stringify([user.accountId, user.iss]) : '');
const publish = () => {
    generation++;
    for (const listener of [...listeners]) listener();
};
const subscribe = (listener: () => void) => {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
};
export const beginAdminReadSession = (user: UserInfo) => {
    const next = identityOf(user);
    if (next !== identity) {
        identity = next;
        publish();
    }
};
export const endAdminReadSession = () => {
    identity = '';
    publish();
};

/** The Shell owns session facts; a page owns only its resource and mounted reads. */
export const useAdminReadScope = (resource: string): ReadScope => {
    const {user, isAdmin} = useAuthorization();
    const epoch = useSyncExternalStore(subscribe, () => generation);
    const expectedIdentity = identityOf(user);
    const allowed = isAdmin && Boolean(expectedIdentity);
    return {
        key: JSON.stringify([expectedIdentity, allowed, epoch, resource]),
        isCurrent: () => allowed && expectedIdentity === identity && epoch === generation,
        subscribeInvalidation: subscribe
    };
};
