import {MarketRadarService} from './market-radar-service';
import requests from './requests';

jest.mock('./requests', () => ({__esModule: true, default: {get: jest.fn()}}));
const cases = [
    {method: 'listHotMarkets', counts: ['candidateCount', 'monitoredMarkets', 'monitoredTokens']},
    {method: 'listRealtimeMarkets', counts: ['candidateCount', 'subscribedMarkets', 'subscribedTokens']},
    {method: 'listMovers', counts: ['candidateCount', 'monitoredMarkets', 'monitoredTokens']}
] as const;
const response = (body: Record<string, unknown>) => {
    const req = Object.assign(Promise.resolve({body}), {abort: jest.fn()});
    (requests.get as jest.Mock).mockReturnValue({query: jest.fn(() => req)});
};
afterEach(() => jest.clearAllMocks());
for (const {method, counts} of cases) {
    test(`${method}: omitted encoding/json proto3 counters mean zero and omitted bools mean false`, async () => {
        response({});
        const result = await new MarketRadarService()[method]();
        expect(result).toMatchObject(Object.fromEntries(counts.map(key => [key, 0])));
        expect(result.stale).toBe(false);
        if ('connected' in result) expect(result.connected).toBe(false);
    });
    test(`${method}: explicit counters retain zero and nonzero; null and malformed stay unknown`, async () => {
        for (const value of [0, 17, null, 'invalid']) {
            response(Object.fromEntries(counts.map(key => [key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`), value])));
            const result = await new MarketRadarService()[method]();
            expect(result).toMatchObject(Object.fromEntries(counts.map(key => [key, typeof value === 'number' ? value : undefined])));
        }
    });
}
