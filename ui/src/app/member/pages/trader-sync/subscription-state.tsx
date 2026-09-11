import * as React from 'react';
import {Alert, Button, Input, Modal, Space, Typography} from 'antd';
import {useNavigate} from 'react-router-dom';
import type {Subscription, TargetDisplay} from '../../trader-sync-models';
import {memberServices as services} from '../../services';
import {requestErrorDetails, requestErrorMessage} from '../../../shared/services/requests';
import type {AbortablePromise, ReadScope} from '../../../shared/use-visible-query';
import {readSubscriptionEdit, saveSubscriptionEdit, SubscriptionEdit, SubscriptionIntent, saveAddDraft} from './state';

export const allowedSubscriptionActions = (status: Subscription['status']): Array<'pause' | 'resume' | 'cancel'> => {
    switch (status) {
        case 'pending_baseline':
            return ['cancel'];
        case 'healthy':
        case 'interrupted':
            return ['pause', 'cancel'];
        case 'paused':
        case 'permission_disabled':
            return ['resume', 'cancel'];
        case 'cancelled':
            return [];
    }
};
export const subscriptionStatusLabel = (status: Subscription['status']) =>
    ({
        pending_baseline: 'Preparing monitoring',
        healthy: 'Monitoring',
        interrupted: 'Monitoring interrupted',
        paused: 'Paused',
        permission_disabled: 'Disabled by access change',
        cancelled: 'Cancelled'
    })[status];
export const subscriptionTime = (value?: string) =>
    value && Number.isFinite(Date.parse(value))
        ? `${new Intl.DateTimeFormat('en-GB', {timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false}).format(new Date(value))} UTC+8`
        : 'Unknown — no confirmed time';
export const SubscriptionIdentity = ({wallet, note, display}: {wallet: string; note: string; display: TargetDisplay}) => {
    const name = display.displayName.evidence.availability === 'available' ? display.displayName.value : undefined;
    const profile = display.profileURL.evidence.availability === 'available' ? display.profileURL.value : undefined;
    const avatar = display.avatar.evidence.availability === 'available' ? display.avatar.value : undefined;
    return (
        <div className='trader-sync-subscription-identity'>
            {avatar && (
                <img
                    src={avatar}
                    alt=''
                    width={40}
                    height={40}
                    loading='lazy'
                    onError={event => {
                        event.currentTarget.hidden = true;
                    }}
                />
            )}
            <div>
                <Typography.Text strong={true}>{note.trim() || name || wallet}</Typography.Text>
                {note.trim() && name && <div>{name}</div>}
                {!name && <div>Public name unavailable: {display.displayName.evidence.reasonCode || 'not provided'}</div>}
                <Typography.Paragraph className='trader-sync-subscription-wallet' copyable={{text: wallet}}>
                    {wallet}
                </Typography.Paragraph>
                {profile && (
                    <a href={profile} target='_blank' rel='noreferrer'>
                        Polymarket profile
                    </a>
                )}
                <div className='trader-sync-subscription-asof'>Profile snapshot queried {subscriptionTime(display.displayName.evidence.queriedAt)}</div>
            </div>
        </div>
    );
};
const mayHaveCommitted = (error: unknown) => {
    const {code, status} = requestErrorDetails(error);
    if (code !== undefined && [3, 5, 6, 7, 8, 9, 10, 16].includes(code)) return false;
    return !(status !== undefined && status >= 400 && status < 500 && status !== 408);
};

/** One persisted intent per subscription, shared by lifecycle actions and wallet note edits. */
export const SubscriptionControls = ({
    ownerId,
    scope,
    subscription,
    onUpdated,
    editNote = false
}: {
    ownerId: string;
    scope: ReadScope;
    subscription: Subscription;
    onUpdated: (next: Subscription) => void;
    editNote?: boolean;
}) => {
    const navigate = useNavigate();
    const subscriptionRef = React.useRef(subscription);
    subscriptionRef.current = subscription;
    const [edit, setEdit] = React.useState(() => readSubscriptionEdit(ownerId, subscription.id));
    const editRef = React.useRef(edit);
    const [busy, setBusy] = React.useState(false);
    const busyRef = React.useRef(false);
    const [confirm, setConfirm] = React.useState(false);
    const [message, setMessage] = React.useState('');
    const mounted = React.useRef(false);
    const requestRef = React.useRef<AbortablePromise<unknown>>();
    const update = (value: SubscriptionEdit) => {
        if (!mounted.current || !scope.isCurrent()) return;
        editRef.current = value;
        saveSubscriptionEdit(ownerId, subscription.id, scope, value);
        setEdit(value);
    };
    React.useLayoutEffect(() => {
        mounted.current = true;
        const stop = () => {
            requestRef.current?.abort?.();
            requestRef.current = undefined;
            busyRef.current = false;
        };
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            stop();
            editRef.current = {};
            setEdit({});
            setMessage('');
            setConfirm(false);
            setBusy(false);
        });
        return () => {
            mounted.current = false;
            stop();
            unsubscribe?.();
        };
    }, [scope.key, subscription.id]);
    const current = () => mounted.current && scope.isCurrent();
    const fetchLatest = async () => {
        const request = services.traderSync.getSubscription(subscription.id);
        requestRef.current = request;
        const latest = await request;
        if (!current()) return;
        onUpdated(latest);
        update({...editRef.current, needsRefresh: undefined, ...(editRef.current.needsRefresh === 'note' ? {noteRevision: latest.noteRevision, serverNote: latest.note} : {})});
    };
    const refreshLatest = async () => {
        if (!current() || busyRef.current) return;
        busyRef.current = true;
        setBusy(true);
        try {
            await fetchLatest();
            if (current()) setMessage('Latest state loaded. Choose your action again.');
        } catch (error) {
            if (current()) setMessage(requestErrorMessage(error));
        } finally {
            if (current()) {
                busyRef.current = false;
                setBusy(false);
                requestRef.current = undefined;
            }
        }
    };
    const run = async (intent: SubscriptionIntent) => {
        if (!current() || busyRef.current) return;
        busyRef.current = true;
        setBusy(true);
        setMessage('');
        update({...editRef.current, intent});
        let dispatched = false;
        try {
            const request =
                intent.action === 'note'
                    ? services.traderSync.updateTargetNote(subscription.wallet, intent.payload)
                    : intent.action === 'pause'
                      ? services.traderSync.pauseSubscription(subscription.id, intent.payload)
                      : intent.action === 'resume'
                        ? services.traderSync.resumeSubscription(subscription.id, intent.payload)
                        : services.traderSync.cancelSubscription(subscription.id, intent.payload);
            dispatched = true;
            requestRef.current = request;
            const result = await request;
            if (!current()) return;
            if (intent.action === 'note') {
                const note = result as import('../../trader-sync-models').TargetNote;
                update({});
                const currentSubscription = subscriptionRef.current;
                onUpdated(
                    BigInt(note.revision) >= BigInt(currentSubscription.noteRevision) ? {...currentSubscription, note: note.note, noteRevision: note.revision} : currentSubscription
                );
                setMessage('Note saved. Historical activity snapshots are unchanged.');
            } else {
                update({...editRef.current, intent: undefined});
                onUpdated(result as Subscription);
                setMessage('Subscription updated. Already queued notifications may still arrive later.');
            }
            setConfirm(false);
        } catch (error) {
            if (!current()) return;
            if (requestErrorDetails(error).code === 10) {
                update({...editRef.current, intent: undefined, needsRefresh: intent.action === 'note' ? 'note' : 'change'});
                setConfirm(false);
                setMessage('The subscription changed. Read the latest state and choose your action again.');
                try {
                    await fetchLatest();
                } catch (readError) {
                    if (current()) setMessage(`Could not load the latest state. Refresh before choosing again. ${requestErrorMessage(readError)}`);
                }
            } else if (dispatched && mayHaveCommitted(error)) {
                setConfirm(false);
                setMessage('The request result is unknown. Recover the original request before making another change.');
            } else {
                update({...editRef.current, intent: undefined});
                setConfirm(false);
                setMessage(requestErrorMessage(error));
            }
        } finally {
            if (current()) {
                busyRef.current = false;
                setBusy(false);
                requestRef.current = undefined;
            }
        }
    };
    const start = (action: 'pause' | 'resume' | 'cancel') => {
        if (busyRef.current || editRef.current.intent || editRef.current.needsRefresh || !scope.isCurrent()) return;
        void run({action, payload: {expectedRevision: subscription.revision, requestId: crypto.randomUUID()}});
    };
    const locked = busy || Boolean(edit.intent) || Boolean(edit.needsRefresh) || !scope.isCurrent();
    const note = edit.note ?? subscription.note;
    return (
        <div className='trader-sync-subscription-controls'>
            {editNote && (
                <div>
                    <label htmlFor='subscription-note'>Private wallet note</label>
                    <Input
                        id='subscription-note'
                        value={note}
                        disabled={locked}
                        onChange={event => {
                            if (!busyRef.current && !editRef.current.intent)
                                update({...editRef.current, note: event.target.value, noteRevision: editRef.current.noteRevision ?? subscription.noteRevision});
                        }}
                    />
                    <p>
                        {Array.from(note).length} / 20 characters. This retained note applies to current and future subscriptions for this wallet. Historical activity snapshots
                        stay unchanged.
                    </p>
                    {edit.serverNote !== undefined && (
                        <Alert
                            type='warning'
                            title='Note changed elsewhere'
                            description={<span>Latest server note: {edit.serverNote || '(empty)'}. Your draft is preserved; save again to apply it.</span>}
                        />
                    )}
                    <Button
                        disabled={locked || Array.from(note).length > 20}
                        onClick={() => {
                            if (!locked)
                                void run({action: 'note', payload: {requestId: crypto.randomUUID(), expectedRevision: edit.noteRevision ?? subscription.noteRevision, note}});
                        }}>
                        Save note
                    </Button>
                </div>
            )}
            <Space wrap={true}>
                {allowedSubscriptionActions(subscription.status).map(action => (
                    <Button key={action} danger={action === 'cancel'} disabled={locked} onClick={() => (action === 'cancel' ? setConfirm(true) : start(action))}>
                        {action === 'pause' ? 'Pause' : action === 'resume' ? 'Resume' : 'Cancel subscription'}
                    </Button>
                ))}
                {subscription.status === 'cancelled' && (
                    <Button
                        disabled={locked}
                        onClick={() => {
                            if (!scope.isCurrent()) return;
                            saveAddDraft({ownerId, input: subscription.wallet, note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
                            navigate('/trader-sync/add');
                        }}>
                        Subscribe again
                    </Button>
                )}
                {edit.needsRefresh && (
                    <Button disabled={busy || !scope.isCurrent()} onClick={() => void refreshLatest()}>
                        Refresh latest state
                    </Button>
                )}
                {edit.intent && (
                    <Button loading={busy} disabled={busy || !scope.isCurrent()} onClick={() => void run(edit.intent!)}>
                        Recover request result
                    </Button>
                )}
            </Space>
            <p>Already queued notifications may still arrive later. Resume prepares a new monitoring baseline; interruptions recover automatically without a manual resume.</p>
            {message && <Alert type='info' title={message} />}
            {confirm && (
                <Modal
                    open={true}
                    title='Cancel subscription?'
                    okText='Confirm cancellation'
                    okButtonProps={{danger: true}}
                    confirmLoading={busy}
                    onCancel={() => {
                        if (!busy) setConfirm(false);
                    }}
                    onOk={() => start('cancel')}>
                    <SubscriptionIdentity wallet={subscription.wallet} note={subscription.note} display={subscription.targetDisplay} />
                    <p>
                        This cannot be restored. Cancellation releases one subscription slot. History and the wallet note are retained. Already queued notifications continue and
                        may arrive later. A new subscription requires a fresh confirmation.
                    </p>
                </Modal>
            )}
        </div>
    );
};
