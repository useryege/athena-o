import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {MemoryRouter, Route, Routes} from 'react-router-dom';
import {AuthorizationCtx} from '../../shared/context';
import {parseUserInfo} from '../../shared/models';
import {beginAdminReadSession, endAdminReadSession} from '../read-scope';
import {Tag} from 'antd';
import {adminServices, ensureAdminBusinessServices} from '../services';
import {SystemNotificationDetailPage} from './system-notification-detail';

beforeAll(() => ensureAdminBusinessServices());

const user = parseUserInfo({accountId: 'admin-detail', iss: 'fixture', loggedIn: true, administrator: true});
beforeEach(() => {
    beginAdminReadSession(user);
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn()
    }));
});
afterEach(() => {
    endAdminReadSession();
    jest.restoreAllMocks();
});

test('unknown detail explains no automatic resend and preserves absent started time', async () => {
    jest.spyOn(adminServices.adminNotifications, 'getNotification').mockResolvedValue({
        id: 1,
        source: 'test',
        severity: 'info',
        topicLabel: 'ops',
        title: 'Test',
        body: 'Hello',
        link: '',
        channel: 'telegram',
        status: 'unknown',
        telegramChat: 'test',
        providerMessageId: '',
        errorMessage: 'response_lost',
        createdAt: '2026-09-10T01:00:00Z',
        sentAt: '',
        authorizedAt: '2026-09-10T01:00:01Z',
        startedAt: '',
        resultAt: '2026-09-10T01:00:06Z'
    });
    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter initialEntries={['/notifications/1']} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <AuthorizationCtx.Provider value={{user, isAdmin: true} as any}>
                    <Routes>
                        <Route path='/notifications/:id' element={<SystemNotificationDetailPage />} />
                    </Routes>
                </AuthorizationCtx.Provider>
            </MemoryRouter>
        );
    });
    const content = JSON.stringify(tree.toJSON());
    expect(content).toContain('It will not be resent automatically.');
    expect(content).toContain('HTTP started');
    expect(content).toContain('09:00:06');
    expect(tree.root.findAllByType(Tag).find(tag => tag.props.children === 'unknown')?.props.color).toBe('warning');
    act(() => tree.unmount());
});
