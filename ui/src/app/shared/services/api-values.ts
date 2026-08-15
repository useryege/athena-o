export const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

export const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) || '');

export const readNumber = (item: any, ...names: string[]) => {
    const value = Number(readValue(item, ...names));
    return Number.isFinite(value) ? value : undefined;
};

export const readBoolean = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    return value === true || value === 'true' || value === 1 || value === '1';
};

export const readNumberArray = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (!Array.isArray(value)) {
        return [];
    }
    return value.map(next => Number(next)).filter(next => Number.isFinite(next));
};
