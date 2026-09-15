import {SafetyCertificateOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Typography} from 'antd';
import * as React from 'react';
import {useLocation} from 'react-router-dom';
import googleMark from '../../../assets/images/google-g.svg';
import phantomMark from '../../../assets/images/phantom-mark.svg';
import {readLoginReturnTo} from '../../shared/login-navigation';
import requests, {ACCOUNT_MAINTENANCE_MESSAGE, APPLICATION_REALM_HEADER, APPLICATION_REALM_QUERY} from '../../shared/services/requests';

type LoginMethod = 'google' | 'phantom';
type PhantomLoginReason =
    | 'phantom_not_installed'
    | 'phantom_cancelled'
    | 'phantom_busy'
    | 'phantom_state_invalid'
    | 'phantom_signature_invalid'
    | 'phantom_unavailable'
    | 'maintenance';

interface PhantomPublicKey {
    toString(): string;
}

interface PhantomProvider {
    isPhantom?: boolean;
    publicKey?: PhantomPublicKey | null;
    connect(): Promise<{publicKey: PhantomPublicKey}>;
    signMessage(message: Uint8Array, display?: 'utf8'): Promise<{signature: Uint8Array}>;
    on?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
    off?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
    removeListener?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
}

interface LoginAlert {
    type: 'error' | 'warning';
    message: string;
    installPhantom?: boolean;
}

const loginReasonAlerts: Record<string, LoginAlert> = {
    google_cancelled: {type: 'warning', message: 'Google sign-in was cancelled. Try again when you are ready.'},
    google_not_allowed: {type: 'error', message: 'This Google identity could not be used for Athena. Contact the administrator if the problem continues.'},
    google_state_invalid: {type: 'error', message: 'This Google sign-in request has expired or is no longer valid. Please try again.'},
    google_unavailable: {type: 'error', message: 'Google sign-in is temporarily unavailable. Please try again later.'},
    phantom_not_installed: {type: 'warning', message: 'Phantom is not installed in this browser.', installPhantom: true},
    phantom_cancelled: {type: 'warning', message: 'Phantom sign-in was cancelled. Try again when you are ready.'},
    phantom_busy: {type: 'warning', message: 'Phantom already has a request waiting for approval. Open Phantom to finish or cancel it.'},
    phantom_state_invalid: {type: 'error', message: 'This Phantom sign-in request expired, was already used, or the connected wallet changed. Please try again.'},
    phantom_signature_invalid: {type: 'error', message: 'Athena could not verify this Phantom signature. Please try signing again.'},
    phantom_unavailable: {type: 'error', message: 'Phantom sign-in is temporarily unavailable. Please try again later.'},
    maintenance: {type: 'warning', message: ACCOUNT_MAINTENANCE_MESSAGE}
};

const phantomInstallURL = 'https://phantom.com/download';
const phantomServerReasons = new Set<PhantomLoginReason>(['phantom_state_invalid', 'phantom_signature_invalid', 'phantom_unavailable', 'maintenance']);

class PhantomLoginError extends Error {
    public readonly reason: PhantomLoginReason;

    constructor(reason: PhantomLoginReason) {
        super(reason);
        this.name = 'PhantomLoginError';
        this.reason = reason;
    }
}

const isRecord = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === 'object';

const phantomProvider = (): PhantomProvider | undefined => {
    const provider = (window as Window & {phantom?: {solana?: PhantomProvider}}).phantom?.solana;
    return provider?.isPhantom ? provider : undefined;
};

const readJSON = async (response: Response): Promise<Record<string, unknown>> => {
    const text = await response.text();
    if (!text) {
        return {};
    }
    try {
        const value: unknown = JSON.parse(text);
        return isRecord(value) ? value : {};
    } catch {
        return {};
    }
};

const postPhantom = async (path: string, body: Record<string, unknown>, signal: AbortSignal): Promise<Record<string, unknown>> => {
    const response = await fetch(requests.toAbsURL(path), {
        method: 'POST',
        credentials: 'same-origin',
        headers: {'Accept': 'application/json', 'Content-Type': 'application/json', [APPLICATION_REALM_HEADER]: 'member'},
        body: JSON.stringify(body),
        signal
    });
    const responseBody = await readJSON(response);
    if (!response.ok) {
        const nestedError = isRecord(responseBody.error) ? responseBody.error : undefined;
        const reason = String(responseBody.reason || nestedError?.reason || '');
        throw new PhantomLoginError(phantomServerReasons.has(reason as PhantomLoginReason) ? (reason as PhantomLoginReason) : 'phantom_unavailable');
    }
    return responseBody;
};

const rawBase64URL = (bytes: Uint8Array): string =>
    window
        .btoa(String.fromCharCode(...bytes))
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '');

const phantomFailureReason = (error: unknown, accountChanged: boolean): PhantomLoginReason => {
    if (accountChanged) {
        return 'phantom_state_invalid';
    }
    if (error instanceof PhantomLoginError) {
        return error.reason;
    }
    const code = Number(isRecord(error) ? error.code : undefined);
    if (code === 4001) {
        return 'phantom_cancelled';
    }
    if (code === -32002) {
        return 'phantom_busy';
    }
    return 'phantom_unavailable';
};

export const LoginPage = () => {
    const location = useLocation();
    const [loadingMethod, setLoadingMethod] = React.useState<LoginMethod>();
    const [phantomStage, setPhantomStage] = React.useState('');
    const [localReason, setLocalReason] = React.useState<PhantomLoginReason>();
    const queryReason = new URLSearchParams(location.search).get('reason') || '';
    const alert = loginReasonAlerts[localReason || queryReason];
    const busy = Boolean(loadingMethod);

    const continueWithGoogle = () => {
        if (busy) {
            return;
        }
        setLocalReason(undefined);
        setPhantomStage('');
        setLoadingMethod('google');
        const query = new URLSearchParams({returnTo: readLoginReturnTo(location.search), [APPLICATION_REALM_QUERY]: 'member'});
        window.location.assign(`${requests.toAbsURL('/auth/google/login')}?${query.toString()}`);
    };

    const continueWithPhantom = async () => {
        if (busy) {
            return;
        }
        setLocalReason(undefined);
        setPhantomStage('');
        const provider = phantomProvider();
        if (!provider) {
            setLocalReason('phantom_not_installed');
            return;
        }

        setLoadingMethod('phantom');
        setPhantomStage('Connecting to Phantom…');
        const controller = new AbortController();
        let accountChanged = false;
        let address = '';
        const onAccountChanged = (publicKey: PhantomPublicKey | null) => {
            if (address && (!publicKey || publicKey.toString() !== address)) {
                accountChanged = true;
                controller.abort();
            }
        };
        const removeAccountListener = () => {
            try {
                if (provider.off) {
                    provider.off('accountChanged', onAccountChanged);
                } else {
                    provider.removeListener?.('accountChanged', onAccountChanged);
                }
            } catch {
                // Listener cleanup must not replace the safe sign-in outcome.
            }
        };

        try {
            const connection = await provider.connect();
            address = connection.publicKey?.toString() || '';
            if (!address) {
                throw new PhantomLoginError('phantom_unavailable');
            }
            provider.on?.('accountChanged', onAccountChanged);

            setPhantomStage('Preparing a secure sign-in message…');
            const challenge = await postPhantom('/auth/phantom/challenge', {address, returnTo: readLoginReturnTo(location.search)}, controller.signal);
            const message = typeof challenge.message === 'string' ? challenge.message : '';
            if (!message || accountChanged || provider.publicKey?.toString() !== address) {
                throw new PhantomLoginError(accountChanged ? 'phantom_state_invalid' : 'phantom_unavailable');
            }

            setPhantomStage('Approve the message in Phantom…');
            const signed = await provider.signMessage(new TextEncoder().encode(message), 'utf8');
            if (accountChanged || provider.publicKey?.toString() !== address) {
                throw new PhantomLoginError('phantom_state_invalid');
            }

            setPhantomStage('Verifying your wallet…');
            const verified = await postPhantom('/auth/phantom/verify', {signature: rawBase64URL(signed.signature)}, controller.signal);
            if (accountChanged) {
                throw new PhantomLoginError('phantom_state_invalid');
            }
            const redirectTo = typeof verified.redirectTo === 'string' ? verified.redirectTo : '/account/access';
            removeAccountListener();
            setPhantomStage('Opening Athena…');
            window.location.replace(requests.toAbsURL(redirectTo));
        } catch (error) {
            removeAccountListener();
            setLocalReason(phantomFailureReason(error, accountChanged));
            setPhantomStage('');
            setLoadingMethod(undefined);
        }
    };

    return (
        <main className='login-screen'>
            <div className='identity-brand'>
                <svg className='identity-brand-symbol' viewBox='0 0 28 28' aria-hidden='true'>
                    <path d='m4 23 10-19 10 19M8 17h12M11 23h6' />
                </svg>
                <span>ATHENA</span>
            </div>
            <Card className='login-panel'>
                <header className='identity-panel-heading'>
                    <Typography.Title level={1}>Welcome to Athena</Typography.Title>
                    <Typography.Paragraph type='secondary'>Sign in with your verified identity.</Typography.Paragraph>
                </header>
                <div className='login-panel__alerts' aria-live='polite' aria-atomic='true'>
                    {alert && (
                        <Alert
                            type={alert.type}
                            title={alert.message}
                            description={
                                alert.installPhantom ? (
                                    <a href={phantomInstallURL} target='_blank' rel='noopener noreferrer'>
                                        Install Phantom from the official website
                                    </a>
                                ) : undefined
                            }
                            showIcon={true}
                        />
                    )}
                </div>
                <div className='login-panel__actions' aria-busy={busy || undefined}>
                    <div className='login-method-buttons' role='group' aria-label='Sign-in methods'>
                        <Button
                            className='login-provider-button login-google-button'
                            block={true}
                            htmlType='button'
                            loading={loadingMethod === 'google'}
                            disabled={busy}
                            onClick={continueWithGoogle}>
                            {loadingMethod !== 'google' && <img src={googleMark} alt='' aria-hidden='true' />}
                            <span>Continue with Google</span>
                        </Button>
                        <Button
                            className='login-provider-button login-phantom-button'
                            block={true}
                            htmlType='button'
                            loading={loadingMethod === 'phantom'}
                            disabled={busy}
                            onClick={() => void continueWithPhantom()}>
                            {loadingMethod !== 'phantom' && <img src={phantomMark} alt='' aria-hidden='true' />}
                            <span>Continue with Phantom</span>
                        </Button>
                    </div>
                    <div className='login-panel__status' aria-live='polite' aria-atomic='true'>
                        {phantomStage}
                    </div>
                    <Typography.Paragraph className='login-panel__wallet-note' type='secondary'>
                        <SafetyCertificateOutlined aria-hidden='true' />
                        <span>
                            Phantom signs only this login message.
                            <br />
                            No transaction or network fee.
                        </span>
                    </Typography.Paragraph>
                    <div className='login-panel__new-account'>
                        <Typography.Text strong={true}>New to Athena?</Typography.Text>
                        <Typography.Paragraph type='secondary'>Verify your identity first, then choose a username to create your account.</Typography.Paragraph>
                    </div>
                </div>
            </Card>
            <Typography.Paragraph className='identity-access-note' type='secondary'>
                Business access is granted separately by an Athena administrator.
            </Typography.Paragraph>
        </main>
    );
};
