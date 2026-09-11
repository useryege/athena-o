import './trader-sync.css';
import * as React from 'react';
import {Alert, Button, Input, Space, Typography} from 'antd';
import {useNavigate} from 'react-router-dom';
import {AppPage, Section} from '../../../components';
import {Context} from '../../../shared/context';
import {requestErrorDetails, requestErrorMessage} from '../../../shared/services/requests';
import type {AbortablePromise} from '../../../shared/use-visible-query';
import {memberServices as services} from '../../services';
import type {CreateRequest} from '../../trader-sync-service';
import {ConfirmationCard} from './confirmation-card';
import {AddDraft, captureTraderSyncScope, readAddDraft, saveAddDraft, saveNewSubscriptionFocus} from './state';

type Phase = {kind: 'input' | 'resolving' | 'review' | 'creating'} | {kind: 'error'; message: string; unknown: boolean};
const tokenExpired = (expiresAt: string, now: number) => !Number.isFinite(Date.parse(expiresAt)) || Date.parse(expiresAt) <= now;
const blankDraft = (ownerId: string): AddDraft => ({ownerId, input: '', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
const mayHaveCommitted = (error: unknown) => {
    const {status, code} = requestErrorDetails(error);
    if (code !== undefined && [3, 5, 6, 7, 8, 9, 16].includes(code)) return false;
    if (status !== undefined && status >= 400 && status < 500 && status !== 408) return false;
    return true;
};
// A new owner mounts a fresh component before rendering any previous target.
export const TraderSyncAddPage = ({ownerId}: {ownerId: string}) => <AddFlow key={ownerId} ownerId={ownerId} />;
const AddFlow = ({ownerId}: {ownerId: string}) => {
    const navigate = useNavigate();
    const ctx = React.useContext(Context);
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [draft, setDraft] = React.useState(() => readAddDraft(ownerId) || blankDraft(ownerId));
    const draftRef = React.useRef(draft);
    const [phase, setPhase] = React.useState<Phase>(() =>
        draft.createRequest ? {kind: 'error', unknown: true, message: 'The subscription result needs to be recovered.'} : {kind: draft.target ? 'review' : 'input'}
    );
    const [now, setNow] = React.useState(Date.now());
    const [binding, setBinding] = React.useState<'loading' | 'connected' | 'unbound' | 'unreachable' | 'error'>('loading');
    const requestRef = React.useRef<AbortablePromise<unknown>>();
    const sequence = React.useRef(0);
    const mounted = React.useRef(false);
    const busy = React.useRef(false);
    const update = (next: AddDraft) => {
        if (!scope.isCurrent() || !mounted.current) return;
        draftRef.current = next;
        saveAddDraft(next);
        setDraft(next);
    };
    const cancel = () => {
        sequence.current++;
        requestRef.current?.abort?.();
        requestRef.current = undefined;
        busy.current = false;
    };
    React.useLayoutEffect(() => {
        mounted.current = true;
        if (draftRef.current.scrollY > 0) window.scrollTo({top: draftRef.current.scrollY, behavior: 'instant'});
        const unsubscribe = scope.subscribeInvalidation!(() => {
            cancel();
            draftRef.current = blankDraft(ownerId);
            setDraft(draftRef.current);
            setPhase({kind: 'input'});
        });
        return () => {
            mounted.current = false;
            cancel();
            unsubscribe();
        };
    }, [scope]);
    React.useEffect(() => {
        const timer = window.setInterval(() => setNow(Date.now()), 1000);
        const request = services.memberNotifications.getTelegramSettings();
        const unsubscribe = scope.subscribeInvalidation!(() => request.abort?.());
        request.then(
            settings => {
                if (mounted.current && scope.isCurrent()) setBinding(settings.binding?.status || 'unbound');
            },
            () => {
                if (mounted.current && scope.isCurrent()) setBinding('error');
            }
        );
        return () => {
            window.clearInterval(timer);
            unsubscribe();
            request.abort?.();
        };
    }, [scope]);
    const resolve = async () => {
        if (!scope.isCurrent() || busy.current || !draftRef.current.input.trim()) return;
        cancel();
        const current: AddDraft = {...draftRef.current, target: undefined, createRequest: undefined};
        update(current);
        const id = sequence.current;
        const isCurrent = () => mounted.current && scope.isCurrent() && id === sequence.current;
        busy.current = true;
        setPhase({kind: 'resolving'});
        try {
            const request = services.traderSync.resolveTarget(current.input.trim());
            requestRef.current = request;
            const target = await request;
            if (!isCurrent()) return;
            const sameWallet = current.wallet?.toLowerCase() === target.wallet.toLowerCase();
            update({
                ...current,
                wallet: target.wallet,
                target,
                note: sameWallet && current.noteEdited ? current.note : target.savedNote?.note || '',
                noteEdited: sameWallet && current.noteEdited
            });
            setNow(Date.now());
            setPhase({kind: 'review'});
        } catch (error) {
            if (isCurrent())
                setPhase({kind: 'error', unknown: false, message: requestErrorMessage(error, 'Could not verify this trader. Check the wallet or profile URL and try again.')});
        } finally {
            if (isCurrent()) {
                busy.current = false;
                requestRef.current = undefined;
            }
        }
    };
    const create = async () => {
        const current = draftRef.current;
        const target = current.target;
        if (!scope.isCurrent() || busy.current || !target || Array.from(current.note).length > 20 || target.existingSubscription || target.quota.used >= target.quota.limit) return;
        if (!current.createRequest && tokenExpired(target.expiresAt, Date.now())) return;
        const payload: CreateRequest = current.createRequest || {
            confirmationToken: target.confirmationToken,
            requestId: crypto.randomUUID(),
            ...(current.noteEdited ? {note: {value: current.note}} : {})
        };
        update({...current, createRequest: payload});
        const id = ++sequence.current;
        const isCurrent = () => mounted.current && scope.isCurrent() && id === sequence.current;
        busy.current = true;
        setPhase({kind: 'creating'});
        let dispatched = false;
        try {
            const request = services.traderSync.createSubscription(payload);
            dispatched = true;
            requestRef.current = request;
            const subscription = await request;
            if (!isCurrent()) return;
            saveNewSubscriptionFocus(ownerId, subscription.id);
            update({...current, target: undefined, createRequest: undefined});
            setPhase({kind: 'input'});
            ctx.notifications.success(
                'Subscription created',
                subscription.status === 'pending_baseline'
                    ? 'Preparing monitoring. Monitoring begins after the baseline is ready.'
                    : 'Open the subscription to view its current monitoring state.'
            );
            // Task17 consumes owner-scoped focus while retaining its current filters.
            navigate(current.returnPath === '/trader-sync' || current.returnPath.startsWith('/trader-sync?') ? current.returnPath : '/trader-sync');
        } catch (error) {
            if (!isCurrent()) return;
            const unknown = dispatched && mayHaveCommitted(error);
            if (!unknown) update({...current, target: undefined, createRequest: undefined});
            setPhase({
                kind: 'error',
                unknown,
                message: unknown ? 'The request result is unknown. Recover the original subscription result before starting a new confirmation.' : requestErrorMessage(error)
            });
        } finally {
            if (isCurrent()) {
                busy.current = false;
                requestRef.current = undefined;
            }
        }
    };
    const target = draft.target;
    const recovering = Boolean(draft.createRequest);
    const expired = Boolean(target && tokenExpired(target.expiresAt, now));
    const creating = phase.kind === 'creating';
    return (
        <div className='trader-sync-add'>
            <AppPage
                title='Add trader'
                subtitle='Review a Polymarket trader before starting Activity Alerts.'
                extra={<Button onClick={() => navigate('/trader-sync')}>Back to Trader Sync</Button>}>
                <Section title='Find trader'>
                    <form
                        onSubmit={event => {
                            event.preventDefault();
                            void resolve();
                        }}>
                        <label htmlFor='trader-sync-input'>Wallet address or Polymarket profile URL</label>
                        <Input
                            id='trader-sync-input'
                            value={draft.input}
                            autoComplete='off'
                            placeholder='0x… or a Polymarket profile URL'
                            onChange={event => {
                                cancel();
                                update({...draftRef.current, input: event.target.value, target: undefined, createRequest: undefined});
                                setPhase({kind: 'input'});
                            }}
                        />
                        <Typography.Paragraph type='secondary'>Use an address or profile URL. A display name alone cannot identify a trader.</Typography.Paragraph>
                        <Button type='primary' htmlType='submit' loading={phase.kind === 'resolving'} disabled={!draft.input.trim() || creating || !scope.isCurrent()}>
                            {target ? 'Resolve again' : 'Resolve trader'}
                        </Button>
                    </form>
                </Section>
                {phase.kind === 'error' && (
                    <Alert type='error' showIcon={true} title={phase.unknown ? 'Subscription result unknown' : 'Could not complete request'} description={phase.message} />
                )}
                {target && (
                    <>
                        <fieldset className='trader-sync-add__review' disabled={creating || recovering}>
                            <ConfirmationCard
                                key={target.confirmationToken}
                                target={target}
                                note={draft.note}
                                onNoteChange={note => {
                                    if (!busy.current && !draftRef.current.createRequest) update({...draftRef.current, note, noteEdited: true});
                                }}
                            />
                        </fieldset>
                        <Section title='Confirm subscription'>
                            <Typography.Paragraph>
                                Choose low-frequency traders. High-frequency activity is outside the performance guarantees. Activity Alerts does not execute trades. Missed
                                activity during interruptions is not backfilled.
                            </Typography.Paragraph>
                            <Typography.Paragraph>
                                {binding === 'unbound'
                                    ? 'Telegram is not connected. You can still subscribe and read activities in Athena.'
                                    : binding === 'connected'
                                      ? 'Telegram is connected for private-chat notifications.'
                                      : binding === 'unreachable'
                                        ? 'Telegram is unreachable. In-app activities remain available.'
                                        : binding === 'error'
                                          ? 'Telegram status is unavailable. You can still subscribe and read activities in Athena.'
                                          : 'Checking Telegram status…'}
                            </Typography.Paragraph>
                            <Button
                                disabled={creating}
                                onClick={() => {
                                    update({...draftRef.current, scrollY: window.scrollY});
                                    navigate('/notifications', {state: {returnTo: '/trader-sync/add'}});
                                }}>
                                Open Notifications
                            </Button>
                            <Typography.Paragraph>
                                {target.quota.used} / {target.quota.limit} current subscriptions
                            </Typography.Paragraph>
                            {target.existingSubscription ? (
                                <Alert
                                    type='info'
                                    title='Already subscribed'
                                    description={
                                        <Space orientation='vertical'>
                                            <span>
                                                Current state:{' '}
                                                {target.existingSubscription.status === 'healthy'
                                                    ? 'Monitoring'
                                                    : target.existingSubscription.status === 'interrupted'
                                                      ? 'Monitoring interrupted'
                                                      : target.existingSubscription.status}
                                            </span>
                                            <Button onClick={() => navigate(`/trader-sync/subscriptions/${target.existingSubscription!.id}`)}>View subscription</Button>
                                        </Space>
                                    }
                                />
                            ) : target.quota.used >= target.quota.limit ? (
                                <Alert
                                    type='warning'
                                    title='Subscription limit reached'
                                    description={
                                        <>
                                            <p>All current subscriptions, including paused and interrupted subscriptions, count toward your limit.</p>
                                            <Button onClick={() => navigate('/trader-sync/subscriptions')}>Manage subscriptions</Button>
                                        </>
                                    }
                                />
                            ) : (
                                <>
                                    {expired && !recovering && (
                                        <Alert type='warning' title='Confirmation expired' description='Resolve this trader again to review a fresh confirmation.' />
                                    )}
                                    <Button
                                        type='primary'
                                        loading={creating}
                                        disabled={creating || (!recovering && expired) || Array.from(draft.note).length > 20 || !scope.isCurrent()}
                                        onClick={() => void create()}>
                                        {recovering && !creating ? 'Recover subscription result' : 'Confirm subscription'}
                                    </Button>
                                </>
                            )}
                        </Section>
                    </>
                )}
            </AppPage>
        </div>
    );
};
