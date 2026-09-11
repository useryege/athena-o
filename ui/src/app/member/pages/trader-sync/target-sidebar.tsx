import * as React from 'react';
import {Link} from 'react-router-dom';
import type {SubscriptionPage} from '../../trader-sync-models';
import {subscriptionStatusLabel, subscriptionTime} from './subscription-state';
import {consumeNewSubscriptionFocus, readNewSubscriptionFocus} from './state';
export const TargetSidebar = ({
    ownerId,
    page,
    selected,
    onSelect,
    disabled
}: {
    ownerId: string;
    page?: SubscriptionPage;
    selected?: string;
    onSelect: (id?: string) => void;
    disabled: boolean;
}) => {
    const [open, setOpen] = React.useState(false);
    const [highlight, setHighlight] = React.useState<string>();
    const buttons = React.useRef(new Map<string, HTMLButtonElement>());
    const targets = page?.subscriptions || [];
    const chosen = targets.find(item => item.id === selected);
    const label = (item: typeof chosen) =>
        item
            ? item.note.trim() || (item.targetDisplay.displayName.evidence.availability === 'available' ? item.targetDisplay.displayName.value : undefined) || item.wallet
            : selected
              ? 'Historical target'
              : 'All targets';
    const anomalies = targets.filter(item => item.status === 'interrupted' || item.status === 'permission_disabled').length;
    React.useEffect(() => {
        const id = readNewSubscriptionFocus(ownerId);
        if (!disabled && id && targets.some(item => item.id === id) && consumeNewSubscriptionFocus(ownerId, id)) {
            setOpen(true);
            setHighlight(id);
        }
    }, [ownerId, page, disabled]);
    React.useEffect(() => {
        if (highlight) buttons.current.get(highlight)?.focus();
    }, [highlight]);
    return (
        <aside className='trader-sync-targets' aria-label='Trader targets'>
            <button type='button' className='trader-sync-target-toggle' aria-expanded={open} aria-controls='trader-sync-target-list' onClick={() => setOpen(!open)}>
                Targets: {label(chosen)} · {page ? `${page.quota.used} / ${page.quota.limit}` : 'Quota unavailable'} · {page ? `${anomalies} alerts` : 'Status unavailable'}
            </button>
            <div id='trader-sync-target-list' className={open ? 'trader-sync-target-list is-open' : 'trader-sync-target-list'}>
                <h2>Targets</h2>
                <p>{page ? `${page.quota.used} / ${page.quota.limit} current subscriptions` : 'Loading targets…'}</p>
                <button type='button' aria-pressed={!selected} disabled={disabled} onClick={() => onSelect()}>
                    All activity
                </button>
                {targets.map(item => (
                    <div key={item.id} className={highlight === item.id ? 'trader-sync-target is-highlighted' : 'trader-sync-target'}>
                        <button
                            type='button'
                            ref={node => {
                                if (node) buttons.current.set(item.id, node);
                                else buttons.current.delete(item.id);
                            }}
                            aria-pressed={selected === item.id}
                            disabled={disabled}
                            onClick={() => onSelect(item.id)}>
                            {label(item)}
                        </button>
                        {item.note.trim() && item.targetDisplay.displayName.evidence.availability === 'available' && <p>{item.targetDisplay.displayName.value}</p>}
                        <p className='trader-sync-wallet'>{item.wallet}</p>
                        <p>{subscriptionStatusLabel(item.status)}</p>
                        <p>Effective: {subscriptionTime(item.currentInterval?.effectiveAt)}</p>
                        <p>Last reliable observation: {subscriptionTime(item.observation.lastReliableAt)}</p>
                        {item.observation.reason && <p>{item.observation.reason}</p>}
                        <Link to={`/trader-sync/subscriptions/${item.id}`}>View subscription</Link>
                    </div>
                ))}
                <Link to='/trader-sync/subscriptions'>Manage subscriptions</Link>
            </div>
            {selected && page && !chosen && (
                <p role='status'>
                    This target is outside Current. Its activity history remains selected.{' '}
                    <button type='button' disabled={disabled} onClick={() => onSelect()}>
                        Clear target filter
                    </button>
                </p>
            )}
        </aside>
    );
};
