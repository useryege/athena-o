import {CheckCircleOutlined, CopyOutlined, DeleteOutlined, DisconnectOutlined, LinkOutlined, ReloadOutlined, SendOutlined, WarningOutlined} from '@ant-design/icons';
import {Alert, Button, QRCode, Space, Spin, Steps, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, Section} from '../../components';
import {useLocation, useNavigate} from 'react-router-dom';
import {readAddDraft, captureTraderSyncScope} from './trader-sync/state';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingDateTime} from '../../shared/format';
import {requestErrorMessage} from '../../shared/services/requests';
import {clearTelegramBindingInstructions, readTelegramBindingInstructions, storeTelegramBindingInstructions} from '../notification-storage';
import type {StoredTelegramInstructions} from '../notification-storage';
import {memberServices as services} from '../services';
import type {TelegramBindingAttempt, TelegramNotificationSettings} from '../notification-service';

const telegramStatusPollIntervalMs = 3_000;

type NotificationAction = 'begin' | 'cancel' | 'disconnect' | 'restart' | null;
type AbortableRequest = Promise<unknown> & {abort?: () => void};

const telegramIdentity = (settings: TelegramNotificationSettings) => {
    const binding = settings.binding;
    if (!binding) {
        return '';
    }
    const username = binding.telegramUsername.replace(/^@+/, '');
    if (binding.telegramDisplayName && username) {
        return `${binding.telegramDisplayName} (@${username})`;
    }
    if (username) {
        return `@${username}`;
    }
    return binding.telegramDisplayName || 'Telegram account';
};

const expiresAtMilliseconds = (attempt?: TelegramBindingAttempt) => {
    const value = Date.parse(attempt?.expiresAt || '');
    return Number.isFinite(value) ? value : 0;
};

const remainingTime = (expiresAt: number, now: number) => {
    if (expiresAt <= 0) {
        return '';
    }
    const remainingSeconds = Math.max(0, Math.ceil((expiresAt - now) / 1_000));
    const minutes = Math.floor(remainingSeconds / 60);
    const seconds = remainingSeconds % 60;
    return `${minutes}:${String(seconds).padStart(2, '0')}`;
};

const bindingFailureMessage = (reason: string) => {
    switch (reason.trim().toLowerCase()) {
        case 'expired':
            return 'This setup link expired. Create a new one-time link and try again.';
        case 'telegram_identity_in_use':
        case 'telegram_account_in_use':
        case 'account_conflict':
            return 'This Telegram identity cannot be connected to this account. Use a different Telegram account or contact support.';
        default:
            return 'Athena could not complete this Telegram setup. Create a new link and try again.';
    }
};

export const NotificationsPage = () => {
    const ctx = React.useContext(Context);
    const {user} = useAuthorization();
    const location = useLocation();
    const navigate = useNavigate();
    const scope = React.useMemo(() => captureTraderSyncScope(user.accountId), [user.accountId]);
    const [, refreshDraft] = React.useReducer(value => value + 1, 0);
    React.useLayoutEffect(() => scope.subscribeInvalidation!(refreshDraft), [scope]);
    const canReturn = location.state?.returnTo === '/trader-sync/add' && Boolean(readAddDraft(user.accountId));
    const [settings, setSettings] = React.useState<TelegramNotificationSettings>();
    const [instructions, setInstructions] = React.useState<StoredTelegramInstructions | undefined>(readTelegramBindingInstructions);
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error>();
    const [action, setAction] = React.useState<NotificationAction>(null);
    const [now, setNow] = React.useState(Date.now());
    const mountedRef = React.useRef(false);
    const readGenerationRef = React.useRef(0);
    const readRequestRef = React.useRef<ReturnType<typeof services.memberNotifications.getTelegramSettings>>();
    const readPendingRef = React.useRef<Promise<void>>();
    const actionRequestRef = React.useRef<AbortableRequest>();
    const confirmRef = React.useRef<{destroy(): void}>();
    const channelStatusFocusRef = React.useRef<HTMLDivElement>(null);
    const setupFocusRef = React.useRef<HTMLDivElement>(null);
    const connectedFocusRef = React.useRef<HTMLDivElement>(null);
    const previousFocusStateRef = React.useRef('');

    const refresh = React.useCallback((showLoading = false): Promise<void> => {
        if (actionRequestRef.current) {
            return Promise.resolve();
        }
        if (showLoading) {
            setLoading(true);
        }
        if (readPendingRef.current) {
            return readPendingRef.current;
        }
        const generation = readGenerationRef.current;
        const request = services.memberNotifications.getTelegramSettings();
        readRequestRef.current = request;
        const pending = (async () => {
            try {
                const next = await request;
                if (!mountedRef.current || generation !== readGenerationRef.current) {
                    return;
                }
                setSettings(next);
                setError(undefined);
                setInstructions(current => {
                    if (!current || (next.attempt?.status === 'pending' && next.attempt.id === current.attemptId)) {
                        return current;
                    }
                    clearTelegramBindingInstructions();
                    return undefined;
                });
            } catch (refreshError) {
                if (mountedRef.current && generation === readGenerationRef.current) {
                    setError(refreshError instanceof Error ? refreshError : new Error(requestErrorMessage(refreshError)));
                }
            } finally {
                if (readRequestRef.current === request) {
                    readRequestRef.current = undefined;
                }
                if (mountedRef.current && generation === readGenerationRef.current) {
                    setLoading(false);
                }
            }
        })();
        readPendingRef.current = pending;
        void pending.finally(() => {
            if (readPendingRef.current === pending) {
                readPendingRef.current = undefined;
            }
        });
        return pending;
    }, []);

    const invalidatePendingRefresh = React.useCallback(() => {
        readGenerationRef.current += 1;
        readRequestRef.current?.abort?.();
        readRequestRef.current = undefined;
        readPendingRef.current = undefined;
        setLoading(false);
    }, []);

    React.useEffect(() => {
        mountedRef.current = true;
        void refresh(true);
        return () => {
            mountedRef.current = false;
            readGenerationRef.current += 1;
            readRequestRef.current?.abort?.();
            readRequestRef.current = undefined;
            readPendingRef.current = undefined;
            actionRequestRef.current?.abort?.();
            confirmRef.current?.destroy();
        };
    }, [refresh]);

    const attempt = settings?.attempt;
    const expiryMilliseconds = expiresAtMilliseconds(attempt);
    const attemptExpired = Boolean(attempt?.status === 'pending' && expiryMilliseconds > 0 && expiryMilliseconds <= now);
    const countdownAttemptId = attempt?.status === 'pending' && !attemptExpired ? attempt.id : '';

    React.useEffect(() => {
        if (!attemptExpired || !attempt || instructions?.attemptId !== attempt.id) {
            return;
        }
        clearTelegramBindingInstructions();
        setInstructions(undefined);
    }, [attempt, attemptExpired, instructions?.attemptId]);

    React.useEffect(() => {
        if (action) {
            return;
        }
        const pollWhenVisible = () => {
            if (document.visibilityState === 'visible') {
                void refresh(false);
            }
        };
        const timer = window.setInterval(pollWhenVisible, telegramStatusPollIntervalMs);
        window.addEventListener('focus', pollWhenVisible);
        document.addEventListener('visibilitychange', pollWhenVisible);
        return () => {
            window.clearInterval(timer);
            window.removeEventListener('focus', pollWhenVisible);
            document.removeEventListener('visibilitychange', pollWhenVisible);
        };
    }, [action, refresh]);

    React.useEffect(() => {
        if (!countdownAttemptId || expiryMilliseconds <= 0) {
            return;
        }
        setNow(Date.now());
        const timer = window.setInterval(() => setNow(Date.now()), 1_000);
        return () => window.clearInterval(timer);
    }, [countdownAttemptId, expiryMilliseconds]);

    const beginSetup = async () => {
        if (!mountedRef.current || action || actionRequestRef.current) {
            return;
        }
        setAction(settings?.attempt ? 'restart' : 'begin');
        invalidatePendingRefresh();
        try {
            const beginRequest = services.memberNotifications.beginTelegramBinding();
            actionRequestRef.current = beginRequest;
            const result = await beginRequest;
            if (!mountedRef.current) {
                return;
            }
            const nextInstructions = storeTelegramBindingInstructions(result);
            setInstructions(nextInstructions);
            setSettings(current => ({
                botAvailable: true,
                botUsername: result.botUsername || current?.botUsername || '',
                binding: current?.binding,
                attempt: result.attempt
            }));
            setError(undefined);
            setNow(Date.now());
            ctx.notifications.success('Telegram setup started', 'Open Athena Bot and tap Start before the link expires.');
        } catch (actionError) {
            if (mountedRef.current) {
                ctx.notifications.error('Could not start Telegram setup', requestErrorMessage(actionError));
                actionRequestRef.current = undefined;
                void refresh(false);
            }
        } finally {
            if (mountedRef.current) {
                setAction(null);
            }
            actionRequestRef.current = undefined;
        }
    };

    const cancelSetup = async () => {
        if (!mountedRef.current || action || actionRequestRef.current) {
            return;
        }
        setAction('cancel');
        invalidatePendingRefresh();
        try {
            const request = services.memberNotifications.cancelTelegramBindingAttempt();
            actionRequestRef.current = request;
            await request;
            if (!mountedRef.current) {
                return;
            }
            clearTelegramBindingInstructions();
            setInstructions(undefined);
            setSettings(current => (current ? {...current, attempt: undefined} : current));
            setError(undefined);
            ctx.notifications.success('Telegram setup cancelled');
        } catch (actionError) {
            if (mountedRef.current) {
                ctx.notifications.error('Could not cancel Telegram setup', requestErrorMessage(actionError));
                actionRequestRef.current = undefined;
                void refresh(false);
            }
        } finally {
            if (mountedRef.current) {
                setAction(null);
            }
            actionRequestRef.current = undefined;
        }
    };

    const disconnect = async () => {
        if (!mountedRef.current || action || actionRequestRef.current) {
            return;
        }
        setAction('disconnect');
        invalidatePendingRefresh();
        try {
            const request = services.memberNotifications.disconnectTelegram();
            actionRequestRef.current = request;
            await request;
            if (!mountedRef.current) {
                return;
            }
            clearTelegramBindingInstructions();
            setInstructions(undefined);
            setSettings(current => (current ? {...current, binding: undefined, attempt: undefined} : current));
            setError(undefined);
            ctx.notifications.success('Telegram disconnected', 'Athena will no longer send notifications to this Telegram account.');
        } catch (actionError) {
            if (mountedRef.current) {
                ctx.notifications.error('Could not disconnect Telegram', requestErrorMessage(actionError));
                actionRequestRef.current = undefined;
                void refresh(false);
            }
        } finally {
            if (mountedRef.current) {
                setAction(null);
            }
            actionRequestRef.current = undefined;
        }
    };

    const confirmDisconnect = () => {
        confirmRef.current?.destroy();
        const handle = ctx.modal.confirm({
            title: 'Disconnect Telegram?',
            content: `${settings ? telegramIdentity(settings) : 'Current Telegram account'}. Athena will stop sending notifications to this Telegram account and cancel setup. Old Trader Sync notifications without send permission are cancelled; a notification already authorized may still arrive. History is not backfilled after reconnecting.`,
            okText: 'Disconnect',
            cancelText: 'Keep connected',
            autoFocusButton: 'cancel',
            okButtonProps: {danger: true},
            onOk: async () => {
                try {
                    await disconnect();
                } finally {
                    if (confirmRef.current === handle) {
                        confirmRef.current = undefined;
                    }
                }
            },
            onCancel: () => {
                if (confirmRef.current === handle) {
                    confirmRef.current = undefined;
                }
            }
        });
        confirmRef.current = handle;
    };

    const copyFallbackCommand = async () => {
        if (!instructions?.fallbackCommand) {
            return;
        }
        try {
            if (!navigator.clipboard?.writeText) {
                throw new Error('Clipboard access is unavailable');
            }
            await navigator.clipboard.writeText(instructions.fallbackCommand);
            ctx.notifications.success('Telegram command copied');
        } catch (copyError) {
            ctx.notifications.error('Could not copy Telegram command', requestErrorMessage(copyError));
        }
    };

    const binding = settings?.binding;
    const activeInstructions = Boolean(attempt && instructions?.attemptId === attempt.id && attempt.status === 'pending' && !attemptExpired);
    const setupNeedsRecovery = Boolean(attempt && (attempt.status === 'failed' || attemptExpired));
    const botUsername = (settings?.botUsername || '').replace(/^@+/, '');
    const status = !settings?.botAvailable
        ? {label: 'Unavailable', color: 'default' as const, className: 'telegram-channel-card--unavailable'}
        : binding?.status === 'connected'
          ? {label: 'Connected', color: 'success' as const, className: 'telegram-channel-card--connected'}
          : binding?.status === 'unreachable'
            ? {label: 'Needs attention', color: 'warning' as const, className: 'telegram-channel-card--warning'}
            : attempt?.status === 'failed'
              ? {label: 'Setup failed', color: 'error' as const, className: 'telegram-channel-card--warning'}
              : attemptExpired
                ? {label: 'Link expired', color: 'warning' as const, className: 'telegram-channel-card--warning'}
                : attempt?.status === 'pending'
                  ? {label: 'Waiting for Telegram', color: 'processing' as const, className: 'telegram-channel-card--pending'}
                  : {label: 'Not connected', color: 'default' as const, className: ''};
    const focusState = !settings
        ? ''
        : !settings.botAvailable
          ? 'bot-unavailable'
          : attempt
            ? `attempt:${attempt.id}:${attempt.status === 'failed' ? 'failed' : attemptExpired ? 'expired' : activeInstructions ? 'ready' : 'remote'}`
            : binding?.status === 'connected'
              ? `binding:${binding.revision}:connected`
              : binding?.status === 'unreachable'
                ? `binding:${binding.revision}:unreachable`
                : 'unbound';

    React.useEffect(() => {
        if (!focusState) {
            return;
        }
        const previous = previousFocusStateRef.current;
        previousFocusStateRef.current = focusState;
        if (!previous || previous === focusState) {
            return;
        }
        const target = focusState.startsWith('attempt:')
            ? setupFocusRef
            : focusState.endsWith(':connected')
              ? connectedFocusRef
              : focusState.endsWith(':unreachable') || focusState === 'bot-unavailable'
                ? channelStatusFocusRef
                : undefined;
        if (!target) {
            return;
        }
        const frame = window.requestAnimationFrame(() => target.current?.focus({preventScroll: false}));
        return () => window.cancelAnimationFrame(frame);
    }, [focusState]);

    return (
        <AppPage
            title='Notifications'
            extra={
                canReturn ? (
                    <Button
                        onClick={() => {
                            if (readAddDraft(user.accountId)) navigate('/trader-sync/add');
                        }}>
                        Return to Trader Sync
                    </Button>
                ) : undefined
            }
            subtitle='Choose where Athena sends account notifications. Telegram is the only channel currently available.'
            loading={loading && !settings}
            error={!settings ? error : undefined}
            onRefresh={() => void refresh(true)}>
            {settings && (
                <>
                    {error && <Alert className='app-page__alert' type='error' showIcon={true} title='Could not refresh Telegram status' description={error.message} />}
                    <section className='telegram-delivery' aria-labelledby='telegram-delivery-title'>
                        <Typography.Title id='telegram-delivery-title' level={2}>
                            Delivery channels
                        </Typography.Title>
                        <div ref={channelStatusFocusRef} className={`telegram-channel-card ${status.className}`} tabIndex={-1}>
                            <div className='telegram-channel-card__copy'>
                                <div className='telegram-channel-card__heading'>
                                    <span className='telegram-channel-card__heading-title'>
                                        <SendOutlined aria-hidden='true' />
                                        <Typography.Title level={3}>Telegram</Typography.Title>
                                    </span>
                                    <span className='telegram-channel-card__status' role='status' aria-live='polite' aria-atomic='true'>
                                        <Tag color={status.color}>{status.label}</Tag>
                                    </span>
                                </div>
                                <Typography.Paragraph type='secondary'>
                                    Receive Athena notifications in a private chat with {botUsername ? `@${botUsername}` : 'Athena Bot'}.
                                </Typography.Paragraph>
                            </div>
                            {binding && (
                                <dl className='telegram-connection-details'>
                                    <div>
                                        <dt>Telegram account</dt>
                                        <dd>{telegramIdentity(settings)}</dd>
                                    </div>
                                    <div>
                                        <dt>Connected</dt>
                                        <dd>{formatBeijingDateTime(binding.boundAt) || '-'}</dd>
                                    </div>
                                </dl>
                            )}
                            <Space className='telegram-channel-card__actions' wrap={true}>
                                {binding ? (
                                    <>
                                        {setupNeedsRecovery ? null : attempt?.status === 'pending' && !attemptExpired ? (
                                            <Button
                                                icon={<DeleteOutlined aria-hidden={true} />}
                                                loading={action === 'cancel'}
                                                disabled={Boolean(action)}
                                                onClick={() => void cancelSetup()}>
                                                Cancel new setup
                                            </Button>
                                        ) : (
                                            <Button
                                                type='primary'
                                                icon={<ReloadOutlined aria-hidden={true} />}
                                                loading={action === (attempt ? 'restart' : 'begin')}
                                                disabled={Boolean(action) || !settings.botAvailable}
                                                onClick={() => void beginSetup()}>
                                                Reconnect
                                            </Button>
                                        )}
                                        <Button
                                            danger={true}
                                            icon={<DisconnectOutlined aria-hidden={true} />}
                                            loading={action === 'disconnect'}
                                            disabled={Boolean(action)}
                                            onClick={confirmDisconnect}>
                                            Disconnect
                                        </Button>
                                    </>
                                ) : attempt ? (
                                    setupNeedsRecovery ? null : (
                                        <Button
                                            icon={<DeleteOutlined aria-hidden={true} />}
                                            loading={action === 'cancel'}
                                            disabled={Boolean(action)}
                                            onClick={() => void cancelSetup()}>
                                            Cancel setup
                                        </Button>
                                    )
                                ) : (
                                    <Button
                                        type='primary'
                                        icon={<LinkOutlined aria-hidden={true} />}
                                        loading={action === 'begin'}
                                        disabled={Boolean(action) || !settings.botAvailable}
                                        onClick={() => void beginSetup()}>
                                        Configure
                                    </Button>
                                )}
                            </Space>
                            <Typography.Paragraph className='telegram-channel-card__consequences' type='secondary'>
                                Disconnecting or reconnecting cancels old Trader Sync notifications that have not received send permission. A notification already authorized may
                                still arrive. History is not backfilled for a new binding.
                            </Typography.Paragraph>
                        </div>
                        {!settings?.botAvailable && (
                            <Alert
                                className='telegram-channel-alert'
                                type='warning'
                                showIcon={true}
                                title='Telegram notifications are unavailable'
                                description='Athena Bot is not configured or is temporarily unavailable. Refresh this page after an administrator restores the service.'
                            />
                        )}
                        {binding?.status === 'unreachable' && (
                            <Alert
                                className='telegram-channel-alert'
                                type='warning'
                                showIcon={true}
                                title='Athena cannot currently reach this Telegram account'
                                description='Open the bot chat and make sure the bot is not blocked, or create a new setup link. Athena keeps the current binding until the new setup succeeds.'
                            />
                        )}
                    </section>
                    {attempt && (
                        <Section title='Telegram setup'>
                            <div
                                ref={setupFocusRef}
                                className='telegram-setup-focus-target'
                                tabIndex={-1}
                                aria-label={`Telegram setup: ${attempt.status === 'failed' ? 'failed' : attemptExpired ? 'link expired' : 'waiting for Telegram'}`}>
                                {binding && (
                                    <Alert
                                        className='telegram-setup-existing-binding'
                                        type='info'
                                        showIcon={true}
                                        title={binding.status === 'connected' ? 'Your current connection remains active' : 'Your current binding remains in place'}
                                        description='Completing this setup replaces the current Telegram binding. Cancelling it leaves the current binding unchanged.'
                                    />
                                )}
                                {attempt.status === 'failed' ? (
                                    <Alert type='error' showIcon={true} title='Telegram setup failed' description={bindingFailureMessage(attempt.failureReason)} />
                                ) : attemptExpired ? (
                                    <Alert
                                        type='warning'
                                        showIcon={true}
                                        title='This setup link has expired'
                                        description='Create a new one-time link before returning to Telegram.'
                                    />
                                ) : activeInstructions ? (
                                    <div className='telegram-setup-layout'>
                                        <div className='telegram-setup-steps'>
                                            <Steps
                                                direction='vertical'
                                                current={0}
                                                items={[
                                                    {
                                                        title: 'Open Athena Bot',
                                                        description: (
                                                            <Button
                                                                type='primary'
                                                                icon={<SendOutlined aria-hidden={true} />}
                                                                href={instructions?.deepLink}
                                                                target='_blank'
                                                                rel='noreferrer'>
                                                                Open Telegram
                                                            </Button>
                                                        )
                                                    },
                                                    {
                                                        title: 'Tap Start in Telegram',
                                                        description: 'The one-time link securely matches the Telegram chat to your signed-in Athena account.'
                                                    },
                                                    {
                                                        title: 'Return to Athena',
                                                        description: 'This page checks the connection every 3 seconds while it is visible.'
                                                    }
                                                ]}
                                            />
                                            <div className='telegram-fallback-command'>
                                                <div>
                                                    <Typography.Text strong={true}>Manual fallback</Typography.Text>
                                                    <Typography.Text type='secondary'>Send this exact command to {botUsername ? `@${botUsername}` : 'Athena Bot'}.</Typography.Text>
                                                </div>
                                                <code>{instructions?.fallbackCommand}</code>
                                                <Button icon={<CopyOutlined aria-hidden={true} />} onClick={() => void copyFallbackCommand()}>
                                                    Copy command
                                                </Button>
                                            </div>
                                        </div>
                                        <div className='telegram-setup-qr' aria-label='Telegram setup QR code'>
                                            <QRCode value={instructions?.deepLink || ''} size={184} bordered={false} />
                                            <Typography.Text strong={true}>Scan with another device</Typography.Text>
                                            <Typography.Text type='secondary'>The QR code opens the same one-time Telegram link.</Typography.Text>
                                        </div>
                                    </div>
                                ) : (
                                    <Alert
                                        type='info'
                                        showIcon={true}
                                        title='Setup is already in progress'
                                        description='The one-time Telegram link is stored only in the browser tab that created it. Cancel this attempt and create a new link to continue here.'
                                    />
                                )}
                                {attempt.status === 'pending' && !attemptExpired && (
                                    <div className='telegram-setup-status'>
                                        <span className='telegram-setup-status__indicator' role='status' aria-live='polite' aria-atomic='true'>
                                            <Spin size='small' />
                                            <strong>Waiting for Telegram</strong>
                                        </span>
                                        <Typography.Text type='secondary'>
                                            {expiryMilliseconds > 0
                                                ? `Expires in ${remainingTime(expiryMilliseconds, now)} · ${formatBeijingDateTime(attempt.expiresAt)}`
                                                : 'This one-time setup remains active until the server expires it.'}
                                        </Typography.Text>
                                        <Button icon={<ReloadOutlined aria-hidden={true} />} loading={loading} disabled={Boolean(action)} onClick={() => void refresh(true)}>
                                            Check now
                                        </Button>
                                    </div>
                                )}
                                {(attempt.status === 'failed' || attemptExpired || !activeInstructions) && (
                                    <div className='telegram-setup-recovery'>
                                        <Space wrap={true}>
                                            <Button
                                                type='primary'
                                                icon={attempt.status === 'failed' ? <WarningOutlined aria-hidden={true} /> : <ReloadOutlined aria-hidden={true} />}
                                                loading={action === 'restart'}
                                                disabled={Boolean(action) || !settings.botAvailable}
                                                onClick={() => void beginSetup()}>
                                                Create new link
                                            </Button>
                                            <Button
                                                icon={<DeleteOutlined aria-hidden={true} />}
                                                loading={action === 'cancel'}
                                                disabled={Boolean(action)}
                                                onClick={() => void cancelSetup()}>
                                                Cancel setup
                                            </Button>
                                        </Space>
                                    </div>
                                )}
                            </div>
                        </Section>
                    )}
                    {settings.botAvailable && binding?.status === 'connected' && (
                        <div ref={connectedFocusRef} className='telegram-connected-alert' role='region' tabIndex={-1} aria-label='Telegram connection ready'>
                            <Alert
                                type='success'
                                showIcon={true}
                                icon={<CheckCircleOutlined aria-hidden={true} />}
                                title='Telegram is ready'
                                description='Athena can now deliver account notifications to the connected private chat.'
                            />
                        </div>
                    )}
                </>
            )}
        </AppPage>
    );
};
