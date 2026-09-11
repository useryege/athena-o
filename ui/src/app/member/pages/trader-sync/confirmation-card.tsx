import * as React from 'react';
import {Avatar, Button, Input, Space, Tag, Typography} from 'antd';
import {Section} from '../../../components';
import type {ResolvedTarget, PnLPeriod, StringField, DecimalField, TimeField} from '../../trader-sync-models';
import {formatBeijingDateTime} from '../../../shared/format';
import {evidenceReason, PnLChart, providerMoney} from './pnl-chart';

const displayField = (field: StringField | DecimalField | TimeField, time = false) =>
    field.evidence.availability === 'available' && field.value !== undefined
        ? time
            ? formatBeijingDateTime(field.value) + ' UTC+8'
            : field.value
        : evidenceReason(field.evidence);
export const ConfirmationCard = ({target, note, onNoteChange}: {target: ResolvedTarget; note: string; onNoteChange: (value: string) => void}) => {
    const [period, setPeriod] = React.useState<PnLPeriod>('1Y');
    const count = Array.from(note).length;
    const title = note.trim() || (target.displayName.evidence.availability === 'available' ? target.displayName.value : '') || target.wallet;
    return (
        <div className='trader-sync-confirmation'>
            <Section title='Review trader'>
                <Space align='start'>
                    {target.avatar.evidence.availability === 'available' && target.avatar.value && <Avatar size={48} src={target.avatar.value} />}
                    <div className='trader-sync-confirmation__identity'>
                        <Typography.Title level={3}>{title}</Typography.Title>
                        <Typography.Paragraph>Public name: {displayField(target.displayName)}</Typography.Paragraph>
                        {target.verified.evidence.availability === 'available' && target.verified.value !== undefined ? (
                            <Tag>{target.verified.value ? 'Verified' : 'Not verified'}</Tag>
                        ) : (
                            <Typography.Text>{evidenceReason(target.verified.evidence)}</Typography.Text>
                        )}
                    </div>
                </Space>
                <Typography.Paragraph className='trader-sync-confirmation__wallet' copyable={{text: target.wallet}}>
                    {target.wallet}
                </Typography.Paragraph>
                {target.canonicalProfileURL && (
                    <Typography.Link href={target.canonicalProfileURL} target='_blank' rel='noopener noreferrer'>
                        View Polymarket profile
                    </Typography.Link>
                )}
                <Typography.Paragraph type='secondary'>Profile queried: {formatBeijingDateTime(target.displayName.evidence.queriedAt) || 'Unavailable'} UTC+8</Typography.Paragraph>
                {target.avatar.evidence.availability === 'unavailable' && (
                    <Typography.Paragraph type='secondary'>Avatar: {evidenceReason(target.avatar.evidence)}</Typography.Paragraph>
                )}
                <dl className='trader-sync-confirmation__stats'>
                    {[
                        ['Joined', target.joinedAt, true],
                        ['Position value', target.positionValue, false],
                        ['Largest win', target.largestWin, false],
                        ['Predictions', target.predictions, false]
                    ].map(([label, field, time]) => (
                        <div key={label as string}>
                            <dt>{label as string}</dt>
                            <dd>
                                <Typography.Text
                                    copyable={
                                        (field as StringField).evidence.availability === 'available' && (field as StringField).value !== undefined
                                            ? {text: (field as StringField).value}
                                            : false
                                    }>
                                    {['Position value', 'Largest win'].includes(label as string) &&
                                    (field as DecimalField).evidence.availability === 'available' &&
                                    (field as DecimalField).value !== undefined
                                        ? providerMoney((field as DecimalField).value!)
                                        : displayField(field as StringField, time as boolean)}
                                </Typography.Text>
                            </dd>
                        </div>
                    ))}
                </dl>
                <Typography.Paragraph type='secondary'>$ follows Polymarket’s display; currency code is not provided.</Typography.Paragraph>
                <label htmlFor='trader-sync-note'>Private note</label>
                <Input
                    id='trader-sync-note'
                    value={note}
                    onChange={event => onNoteChange(event.target.value)}
                    status={count > 20 ? 'error' : undefined}
                    aria-invalid={count > 20}
                    aria-describedby='trader-sync-note-count'
                />
                <Typography.Paragraph id='trader-sync-note-count' type={count > 20 ? 'danger' : 'secondary'}>
                    {count} / 20 characters{count > 20 ? ' — Shorten your note to confirm.' : ''}
                </Typography.Paragraph>
            </Section>
            <Section title='Profit and loss'>
                <Space wrap={true} role='group' aria-label='P/L period'>
                    {(['1D', '1W', '1M', '1Y', 'YTD', 'ALL'] as PnLPeriod[]).map(value => (
                        <Button key={value} aria-pressed={period === value} type={period === value ? 'primary' : 'default'} onClick={() => setPeriod(value)}>
                            {value}
                        </Button>
                    ))}
                </Space>
                <PnLChart view={target.pnl.find(view => view.period === period)!} />
            </Section>
        </div>
    );
};
