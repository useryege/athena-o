import * as React from 'react';
import {act, create, ReactTestRenderer} from 'react-test-renderer';

const listProjects = jest.fn();
const replace = jest.fn();

jest.mock('argo-ui', () => ({
    MockupList: () => <div>Loading</div>,
    Page: ({children}: {children: React.ReactNode}) => <div>{children}</div>
}));

jest.mock('../../../app', () => ({
    history: {
        location: {
            pathname: '/projects',
            search: ''
        },
        replace: (...args: any[]) => replace(...args)
    }
}));

jest.mock('../../../shared/services', () => ({
    services: {
        athenaApplication: {
            listProjects: (...args: any[]) => listProjects(...args)
        }
    }
}));

jest.mock('../project-list-row/project-list-row', () => ({
    ProjectListRow: ({project}: {project: {contract?: string}}) => <div>{project.contract || 'project'}</div>
}));

import {history} from '../../../app';
import {ProjectsList} from './projects-list';

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

const successResponse = (page = 1) => ({
    items: [],
    total: 0,
    page,
    pageSize: 10
});

const nodeText = (node: any): string => {
    if (typeof node === 'string') {
        return node;
    }
    if (!node?.children) {
        return '';
    }
    return node.children.map(nodeText).join('');
};

const buttonTexts = (renderer: ReactTestRenderer) => renderer.root.findAllByType('button').map(button => nodeText(button));

describe('projects list polling', () => {
    let renderer: ReactTestRenderer | undefined;

    beforeEach(() => {
        jest.useFakeTimers();
        renderer = undefined;
        listProjects.mockReset();
        replace.mockReset();
        history.location.pathname = '/projects';
        history.location.search = '';
        listProjects.mockImplementation((page = 1) => promiseWithAbort(Promise.resolve(successResponse(page))));
    });

    afterEach(() => {
        if (renderer) {
            act(() => {
                renderer!.unmount();
            });
        }
        jest.clearAllTimers();
        jest.useRealTimers();
    });

    it('starts polling on mount and removes manual refresh controls', async () => {
        act(() => {
            renderer = create(<ProjectsList />);
        });
        await flush();

        expect(listProjects).toHaveBeenCalledTimes(1);
        expect(listProjects).toHaveBeenLastCalledWith(1, 10);
        expect(buttonTexts(renderer!)).toEqual(['Prev', 'Next']);
        expect(nodeText(renderer!.root)).toContain('Polling: Active');

        act(() => {
            jest.advanceTimersByTime(3000);
        });
        await flush();

        expect(listProjects).toHaveBeenCalledTimes(2);
        expect(listProjects).toHaveBeenLastCalledWith(1, 10);
    });

    it('stops polling after ten consecutive failures', async () => {
        listProjects.mockImplementation(() => promiseWithAbort(Promise.reject(new Error('backend unavailable'))));

        act(() => {
            renderer = create(<ProjectsList />);
        });
        await flush();

        for (let i = 1; i < 10; i++) {
            act(() => {
                jest.advanceTimersByTime(3000);
            });
            await flush();
        }

        expect(listProjects).toHaveBeenCalledTimes(10);
        expect(nodeText(renderer!.root)).toContain('Polling stopped after 10 failures');
        expect(nodeText(renderer!.root)).toContain('backend unavailable');

        act(() => {
            jest.advanceTimersByTime(3000);
        });
        await flush();

        expect(listProjects).toHaveBeenCalledTimes(10);
    });

    it('resets the consecutive failure count after a successful response', async () => {
        listProjects
            .mockImplementationOnce(() => promiseWithAbort(Promise.reject(new Error('first failure'))))
            .mockImplementationOnce(() => promiseWithAbort(Promise.resolve(successResponse(1))))
            .mockImplementation(() => promiseWithAbort(Promise.reject(new Error('later failure'))));

        act(() => {
            renderer = create(<ProjectsList />);
        });
        await flush();

        act(() => {
            jest.advanceTimersByTime(3000);
        });
        await flush();

        for (let i = 0; i < 9; i++) {
            act(() => {
                jest.advanceTimersByTime(3000);
            });
            await flush();
        }

        expect(listProjects).toHaveBeenCalledTimes(11);
        expect(nodeText(renderer!.root)).toContain('Polling: Active');
        expect(nodeText(renderer!.root)).not.toContain('Polling stopped after 10 failures');
    });

    it('aborts the active request on unmount', () => {
        let resolveRequest = (_value: ReturnType<typeof successResponse>) => {};
        const pendingRequest = promiseWithAbort(
            new Promise<ReturnType<typeof successResponse>>(resolve => {
                resolveRequest = resolve;
            })
        );
        listProjects.mockReturnValue(pendingRequest);

        act(() => {
            renderer = create(<ProjectsList />);
        });

        act(() => {
            renderer.unmount();
        });

        expect(pendingRequest.abort).toHaveBeenCalledTimes(1);
        resolveRequest(successResponse(1));
    });
});
