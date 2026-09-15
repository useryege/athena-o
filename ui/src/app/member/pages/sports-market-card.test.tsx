import * as React from 'react';
import {render, screen, cleanup} from '@testing-library/react';
import {SportsLiveEventCard} from './sports-market-card';
const item = {
    eventKey: 'event-full',
    eventId: 'event-id',
    slug: 'atp-test',
    title: 'A vs B',
    score: '6–0',
    period: 'Set 1',
    markets: [
        {
            marketKey: 'market-full',
            conditionId: 'condition-full',
            slug: 'atp-test',
            question: 'A vs B',
            sportsMarketType: 'moneyline',
            outcomes: '["A","B"]',
            outcomePrices: '["0.615","0"]'
        }
    ],
    teams: []
};
afterEach(cleanup);
test('missing business timestamps cannot become an index-based curve', () => {
    render(<SportsLiveEventCard item={item} history={[{marketKey: 'market-full', tokenId: 'token', outcome: 'A', prices: [0.4, 0.615], timestamps: []}]} />);
    expect(screen.queryByRole('img', {name: 'Moneyline price history'})).toBeNull();
    expect(screen.queryByText('No history')).not.toBeNull();
});
test('a missing outcome series cannot borrow another outcome curve', () => {
    const {container} = render(
        <SportsLiveEventCard item={item} history={[{marketKey: 'market-full', tokenId: 'token', outcome: 'A', prices: [0.4, 0.615], timestamps: [1789350000, 1789351200]}]} />
    );
    expect(container.querySelectorAll('.sports-live-chart__line')).toHaveLength(1);
    expect(screen.queryByText('0.615')).not.toBeNull();
    expect(screen.queryByText('6–0')).not.toBeNull();
});
test('time navigation is keyboard accessible and sources preserve identifiers', () => {
    render(<SportsLiveEventCard item={item} history={[{marketKey: 'market-full', tokenId: 'token', outcome: 'A', prices: [0.4, 0.615], timestamps: [1789350000, 1789351200]}]} />);
    expect(screen.queryByRole('slider', {name: 'History time'})).not.toBeNull();
    expect(screen.queryByText('event-full')).not.toBeNull();
});
