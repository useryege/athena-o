import * as React from 'react';
import {act, create, ReactTestRenderer} from 'react-test-renderer';

import {NotificationsList} from './notifications-list';

const listNotifications = jest.fn();
const sendTestNotification = jest.fn();

jest.mock('argo-ui', () => ({
    MockupList: () => <div>Loading</div>,
    Page: ({children}: {children: React.ReactNode}) => <div>{children}</div>
}));

jest.mock('../../shared/services', () => ({
    services: {
        notification: {
            listNotifications: (...args: any[]) => listNotifications(...args),
            sendTestNotification: (...args: any[]) => sendTestNotification(...args)
        }
    }
}));

const flush = () => new Promise(resolve => setTimeout(resolve, 0));

const promiseWithAbort = <T,>(value: Promise<T>) => {
    const promise = value as Promise<T> & {abort?: () => void};
    promise.abort = jest.fn();
    return promise;
};

const routeProps = () =>
    ({
        history: {push: jest.fn()},
        location: {search: ''},
        match: {params: {}, path: '/notifications', url: '/notifications'},
        staticContext: undefined
    }) as any;

const nodeText = (node: any): string => {
    if (typeof node === 'string') {
        return node;
    }
    if (!node?.children) {
        return '';
    }
    return node.children.map(nodeText).join('');
};

const findButton = (renderer: ReactTestRenderer, text: string) =>
    renderer.root.findAllByType('button').find(button => nodeText(button).includes(text));

describe('notifications list test sender', () => {
    beforeEach(() => {
        listNotifications.mockReset();
        sendTestNotification.mockReset();
        listNotifications.mockReturnValue(
            promiseWithAbort(
                Promise.resolve({
                    items: [],
                    total: 0,
                    page: 1,
                    pageSize: 20
                })
            )
        );
    });

    it('sends TOKEN and POLY test notifications and refreshes the list', async () => {
        sendTestNotification.mockReturnValue(
            promiseWithAbort(
                Promise.resolve({
                    notificationId: 11,
                    status: 'sent',
                    providerMessageId: '123',
                    errorMessage: ''
                })
            )
        );

        let renderer = null as unknown as ReactTestRenderer;
        act(() => {
            renderer = create(<NotificationsList {...routeProps()} />);
        });
        await flush();
        await flush();

        act(() => {
            findButton(renderer, 'Send TOKEN Test').props.onClick();
        });
        await flush();
        await flush();

        expect(sendTestNotification).toHaveBeenCalledWith('token');
        expect(listNotifications).toHaveBeenCalledTimes(2);
        expect(nodeText(renderer!.root)).toContain('TOKEN test notification sent (sent).');

        act(() => {
            findButton(renderer, 'Send POLY Test').props.onClick();
        });
        await flush();
        await flush();

        expect(sendTestNotification).toHaveBeenCalledWith('poly');
        expect(sendTestNotification).toHaveBeenCalledTimes(2);
        expect(listNotifications).toHaveBeenCalledTimes(3);
        expect(nodeText(renderer!.root)).toContain('POLY test notification sent (sent).');
    });

    it('shows send errors and refreshes the list', async () => {
        sendTestNotification.mockImplementation(() => promiseWithAbort(Promise.reject(new Error('telegram unavailable'))));

        let renderer = null as unknown as ReactTestRenderer;
        act(() => {
            renderer = create(<NotificationsList {...routeProps()} />);
        });
        await flush();
        await flush();

        act(() => {
            findButton(renderer, 'Send TOKEN Test').props.onClick();
        });
        await flush();
        await flush();

        expect(sendTestNotification).toHaveBeenCalledWith('token');
        expect(sendTestNotification).toHaveBeenCalledTimes(1);
        expect(listNotifications).toHaveBeenCalledTimes(2);
        expect(nodeText(renderer!.root)).toContain('telegram unavailable');
    });
});
