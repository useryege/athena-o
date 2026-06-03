import * as React from 'react';
import {act, create, ReactTestRenderer} from 'react-test-renderer';

const getProjectDiscoveryStatus = jest.fn();
const startProjectDiscovery = jest.fn();
const stopProjectDiscovery = jest.fn();

jest.mock('../../../shared/components', () => ({
    Page: ({children}: {children: React.ReactNode}) => <div>{children}</div>
}));

jest.mock('../../../shared/services', () => ({
    services: {
        athenaApplication: {
            getProjectDiscoveryStatus: (...args: any[]) => getProjectDiscoveryStatus(...args),
            startProjectDiscovery: (...args: any[]) => startProjectDiscovery(...args),
            stopProjectDiscovery: (...args: any[]) => stopProjectDiscovery(...args)
        }
    }
}));

import {ApplicationDiscovery} from './application-discovery';

const flush = async () => {
    await Promise.resolve();
    await Promise.resolve();
    act(() => {});
};

const promiseWithAbort = <T,>(value: Promise<T>) => {
    const promise = value as Promise<T> & {abort?: () => void};
    promise.abort = jest.fn();
    return promise;
};

const nodeText = (node: any): string => {
    if (typeof node === 'string') {
        return node;
    }
    if (!node?.children) {
        return '';
    }
    return node.children.map(nodeText).join('');
};

const buttonByText = (renderer: ReactTestRenderer, text: string) => renderer.root.findAllByType('button').find(button => nodeText(button) === text)!;

describe('application discovery settings', () => {
    let renderer: ReactTestRenderer | undefined;

    beforeEach(() => {
        renderer = undefined;
        getProjectDiscoveryStatus.mockReset();
        startProjectDiscovery.mockReset();
        stopProjectDiscovery.mockReset();
        getProjectDiscoveryStatus.mockReturnValue(promiseWithAbort(Promise.resolve({started: false, status: 'stopped'})));
        startProjectDiscovery.mockReturnValue(promiseWithAbort(Promise.resolve({started: true, status: 'running'})));
        stopProjectDiscovery.mockReturnValue(promiseWithAbort(Promise.resolve({started: false, status: 'stopped'})));
    });

    afterEach(() => {
        if (renderer) {
            act(() => {
                renderer!.unmount();
            });
        }
    });

    it('loads discovery status', async () => {
        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        expect(getProjectDiscoveryStatus).toHaveBeenCalledTimes(1);
        expect(nodeText(renderer!.root)).toContain('stopped');
    });

    it('disables buttons according to running state', async () => {
        getProjectDiscoveryStatus.mockReturnValue(promiseWithAbort(Promise.resolve({started: true, status: 'running'})));
        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        expect(buttonByText(renderer!, 'Start').props.disabled).toBe(true);
        expect(buttonByText(renderer!, 'Stop').props.disabled).toBe(false);

        act(() => {
            renderer!.unmount();
        });
        renderer = undefined;
        getProjectDiscoveryStatus.mockReturnValue(promiseWithAbort(Promise.resolve({started: false, status: 'stopped'})));

        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        expect(buttonByText(renderer!, 'Start').props.disabled).toBe(false);
        expect(buttonByText(renderer!, 'Stop').props.disabled).toBe(true);
    });

    it('starts discovery and refreshes status', async () => {
        getProjectDiscoveryStatus
            .mockReturnValueOnce(promiseWithAbort(Promise.resolve({started: false, status: 'stopped'})))
            .mockReturnValueOnce(promiseWithAbort(Promise.resolve({started: true, status: 'running'})));

        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        act(() => {
            buttonByText(renderer!, 'Start').props.onClick();
        });
        await flush();

        expect(startProjectDiscovery).toHaveBeenCalledTimes(1);
        expect(getProjectDiscoveryStatus).toHaveBeenCalledTimes(2);
        expect(nodeText(renderer!.root)).toContain('running');
    });

    it('stops discovery and refreshes status', async () => {
        getProjectDiscoveryStatus
            .mockReturnValueOnce(promiseWithAbort(Promise.resolve({started: true, status: 'running'})))
            .mockReturnValueOnce(promiseWithAbort(Promise.resolve({started: false, status: 'stopped'})));

        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        act(() => {
            buttonByText(renderer!, 'Stop').props.onClick();
        });
        await flush();

        expect(stopProjectDiscovery).toHaveBeenCalledTimes(1);
        expect(getProjectDiscoveryStatus).toHaveBeenCalledTimes(2);
        expect(nodeText(renderer!.root)).toContain('stopped');
    });

    it('shows request errors', async () => {
        getProjectDiscoveryStatus.mockReturnValue(promiseWithAbort(Promise.reject(new Error('status failed'))));
        act(() => {
            renderer = create(<ApplicationDiscovery />);
        });
        await flush();

        expect(nodeText(renderer!.root)).toContain('status failed');
    });
});
