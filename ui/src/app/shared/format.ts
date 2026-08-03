const BEIJING_TIME_ZONE = 'Asia/Shanghai';

const beijingDateTimeFormatter = new Intl.DateTimeFormat('en-US-u-nu-latn', {
    timeZone: BEIJING_TIME_ZONE,
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23'
});

const beijingDateTimeText = (value: Date) => {
    if (Number.isNaN(value.getTime())) {
        return undefined;
    }
    const parts = new Map(beijingDateTimeFormatter.formatToParts(value).map(part => [part.type, part.value]));
    const year = parts.get('year');
    const month = parts.get('month');
    const day = parts.get('day');
    const hour = parts.get('hour');
    const minute = parts.get('minute');
    const second = parts.get('second');
    if (!year || !month || !day || !hour || !minute || !second) {
        return undefined;
    }
    return `${year}/${Number(month)}/${Number(day)} ${hour.padStart(2, '0')}:${minute.padStart(2, '0')}:${second.padStart(2, '0')}`;
};

export const formatBlockNumber = (value?: string | number | null) => {
    if (value === undefined || value === null || value === '') {
        return '-';
    }
    return String(value);
};

export const formatBeijingDateTime = (value?: string | Date | null) => {
    if (value === undefined || value === null || value === '') {
        return undefined;
    }
    const parsed = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return typeof value === 'string' ? value : undefined;
    }
    return beijingDateTimeText(parsed);
};

export const formatBeijingUnixSeconds = (value?: number | null) => {
    if (value === undefined || value === null || value === 0 || !Number.isFinite(value)) {
        return undefined;
    }
    return beijingDateTimeText(new Date(value * 1000));
};
