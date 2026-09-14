import * as React from 'react';
import {render, screen, cleanup} from '@testing-library/react';
import {MarketVolume} from './market-radar-presentation';

afterEach(cleanup);
test('preserves a known zero volume', () => {
    render(<MarketVolume value={0} />);
    expect(screen.queryByText('0')).not.toBeNull();
});
test('distinguishes missing volume from zero', () => {
    render(<MarketVolume value={undefined} />);
    expect(screen.queryByText('Unavailable')).not.toBeNull();
    expect(screen.queryByText('0')).toBeNull();
});

import {HotMarketRecord, RealtimeMarketRecord, MoverMarketRecord} from './market-radar-presentation';
const item = {conditionId: 'condition-full', marketSlug: 'market', eventSlug: 'event', question: 'Contract values?', tokens: []};
test('hot records retain separate 24h and total volumes and full identifiers', () => {
    render(<HotMarketRecord item={{...item, volume24hr: 0, volumeNum: 123456.789, liquidityNum: undefined, spread: 0}} />);
    expect(screen.queryByText('Volume 24h')).not.toBeNull();
    expect(screen.queryByText('123,456.789')).not.toBeNull();
    expect(screen.queryByText('Unavailable')).not.toBeNull();
    expect(screen.queryByText('condition-full')).not.toBeNull();
});
test('realtime records distinguish a zero change, missing window and warmup', () => {
    render(
        <RealtimeMarketRecord
            item={{
                ...item,
                tokens: [
                    {
                        tokenId: 'full-token',
                        outcome: 'Yes',
                        price: 0,
                        windows: [
                            {window: '1m', priceChangePp: 0},
                            {window: '5m', warmup: true}
                        ]
                    }
                ]
            }}
        />
    );
    expect(screen.queryByText('0%')).not.toBeNull();
    expect(screen.queryByText('0 pp')).not.toBeNull();
    expect(screen.queryAllByText('Warmup').length).toBeGreaterThan(0);
    expect(screen.queryAllByText('Unavailable').length).toBeGreaterThan(0);
});
test('movers retain supplied market score, direction and leader instead of deriving them', () => {
    render(<MoverMarketRecord item={{...item, score: 0, direction: 'unclassified', leader: {tokenId: 'leader', outcome: 'No', score: 9, price: 0.125, windows: []}}} />);
    expect(screen.queryByText('unclassified')).not.toBeNull();
    expect(screen.queryByText('Mover score')).not.toBeNull();
    expect(screen.queryAllByText('No').length).toBeGreaterThan(0);
    expect(screen.queryByText('9')).toBeNull();
});
