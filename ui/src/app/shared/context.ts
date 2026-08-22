import * as React from 'react';
import * as models from './models';

export interface NavigationApi {
    goto(path: string): void;
    replace(path: string): void;
}

export interface NotificationsApi {
    success(message: string, description?: string): void;
    error(message: string, description?: string): void;
    info(message: string, description?: string): void;
    warning(message: string, description?: string): void;
}

export interface ModalApi {
    confirm(options: {title: string; content?: React.ReactNode; onOk?: () => void | Promise<void>}): void;
    info(options: {title: string; content?: React.ReactNode}): void;
    error(options: {title: string; content?: React.ReactNode}): void;
}

export type AppContext = {apis: ContextApis};

export interface ContextApis {
    notifications: NotificationsApi;
    modal: ModalApi;
    navigation: NavigationApi;
    baseHref: string;
}
export const Context = React.createContext<ContextApis>(null);
export const {Provider, Consumer} = Context;

export const AuthSettingsCtx = React.createContext<models.AuthSettings>(null);

export interface AuthorizationState {
    user: models.UserInfo;
    isAdmin: boolean;
    canReadData: boolean;
    canWriteData: boolean;
    revision: number;
    refresh(): Promise<void>;
}

export const AuthorizationCtx = React.createContext<AuthorizationState>(null);

export const useAuthorization = (): AuthorizationState => {
    const authorization = React.useContext(AuthorizationCtx);
    if (!authorization) {
        throw new Error('Authorization context is unavailable');
    }
    return authorization;
};
