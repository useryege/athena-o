import {formatRaw, formatFillPrice} from './precision';

test.each([
    ['9007199254740993', 6, '9007199254.740993'],
    ['1', 6, '0.000001'],
    ['-1234000', 6, '-1.234'],
    ['0', 6, '0'],
    ['100', 0, '100'],
    ['10', 2, '0.1']
])('raw %s at %s decimals stays exact', (raw, decimals, expected) => {
    expect(formatRaw(raw as string, decimals as number)).toBe(expected);
});
test.each(['1.2', '', '1e3', ' 1', '+1'])('rejects invalid raw %s', raw => {
    expect(() => formatRaw(raw, 6)).toThrow();
});
test.each([-1, 1.5, Infinity, NaN])('rejects invalid precision %s', decimals => {
    expect(() => formatRaw('1', decimals)).toThrow();
    expect(() => formatFillPrice('1', '3', decimals)).toThrow();
});
test.each([
    ['1', '3', undefined, {text: '0.333333', approximate: true}],
    ['2', '3', 2, {text: '0.67', approximate: true}],
    ['1', '2', undefined, {text: '0.5', approximate: false}],
    ['0', '1', undefined, {text: '0', approximate: false}],
    ['1', '100000000000000000000', undefined, {text: '1/100000000000000000000', approximate: false}],
    ['-1', '3', 2, {text: '-0.33', approximate: true}],
    ['999', '1000', 2, {text: '1', approximate: true}],
    ['9007199254740993', '1', 0, {text: '9007199254740993', approximate: false}],
    ['1', '0', undefined, undefined],
    ['1', '-1', undefined, undefined]
])('price %s/%s at %s places reports precision', (n, d, p, expected) => {
    expect(formatFillPrice(n as string, d as string, p as number | undefined)).toEqual(expected);
});
