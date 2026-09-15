import {CheckCircleFilled, CloseCircleFilled, SafetyCertificateOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Input, Spin, Tag, Typography} from 'antd';
import * as React from 'react';
import {Registration, registrationService, RegistrationRequestError, UsernameAvailability} from '../../shared/services/registration-service';
import requests, {ACCOUNT_MAINTENANCE_MESSAGE, APPLICATION_REALM_QUERY} from '../../shared/services/requests';

type AvailabilityState = 'idle' | 'checking' | UsernameAvailability | 'error';

const usernamePattern = /^[A-Za-z0-9.-]+$/;
const walletAddressPattern = /^0x[0-9a-f]{40}$/i;

const isLocallyValidUsername = (username: string) =>
    username.length >= 3 && username.length <= 42 && usernamePattern.test(username) && /[A-Za-z0-9]/.test(username) && !walletAddressPattern.test(username);

const registrationErrorMessage = (error: unknown, fallback: string) => {
    if (!(error instanceof RegistrationRequestError)) {
        return fallback;
    }
    switch (error.reason) {
        case 'username_invalid':
            return 'Use 3–42 letters, numbers, periods, or hyphens.';
        case 'username_unavailable':
            return 'That username is not available. Choose another one.';
        case 'registration_expired':
            return 'Your registration session has expired. Return to sign in and try again.';
        case 'registration_unavailable':
            return 'Registration is temporarily unavailable. Please try again.';
        case 'google_not_allowed':
            return 'This identity cannot complete registration.';
        case 'maintenance':
            return ACCOUNT_MAINTENANCE_MESSAGE;
        default:
            return fallback;
    }
};

interface DisconnectablePhantomProvider {
    isPhantom?: boolean;
    disconnect?(): Promise<void>;
}

const disconnectPhantomBestEffort = async () => {
    const provider = (window as Window & {phantom?: {solana?: DisconnectablePhantomProvider}}).phantom?.solana;
    if (!provider?.isPhantom || !provider.disconnect) {
        return;
    }
    try {
        await Promise.race([provider.disconnect(), new Promise<void>(resolve => window.setTimeout(resolve, 750))]);
    } catch {
        // The server-side ticket is authoritative. A wallet disconnect is only
        // best-effort cleanup before the user chooses another Phantom account.
    }
};

const registrationIdentity = (registration: Registration) =>
    registration.provider === 'solana_wallet'
        ? {label: 'Verified Phantom wallet', value: registration.solanaAddress}
        : {label: 'Verified Google account', value: registration.verifiedEmail};

const compactSolanaAddress = (address: string) => (address.length > 16 ? `${address.slice(0, 7)}…${address.slice(-7)}` : address);

const restartLabel = (registration?: Registration) =>
    registration?.provider === 'solana_wallet' ? 'Use another Phantom wallet' : registration ? 'Use another Google account' : 'Return to sign in';

const availabilityPresentation: Record<AvailabilityState, {message: string; tone: 'muted' | 'success' | 'error'}> = {
    idle: {message: '3–42 characters: letters, numbers, periods, or hyphens.', tone: 'muted'},
    checking: {message: 'Checking availability…', tone: 'muted'},
    available: {message: 'Username is available.', tone: 'success'},
    invalid: {message: 'Use 3–42 letters, numbers, periods, or hyphens.', tone: 'error'},
    unavailable: {message: 'That username is not available.', tone: 'error'},
    error: {message: 'Unable to check availability right now. Try again.', tone: 'error'}
};

const AvailabilityMessage = (props: {state: AvailabilityState}) => {
    const presentation = availabilityPresentation[props.state];
    return (
        <div id='registration-username-status' className={`registration-username-status registration-username-status--${presentation.tone}`} aria-live='polite' aria-atomic='true'>
            {props.state === 'checking' && <Spin size='small' aria-hidden='true' />}
            {props.state === 'available' && <CheckCircleFilled aria-hidden='true' />}
            {(props.state === 'invalid' || props.state === 'unavailable' || props.state === 'error') && <CloseCircleFilled aria-hidden='true' />}
            <span>{presentation.message}</span>
        </div>
    );
};

export const RegisterPage = () => {
    const [registration, setRegistration] = React.useState<Registration>();
    const [loadError, setLoadError] = React.useState('');
    const [username, setUsername] = React.useState('');
    const [availability, setAvailability] = React.useState<AvailabilityState>('idle');
    const [submitError, setSubmitError] = React.useState('');
    const [submitting, setSubmitting] = React.useState(false);
    const [cancelling, setCancelling] = React.useState(false);
    const availabilityGeneration = React.useRef(0);

    React.useEffect(() => {
        let active = true;
        registrationService
            .get()
            .then(value => {
                if (!active) {
                    return;
                }
                const identity = registrationIdentity(value);
                if (!identity.value || !value.csrfToken) {
                    throw new Error('Registration response is incomplete');
                }
                setRegistration(value);
            })
            .catch(error => {
                if (active) {
                    setLoadError(registrationErrorMessage(error, 'Unable to load your registration. Return to sign in and try again.'));
                }
            });
        return () => {
            active = false;
        };
    }, []);

    React.useEffect(() => {
        const generation = ++availabilityGeneration.current;
        if (!username) {
            setAvailability('idle');
            return;
        }
        if (!isLocallyValidUsername(username)) {
            setAvailability('invalid');
            return;
        }

        const controller = new AbortController();
        setAvailability('checking');
        const timer = window.setTimeout(() => {
            registrationService
                .usernameAvailability(username, controller.signal)
                .then(status => {
                    if (availabilityGeneration.current === generation) {
                        setAvailability(status);
                    }
                })
                .catch(error => {
                    if (controller.signal.aborted || availabilityGeneration.current !== generation) {
                        return;
                    }
                    setAvailability('error');
                    if (error instanceof RegistrationRequestError && error.reason === 'registration_expired') {
                        setSubmitError(registrationErrorMessage(error, 'Your registration session has expired.'));
                    }
                });
        }, 400);

        return () => {
            window.clearTimeout(timer);
            controller.abort();
        };
    }, [username]);

    const updateUsername = (event: React.ChangeEvent<HTMLInputElement>) => {
        availabilityGeneration.current++;
        setUsername(event.target.value);
        setSubmitError('');
    };

    const createAccount = async (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!registration || submitting || availability !== 'available') {
            return;
        }
        setSubmitting(true);
        setSubmitError('');
        try {
            const created = await registrationService.create(username, registration.csrfToken);
            const fallbackReturnTo = registration.administrator ? '/admin/accounts' : '/account/access';
            window.location.replace(requests.toAbsURL(created.redirectTo || fallbackReturnTo));
        } catch (error) {
            setSubmitError(registrationErrorMessage(error, 'Unable to create your account. Please try again.'));
            if (error instanceof RegistrationRequestError && error.reason === 'username_unavailable') {
                setAvailability('unavailable');
            } else if (error instanceof RegistrationRequestError && error.reason === 'username_invalid') {
                setAvailability('invalid');
            }
            setSubmitting(false);
        }
    };

    const restartSignIn = async () => {
        if (cancelling) {
            return;
        }
        setCancelling(true);
        setSubmitError('');
        try {
            const requestedRealm = new URLSearchParams(location.search).get(APPLICATION_REALM_QUERY);
            const registrationRealm = registration ? (registration.administrator ? 'admin' : 'member') : requestedRealm === 'admin' ? 'admin' : 'member';
            const fallbackReturnTo = registrationRealm === 'admin' ? '/admin/accounts' : '/account/access';
            const googleQuery = new URLSearchParams({returnTo: fallbackReturnTo, [APPLICATION_REALM_QUERY]: registrationRealm});
            let redirectTo = registrationRealm === 'admin' ? '/admin/login' : '/login';
            if (registration?.provider === 'google') {
                redirectTo = `/auth/google/login?${googleQuery.toString()}`;
            }
            if (registration?.csrfToken) {
                try {
                    const cancelled = await registrationService.cancel(registration.csrfToken);
                    redirectTo = cancelled.redirectTo || redirectTo;
                } catch (error) {
                    if (!(error instanceof RegistrationRequestError) || error.reason !== 'registration_expired') {
                        throw error;
                    }
                }
            }
            if (registration?.provider === 'solana_wallet') {
                await disconnectPhantomBestEffort();
            }
            window.location.assign(requests.toAbsURL(redirectTo));
        } catch (error) {
            if (registration?.provider === 'solana_wallet') {
                await disconnectPhantomBestEffort();
            }
            setSubmitError(registrationErrorMessage(error, 'Unable to restart sign-in. Please try again.'));
            setCancelling(false);
        }
    };

    const inputError = availability === 'invalid' || availability === 'unavailable' || availability === 'error';
    const busy = submitting || cancelling;

    return (
        <main className='registration-screen' aria-labelledby='registration-title'>
            <div className='identity-brand'>
                <svg className='identity-brand-symbol' viewBox='0 0 28 28' aria-hidden='true'>
                    <path d='m4 23 10-19 10 19M8 17h12M11 23h6' />
                </svg>
                <span>ATHENA</span>
            </div>
            <Card className='registration-panel'>
                <header className='registration-panel__header'>
                    <div>
                        <Typography.Title id='registration-title' level={1}>
                            Choose a username
                        </Typography.Title>
                        <Typography.Paragraph type='secondary'>Your permanent public name in Athena.</Typography.Paragraph>
                    </div>
                </header>

                {!registration && !loadError && (
                    <div className='registration-panel__loading' aria-live='polite'>
                        <Spin />
                        <Typography.Text type='secondary'>Loading your verified identity…</Typography.Text>
                    </div>
                )}

                {loadError && (
                    <div className='registration-panel__failure'>
                        <Alert type='error' showIcon={true} title={loadError} />
                        <Button block={true} htmlType='button' loading={cancelling} disabled={cancelling} onClick={() => void restartSignIn()}>
                            {restartLabel(registration)}
                        </Button>
                    </div>
                )}

                {registration && (
                    <>
                        <section className='registration-identity' aria-label='Verified identity'>
                            <div>
                                <Typography.Text type='secondary'>{registrationIdentity(registration).label}</Typography.Text>
                                <Typography.Text
                                    className='registration-identity__value'
                                    copyable={registration.provider === 'solana_wallet' ? {text: registration.solanaAddress} : false}>
                                    {registration.provider === 'solana_wallet' ? compactSolanaAddress(registration.solanaAddress) : registrationIdentity(registration).value}
                                </Typography.Text>
                            </div>
                            {registration.administrator && (
                                <Tag className='registration-identity__administrator' icon={<SafetyCertificateOutlined />} color='success'>
                                    Administrator account
                                </Tag>
                            )}
                        </section>

                        <form className='registration-form' noValidate={true} onSubmit={createAccount} aria-busy={busy || undefined}>
                            <div className='registration-form__field'>
                                <label htmlFor='registration-username'>Username</label>
                                <Input
                                    id='registration-username'
                                    className={availability === 'available' ? 'registration-username-input registration-username-input--available' : 'registration-username-input'}
                                    value={username}
                                    prefix={<span className='registration-username-input__prefix'>@</span>}
                                    status={inputError ? 'error' : undefined}
                                    aria-invalid={inputError || undefined}
                                    aria-describedby='registration-username-status registration-username-permanence'
                                    autoComplete='username'
                                    autoCapitalize='none'
                                    spellCheck={false}
                                    disabled={busy}
                                    onChange={updateUsername}
                                />
                                <AvailabilityMessage state={availability} />
                                <Typography.Text id='registration-username-permanence' className='registration-form__permanence' type='secondary'>
                                    This username cannot be changed later.
                                </Typography.Text>
                            </div>

                            {submitError && <Alert className='registration-form__error' type='error' showIcon={true} title={submitError} role='alert' />}

                            <div className='registration-form__actions'>
                                <Button type='primary' block={true} htmlType='submit' loading={submitting} disabled={busy || availability !== 'available'}>
                                    Create account
                                </Button>
                                <Button block={true} htmlType='button' loading={cancelling} disabled={busy} onClick={() => void restartSignIn()}>
                                    {restartLabel(registration)}
                                </Button>
                            </div>
                        </form>
                    </>
                )}
            </Card>
            {!registration?.administrator && (
                <Typography.Paragraph className='identity-access-note' type='secondary'>
                    Business access is granted separately by an Athena administrator.
                </Typography.Paragraph>
            )}
        </main>
    );
};
