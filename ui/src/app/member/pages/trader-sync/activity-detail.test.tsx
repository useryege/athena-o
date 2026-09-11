import renderer, {act} from 'react-test-renderer';
import {MemoryRouter, Route, Routes} from 'react-router-dom';
import {deliveryLabel, NotificationResult} from './notification-result';
import {TradeFacts} from './trade-facts';
import {ComboConditions} from './combo-conditions';
import {TraderSyncActivityPage} from './activity-detail';
import {clearTraderSyncState} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeActivity, Activity, Delivery, PartPage} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/05.json';
const activity = normalizeActivity(fixture.activity);
const sent: Delivery = {id: '1', status: 'sent', reason: '', authorizedAt: '2026-09-10T10:00:00Z', resultAt: '2026-09-10T10:00:02Z', attemptCount: '1'};
const counts = {total: '2', pending: '0', sending: '0', sent: '1', failed: '0', unknown: '1', cancelled: '0'};
const parts: PartPage = {
    parts: [
        {id: '1', index: 1, total: 3, delivery: sent, associatedActivityCount: '1'},
        {id: '2', index: 3, total: 3, delivery: {...sent, id: '2', status: 'unknown'}, associatedActivityCount: '2'}
    ],
    page: {nextCursor: 'part-next'},
    asOf: activity.recordedAt
};
let tree: renderer.ReactTestRenderer;
const text = () => JSON.stringify(tree.toJSON());
const render = async (node: React.ReactNode) =>
    act(async () => {
        tree = renderer.create(
            <MemoryRouter initialEntries={['/trader-sync/activities/12']} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                {node}
            </MemoryRouter>
        );
    });
const mount = () =>
    render(
        <Routes>
            <Route path='/trader-sync/activities/:activityId' element={<TraderSyncActivityPage ownerId='A' />} />
        </Routes>
    );
beforeEach(() => {
    ensureMemberBusinessServices();
    jest.useFakeTimers();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('sent remains accepted with no submit time and no read or non-call inference', async () => {
    expect(deliveryLabel(sent)).toBe('Telegram accepted');
    await render(<NotificationResult delivery={sent} />);
    expect(text()).toContain('Submit time not recorded');
    expect(text()).not.toMatch(/Not called|Read by|Unread/);
});
test('NO complements the whole conjunction and unknown legs are never zero', async () => {
    await render(
        <ComboConditions
            metadata={{
                ...activity.metadata,
                relationship: 'NOT(AND(legs))',
                legs: [],
                legsEvidence: {...activity.metadata.legsEvidence, availability: 'unavailable', reasonCode: 'rpc_timeout'}
            }}
        />
    );
    expect(text()).toContain('Not all conditions met');
    expect(text()).toContain('count unknown');
    expect(text()).not.toContain('0 conditions');
});
test('facts retain zero fee, precise price, full source, unavailable public time and UTC8 midnight', async () => {
    await render(
        <TradeFacts
            activity={{
                ...activity,
                settledAt: '2026-09-10T16:00:00Z',
                feeRaw: '0',
                priceNumerator: '1',
                priceDenominator: '3',
                priceEvidence: {...activity.priceEvidence, availability: 'available'}
            }}
        />
    );
    expect(text()).toContain('11/09/2026, 00:00:00 UTC+8');
    expect(text()).toContain('0.333333');
    expect(text()).toContain('Rounded');
    expect(text()).toContain('Public time unavailable');
    expect(text()).toContain(activity.sourceLocation.transactionHash);
    expect(text()).toContain('Fee');
    expect(text()).toContain(activity.wallet);
    expect(text()).toContain(activity.positionId);
});
test('unavailable metadata ignores residual title and URL; finality conflict remains independent', async () => {
    await render(
        <TradeFacts
            activity={{
                ...activity,
                finalityAnomaly: {reason: 'chain conflict', detectedAt: activity.recordedAt, publishedBlockHash: 'original'},
                metadata: {
                    ...activity.metadata,
                    market: {
                        ...activity.metadata.market,
                        evidence: {...activity.metadata.market.evidence, availability: 'unavailable'},
                        title: 'Residual title',
                        url: 'https://invalid.example'
                    }
                }
            }}
        />
    );
    expect(text()).not.toContain('Residual title');
    expect(tree.root.findAllByType('a').some(a => a.props.href === 'https://invalid.example')).toBe(false);
    expect(text()).toContain('Finality anomaly');
    expect(text()).toContain('Conflicting block hash unknown');
});
test('frozen detail shows each related part and refreshes the current part page once per tick', async () => {
    jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue({
        ...activity,
        id: '12',
        notificationMode: 'summary',
        summaryProgress: {
            phase: 'frozen',
            reason: '',
            batchId: '7',
            relatedPartCounts: counts,
            batchPartCounts: {...counts, total: '3', pending: '1'},
            oldestAt: activity.recordedAt
        }
    });
    const list = jest.spyOn(services.traderSync, 'listSummaryParts').mockResolvedValue(parts);
    await mount();
    expect(text()).toContain('Telegram accepted');
    expect(text()).toContain('Delivery unknown');
    expect(list).toHaveBeenCalledWith('7', expect.objectContaining({activityId: '12'}));
    list.mockResolvedValue({...parts, parts: [{...parts.parts[1], index: 3}], page: {}});
    await act(async () =>
        tree.root
            .findAllByType('button')
            .find(n => n.props.children === 'Next')!
            .props.onClick()
    );
    expect(list.mock.calls.at(-1)![1]?.cursor).toBe('part-next');
    expect(text()).toContain('Part ');
    const before = list.mock.calls.length;
    await act(async () => jest.advanceTimersByTime(5000));
    expect(list).toHaveBeenCalledTimes(before + 1);
    expect(list.mock.calls.at(-1)![1]?.cursor).toBe('part-next');
});
test.each(['waiting', 'cancelled_before_freeze'] as const)('%s has no batch link or part request', async phase => {
    jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue({
        ...activity,
        notificationMode: 'summary',
        summaryProgress: {phase, reason: 'binding changed', batchId: '7', relatedPartCounts: counts, batchPartCounts: counts, oldestAt: activity.recordedAt}
    });
    const list = jest.spyOn(services.traderSync, 'listSummaryParts').mockResolvedValue(parts);
    await mount();
    expect(tree.root.findAllByType('a').some(a => a.props.href.includes('/summaries/'))).toBe(false);
    expect(list).not.toHaveBeenCalled();
});
test('invalidating scope clears detail and fences delayed responses', async () => {
    let finish!: (a: Activity) => void;
    jest.spyOn(services.traderSync, 'getActivity').mockImplementation(
        () =>
            new Promise(resolve => {
                finish = resolve;
            })
    );
    await mount();
    act(() => clearTraderSyncState());
    await act(async () => finish(activity));
    expect(text()).not.toContain(activity.wallet);
});

test('raw evidence includes both decimal scales and missing submit time makes duration indeterminate', async () => {
    await render(
        <>
            <TradeFacts activity={activity} />
            <NotificationResult delivery={sent} />
        </>
    );
    expect(text()).toContain('Collateral decimals');
    expect(text()).toContain('Shares decimals');
    expect(text()).toContain('Fill price (value ÷ shares)');
    expect(text()).toContain('Submission-to-result duration cannot be determined');
});
test('long Combo preserves every leg including unavailable metadata and verified links stop propagation', async () => {
    const market = {
        ...activity.metadata.market,
        evidence: {...activity.metadata.market.evidence, availability: 'available' as const},
        title: 'Known condition',
        url: 'https://polymarket.com/event/verified'
    };
    const legs = Array.from({length: 101}, (_, i) => ({
        positionId: String(i),
        market: i === 100 ? {...market, evidence: {...market.evidence, availability: 'unavailable' as const, reasonCode: 'missing'}, title: 'Residual leg'} : market
    }));
    await render(
        <ComboConditions
            metadata={{...activity.metadata, relationship: 'AND(legs)', legsEvidence: {...activity.metadata.legsEvidence, availability: 'available', reasonCode: ''}, legs}}
        />
    );
    expect(tree.root.findAllByType('li')).toHaveLength(101);
    expect(text()).toContain('101 conditions');
    expect(text()).not.toContain('Residual leg');
    const stopPropagation = jest.fn();
    tree.root.findAllByType('a')[0].props.onClick({stopPropagation});
    expect(stopPropagation).toHaveBeenCalledTimes(1);
});
test('activity open callback identifies only the opened activity and preserves the shared save callback', async () => {
    const {ActivityFeed} = await import('./activity-feed');
    const onOpen = jest.fn();
    const onOpenActivity = jest.fn();
    const Feed = ActivityFeed as React.ComponentType<any>;
    await render(
        <Feed
            activities={[
                {...activity, id: '12'},
                {...activity, id: '11'}
            ]}
            onOpen={onOpen}
            onOpenActivity={onOpenActivity}
        />
    );
    const links = tree.root.findAllByType('a');
    links.find(link => link.props.href === '/trader-sync/activities/11')!.props.onClick({defaultPrevented: true});
    expect(onOpenActivity).toHaveBeenCalledWith('11');
    expect(onOpen).toHaveBeenCalledTimes(1);
    onOpenActivity.mockClear();
    links.find(link => link.props.href.includes('/subscriptions/'))!.props.onClick({defaultPrevented: true});
    expect(onOpenActivity).not.toHaveBeenCalled();
    expect(onOpen).toHaveBeenCalledTimes(2);
});
test('not-found after a prior success hides the trade and uses the same unavailable message', async () => {
    const get = jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue(activity);
    await mount();
    expect(text()).toContain(activity.wallet);
    get.mockRejectedValue({response: {body: {code: 5, message: 'private diagnostic'}}});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(text()).toContain('Activity not found');
    expect(text()).not.toContain(activity.wallet);
    expect(text()).not.toContain('private diagnostic');
});

test.each([
    ['pending', 'Queued'],
    ['sending', 'Sending'],
    ['sent', 'Telegram accepted'],
    ['failed', 'Delivery failed'],
    ['unknown', 'Delivery unknown'],
    ['cancelled', 'Delivery cancelled']
] as const)('delivery %s keeps its own outcome', async (status, label) => {
    await render(
        <NotificationResult
            delivery={{...sent, status, reason: 'original reason', latestAttempt: {index: '3', authorizedAt: sent.authorizedAt!, status: 'retryable', reason: 'temporary refusal'}}}
        />
    );
    expect(text()).toContain(label);
    expect(text()).toContain('Retryable response');
    expect(text()).toContain('original reason');
    expect(tree.root.findAllByType('button').some(button => /resend|retry delivery|all attempts/i.test(String(button.props.children)))).toBe(false);
});
test('non-positive price denominator stays unavailable and stale evidence never becomes a price', async () => {
    await render(<TradeFacts activity={{...activity, priceNumerator: '123', priceDenominator: '0', priceEvidence: {...activity.priceEvidence, availability: 'available'}}} />);
    expect(text()).toContain('denominator unavailable');
    await render(
        <TradeFacts
            activity={{
                ...activity,
                priceNumerator: '123',
                priceDenominator: '1',
                priceEvidence: {...activity.priceEvidence, availability: 'unavailable', reasonCode: 'not observed'}
            }}
        />
    );
    expect(text()).toContain('Unavailable: not observed');
    expect(text()).not.toContain('per share');
});

test('selected metadata stays stable until selection clears while finality evidence updates', async () => {
    const selectedNode = document.createElement('div');
    const selection = {isCollapsed: false, rangeCount: 1, getRangeAt: () => ({intersectsNode: (node: Node) => node === selectedNode})} as unknown as Selection;
    jest.spyOn(window, 'getSelection').mockReturnValue(selection);
    const node = (item: Activity) => (
        <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
            <TradeFacts activity={item} />
        </MemoryRouter>
    );
    await act(async () => {
        tree = renderer.create(node(activity), {createNodeMock: element => (element.type === 'div' ? selectedNode : null)});
    });
    const next = {
        ...activity,
        metadata: {
            ...activity.metadata,
            market: {...activity.metadata.market, evidence: {...activity.metadata.market.evidence, availability: 'available' as const}, title: 'Changed selected title'}
        },
        finalityAnomaly: {reason: 'conflict', detectedAt: activity.recordedAt, publishedBlockHash: 'original'}
    };
    await act(async () => tree.update(node(next)));
    expect(text()).not.toContain('Changed selected title');
    expect(text()).toContain('Finality anomaly');
    (selection as any).isCollapsed = true;
    act(() => document.dispatchEvent(new Event('selectionchange')));
    expect(text()).toContain('Changed selected title');
});
