import renderer, {act} from 'react-test-renderer';
import {PnLChart} from './pnl-chart';
import {normalizeResolvedTarget} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/resolve-saved-note-existing.json';

const makeView = () => {
    const view = normalizeResolvedTarget(fixture.target).pnl.find(item => item.period === 'YTD')!;
    view.amount = {evidence: {...view.amount.evidence, availability: 'unavailable', reasonCode: 'reference_unavailable'}};
    view.curve = {
        evidence: {...view.curve.evidence, availability: 'available'},
        points: [
            {t: '1789088400', p: '-9007199254740993.000001'},
            {t: '1789092000', p: '9007199254740993.000002'}
        ]
    };
    return view;
};
test('YTD unavailable amount retains its available curve and accessible exact values', () => {
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(<PnLChart view={makeView()} />);
    });
    expect(tree!.root.findAllByType('svg').filter(item => item.props.viewBox === '0 0 640 240')).toHaveLength(1);
    const content = JSON.stringify(tree!.toJSON());
    expect(content).toContain('reference_unavailable');
    expect(content).toContain('-9007199254740993.000001');
    expect(content).toContain('9007199254740993.000002');
    expect(content).toContain('UTC+8');
    expect(tree!.root.findByType('polyline').props.points).not.toMatch(/NaN|Infinity/);
});
test('unavailable curve states its reason and never draws a zero line', () => {
    const view = makeView();
    view.curve.evidence = {...view.curve.evidence, availability: 'unavailable', reasonCode: 'query_failed'};
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(<PnLChart view={view} />);
    });
    expect(JSON.stringify(tree!.toJSON())).toContain('query_failed');
    expect(tree!.root.findAllByType('svg').filter(item => item.props.viewBox === '0 0 640 240')).toHaveLength(0);
});
test('huge close decimal values produce finite distinct chart coordinates', () => {
    const view = makeView();
    view.curve.points[0].p = '9'.repeat(400) + '.000001';
    view.curve.points[1].p = '9'.repeat(400) + '.000002';
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(<PnLChart view={view} />);
    });
    const points = tree!.root.findByType('polyline').props.points as string;
    expect(points).not.toMatch(/NaN|Infinity/);
    expect(points.split(' ')[0].split(',')[1]).not.toBe(points.split(' ')[1].split(',')[1]);
});
test('uses provider dollar display symbol without asserting a currency code', () => {
    const view = makeView();
    view.amount = {evidence: {...view.amount.evidence, availability: 'available'}, value: '0'};
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(<PnLChart view={view} />);
    });
    const content = JSON.stringify(tree!.toJSON());
    expect(content).toContain('$0');
    expect(content).toContain('$ follows Polymarket’s display; currency code is not provided.');
});
test('unavailable amount cannot expose a copy control for an untrusted retained value', () => {
    const view = makeView();
    view.amount.value = '123';
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(<PnLChart view={view} />);
    });
    expect(tree!.root.findAll(item => item.props.copyable?.text === '123')).toHaveLength(0);
});
