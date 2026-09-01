import {Avatar} from 'antd';
import {AccountIdentity, AccountIdentityProvider, AccountProfile, AccountTier} from './models';
import {realmBoundResourceURL} from './services/requests';

export const accountTierLabel = (tier: AccountTier) => (tier === AccountTier.Pro ? 'Pro' : 'Standard');

const accountInitials = (displayName: string, username: string) => (displayName || username || 'A').trim().slice(0, 1).toUpperCase();

const avatarURL = (url: string) => {
    if (!url) {
        return undefined;
    }
    return realmBoundResourceURL(url);
};

export const AccountAvatar = (props: {profile: AccountProfile; username: string; size?: number; className?: string}) => (
    <Avatar className={props.className} size={props.size || 40} src={avatarURL(props.profile.avatarUrl)}>
        {accountInitials(props.profile.displayName, props.username)}
    </Avatar>
);

export const identityProviderLabel = (provider: AccountIdentityProvider) => {
    switch (provider) {
        case AccountIdentityProvider.Google:
            return 'Google';
        case AccountIdentityProvider.SolanaWallet:
            return 'Phantom';
        case AccountIdentityProvider.Development:
            return 'Development';
        default:
            return 'Not available';
    }
};

export const identityPresentation = (identity: AccountIdentity) => {
    switch (identity.provider) {
        case AccountIdentityProvider.Google:
            return {
                label: 'Verified email',
                value: identity.verifiedEmail,
                pendingTitle: 'Your Google identity is verified',
                signInLabel: 'Google sign-in'
            };
        case AccountIdentityProvider.SolanaWallet:
            return {
                label: 'Solana address',
                value: identity.solanaAddress,
                pendingTitle: 'Your Phantom wallet ownership is verified',
                signInLabel: 'Phantom sign-in'
            };
        default:
            return {
                label: 'Identity',
                value: identity.verifiedEmail || identity.solanaAddress,
                pendingTitle: 'Your identity is verified',
                signInLabel: 'Sign-in'
            };
    }
};
