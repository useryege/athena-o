const integer = (value: string): bigint => {
    if (!/^-?\d+$/.test(value)) {
        throw new Error('Expected an integer string');
    }
    return BigInt(value);
};

const checkPlaces = (places: number) => {
    if (!Number.isSafeInteger(places) || places < 0) {
        throw new Error('Expected non-negative integer decimal places');
    }
};

export const formatRaw = (raw: string, decimals: number): string => {
    checkPlaces(decimals);
    const value = integer(raw);
    const sign = value < 0n ? '-' : '';
    const digits = (value < 0n ? -value : value).toString().padStart(decimals + 1, '0');
    if (decimals === 0) {
        return sign + digits;
    }
    const fraction = digits.slice(-decimals).replace(/0+$/, '');
    return sign + digits.slice(0, -decimals) + (fraction ? '.' + fraction : '');
};

/** Round half away from zero; ratios smaller than the display unit stay exact. */
export const formatFillPrice = (numerator: string, denominator: string, places = 6): {text: string; approximate: boolean} | undefined => {
    checkPlaces(places);
    const n = integer(numerator);
    const d = integer(denominator);
    if (d <= 0n) {
        return undefined;
    }
    const magnitude = n < 0n ? -n : n;
    const scaled = magnitude * 10n ** BigInt(places);
    const quotient = scaled / d;
    const remainder = scaled % d;
    if (magnitude !== 0n && quotient === 0n) {
        return {text: `${n}/${d}`, approximate: false};
    }
    const rounded = quotient + (remainder * 2n >= d ? 1n : 0n);
    return {text: formatRaw((n < 0n ? -rounded : rounded).toString(), places), approximate: remainder !== 0n};
};
