import {CheckCircleFilled, CloseCircleFilled, SafetyCertificateOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Input, Spin, Tag, Typography} from 'antd';
import * as React from 'react';
import {BrandMark} from '../components';
import {GoogleRegistration, registrationService, RegistrationRequestError, UsernameAvailability} from '../shared/services/registration-service';
import requests from '../shared/services/requests';

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
            return 'Your registration session has expired. Continue with Google again.';
        case 'registration_unavailable':
            return 'Registration is temporarily unavailable. Please try again.';
        case 'google_not_allowed':
            return 'This Google identity cannot complete registration.';
        default:
            return fallback;
    }
};

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
    const [registration, setRegistration] = React.useState<GoogleRegistration>();
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
                if (!value.verifiedEmail || !value.csrfToken) {
                    throw new Error('Registration response is incomplete');
                }
                setRegistration(value);
            })
            .catch(error => {
                if (active) {
                    setLoadError(registrationErrorMessage(error, 'Unable to load your registration. Continue with Google again.'));
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
            window.location.replace(requests.toAbsURL(created.redirectTo || '/account/access'));
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

    const useAnotherGoogleAccount = async () => {
        if (cancelling) {
            return;
        }
        setCancelling(true);
        setSubmitError('');
        try {
            let redirectTo = '/auth/google/login?returnTo=%2Faccount%2Faccess';
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
            window.location.assign(requests.toAbsURL(redirectTo));
        } catch (error) {
            setSubmitError(registrationErrorMessage(error, 'Unable to restart Google sign-in. Please try again.'));
            setCancelling(false);
        }
    };

    const inputError = availability === 'invalid' || availability === 'unavailable' || availability === 'error';
    const busy = submitting || cancelling;

    return (
        <main className='registration-screen' aria-labelledby='registration-title'>
            <Card className='registration-panel'>
                <header className='registration-panel__header'>
                    <BrandMark size='large' />
                    <div>
                        <Typography.Title id='registration-title' level={2}>
                            Choose a username
                        </Typography.Title>
                        <Typography.Paragraph type='secondary'>Your permanent public name in Athena</Typography.Paragraph>
                    </div>
                </header>

                {!registration && !loadError && (
                    <div className='registration-panel__loading' aria-live='polite'>
                        <Spin />
                        <Typography.Text type='secondary'>Loading your verified account…</Typography.Text>
                    </div>
                )}

                {loadError && (
                    <div className='registration-panel__failure'>
                        <Alert type='error' showIcon={true} title={loadError} />
                        <Button block={true} htmlType='button' loading={cancelling} disabled={cancelling} onClick={useAnotherGoogleAccount}>
                            Use another Google account
                        </Button>
                    </div>
                )}

                {registration && (
                    <>
                        <section className='registration-identity' aria-label='Verified identity'>
                            <div>
                                <Typography.Text type='secondary'>Verified Google account</Typography.Text>
                                <strong>{registration.verifiedEmail}</strong>
                            </div>
                            {registration.administrator && (
                                <Tag className='registration-identity__administrator' icon={<SafetyCertificateOutlined />} color='gold'>
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
                                <Button block={true} htmlType='button' loading={cancelling} disabled={busy} onClick={useAnotherGoogleAccount}>
                                    Use another Google account
                                </Button>
                            </div>
                        </form>
                    </>
                )}
            </Card>
        </main>
    );
};
