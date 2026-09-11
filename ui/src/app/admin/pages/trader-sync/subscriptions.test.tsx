import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {MemoryRouter, Routes, Route, Link} from 'react-router-dom';
import {Input, Checkbox, Select, Button} from 'antd';
import {AuthorizationCtx, type AuthorizationState} from '../../../shared/context';
import {parseUserInfo} from '../../../shared/models';
import {adminServices as services, ensureAdminBusinessServices} from '../../services';
import {beginAdminReadSession, endAdminReadSession} from '../../read-scope';
import {normalizeSubscriptionSummary} from '../../trader-sync-models';
import {TraderSyncAdminSubscriptionsPage} from './subscriptions';
import {TraderSyncAdminSubscriptionPage} from './subscription-detail';
ensureAdminBusinessServices();
const user = parseUserInfo({accountId: 'administrator', iss: 'issuer', loggedIn: true, administrator: true});
const summary = normalizeSubscriptionSummary({
    subscriptionId: 'sub-2',
    accountId: 'account-1',
    username: 'alice',
    email: 'alice@example.test',
    wallet: '0x1234567890123456789012345678901234567890',
    status: 'healthy',
    createdAt: '2026-09-10T01:00:00Z',
    updatedAt: '2026-09-10T02:00:00Z',
    observation: {state: 'healthy', reason: '', interruptionCount: '2', latestInterruption: {reason: 'disconnect', uncertainty: 'unknown_start', possibleMissing: true}},
    activityCount: '9007199254740993',
    associatedDeliveryCounts: {total: '5', pending: '1', sending: '0', sent: '2', failed: '1', unknown: '1', cancelled: '0'},
    asOf: '2026-09-11T01:00:00Z'
});
let tree: renderer.ReactTestRenderer;
const text = () => JSON.stringify(tree.toJSON());
const mount = async (detail = false) => {
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter
                initialEntries={[detail ? '/trader-sync/subscriptions/sub-2' : '/trader-sync/subscriptions']}
                future={{v7_startTransition: true, v7_relativeSplatPath: true}}
            >
                <AuthorizationCtx.Provider value={{user, isAdmin: true} as AuthorizationState}>
                    <Routes>
                        <Route path='/trader-sync/subscriptions' element={<TraderSyncAdminSubscriptionsPage />} />
                        <Route path='/trader-sync/subscriptions/:id' element={<TraderSyncAdminSubscriptionPage />} />
                    </Routes>
                </AuthorizationCtx.Provider>
            </MemoryRouter>
        );
    });
};
beforeEach(() => {
    jest.useFakeTimers();
    window.matchMedia = jest
        .fn()
        .mockImplementation(query => ({
            matches: false,
            media: query,
            addListener: jest.fn(),
            removeListener: jest.fn(),
            addEventListener: jest.fn(),
            removeEventListener: jest.fn()
        }));
    beginAdminReadSession(user);
    jest.spyOn(services.adminTraderSync, 'listSubscriptionSummaries').mockResolvedValue({
        summaries: [summary, {...summary, subscriptionId: 'sub-1'}],
        page: {nextCursor: 'page-2'},
        asOf: summary.asOf
    });
    jest.spyOn(services.adminTraderSync, 'getSubscriptionSummary').mockResolvedValue(summary);
});
afterEach(() => {
    act(() => {
        tree?.unmount();
        endAdminReadSession();
    });
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('summary list renders full identities, exact lifetime counts, public labels and safe admin links', async () => {
    await mount();
    expect(text()).toContain(summary.wallet);
    expect(text()).toContain('alice@example.test');
    expect(text()).toContain('9007199254740993');
    expect(text()).toContain('Monitoring');
    expect(text()).toContain('Associated deliveries');
    expect(text()).toContain('Do not add');
    const links = tree.root.findAllByType(Link).map(link => link.props.to);
    expect(links).toContain('/trader-sync/subscriptions/sub-2');
    expect(links.every(link => !String(link).includes('/activities') && !String(link).includes('/notifications/'))).toBe(true);
    expect(services.adminTraderSync.listSubscriptionSummaries).toHaveBeenCalledWith(expect.objectContaining({includeCancelled: false}));
    expect(
        tree.root
            .findAllByType(Button)
            .map(button => String(button.props.children))
            .join(' ')
    ).not.toMatch(/Pause|Resume|Cancel subscription|Resend/);
});
test('next/previous retain input cursors; applying account wallet and exact state filters resets cursor and includes all cancellations', async () => {
    await mount();
    expect(services.adminTraderSync.listSubscriptionSummaries).toHaveBeenCalledWith(expect.objectContaining({includeCancelled: false}));
    const button = (label: string) => tree.root.findAllByType(Button).find(button => button.props.children === label)!;
    await act(async () => button('Next').props.onClick());
    expect(services.adminTraderSync.listSubscriptionSummaries).toHaveBeenLastCalledWith(expect.objectContaining({cursor: 'page-2'}));
    await act(async () => button('Previous').props.onClick());
    expect(services.adminTraderSync.listSubscriptionSummaries).toHaveBeenLastCalledWith(expect.objectContaining({cursor: undefined}));
    await act(async () => button('Next').props.onClick());
    act(() => {
        tree.root
            .findAllByType(Input)
            .find(input => input.props['aria-label'] === 'Account ID')!
            .props.onChange({target: {value: 'filtered-owner'}});
        tree.root
            .findAllByType(Input)
            .find(input => input.props['aria-label'] === 'Canonical wallet')!
            .props.onChange({target: {value: summary.wallet}});
        tree.root.findByType(Select).props.onChange('interrupted');
        tree.root.findByType(Checkbox).props.onChange({target: {checked: true}});
    });
    await act(async () => tree.root.findByType('form').props.onSubmit({preventDefault: () => {}}));
    expect(services.adminTraderSync.listSubscriptionSummaries).toHaveBeenLastCalledWith({
        pageSize: 50,
        cursor: undefined,
        accountId: 'filtered-owner',
        wallet: summary.wallet,
        state: 'interrupted',
        includeCancelled: true
    });
});
test('detail shows current lifecycle facts and unknown interruption endpoints without implying ongoing failure', async () => {
    await mount(true);
    expect(text()).toContain('Last reliable checkpoint');
    expect(text()).toContain('Unknown');
    expect(text()).toContain('disconnect');
    expect(text()).toContain('Monitoring');
    expect(text()).toContain('Service Status');
    expect(text()).not.toContain('PRIVATE');
    expect(services.adminTraderSync.getSubscriptionSummary).toHaveBeenCalledWith('sub-2');
    act(() => endAdminReadSession());
    expect(text()).not.toContain(summary.wallet);
});
