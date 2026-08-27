import {CopyOutlined, DeleteOutlined, EditOutlined, EyeOutlined, ImportOutlined, PlusOutlined, SafetyCertificateOutlined, UploadOutlined} from '@ant-design/icons';
import {
    Alert,
    Avatar,
    Button,
    Card,
    Checkbox,
    Descriptions,
    Drawer,
    Dropdown,
    Empty,
    Form,
    Input,
    Modal,
    Pagination,
    Skeleton,
    Space,
    Tag,
    Tooltip,
    Typography,
    Upload
} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {useLocation} from 'react-router-dom';
import {AppPage, AsyncState, ChoiceGroup, SearchBar, useAsyncData} from '../components';
import {AccountDataModule} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {AccountIdentityProvider} from '../shared/models';
import {SensitiveWriteScope, useSensitiveWriteLease} from '../shared/sensitive-write-scope';
import {services} from '../shared/services';
import {
    CreateWalletInput,
    CreateWalletResult,
    ListWalletsResult,
    WALLET_LOGIN_SESSION_REQUIRED,
    WALLET_REAUTH_REQUIRED,
    WALLET_REAUTH_UNAVAILABLE,
    WalletAvatarPresetID,
    WalletItem,
    WalletType,
    walletAvatarPresets,
    walletTypeOptions
} from '../shared/services/wallet-service';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {useKeywordParam, usePagedParams} from './shared';

const walletPageSizes = [12, 24, 48];
const pendingWalletSecretActionKey = 'athena.wallet-secret.pending-action';
const acceptedAvatarTypes = new Set(['image/jpeg', 'image/png', 'image/webp']);

const walletPresetDetails: Record<WalletAvatarPresetID, {label: string; glyph: string}> = {
    'star-violet': {label: 'Star violet', glyph: '★'},
    'bolt-blue': {label: 'Bolt blue', glyph: 'ϟ'},
    'gem-cyan': {label: 'Gem cyan', glyph: '◆'},
    'leaf-green': {label: 'Leaf green', glyph: '♧'},
    'sun-amber': {label: 'Sun amber', glyph: '☀'},
    'flame-orange': {label: 'Flame orange', glyph: '♨'},
    'heart-rose': {label: 'Heart rose', glyph: '♥'},
    'moon-indigo': {label: 'Moon indigo', glyph: '☾'}
};

const defaultAvatarPalettes = [
    ['#5b21b6', '#a78bfa'],
    ['#1d4ed8', '#60a5fa'],
    ['#0f766e', '#2dd4bf'],
    ['#166534', '#4ade80'],
    ['#a16207', '#fbbf24'],
    ['#c2410c', '#fb923c'],
    ['#be123c', '#fb7185'],
    ['#3730a3', '#818cf8']
];

const unicodeCharacterCount = (value: string) => Array.from(value).length;

const remarkValidationMessage = (value: string) => {
    const remark = value.trim();
    if (!remark) {
        return 'A wallet remark is required.';
    }
    if (unicodeCharacterCount(remark) > 50) {
        return 'Use 50 Unicode characters or fewer.';
    }
    return '';
};

const errorReason = (error: unknown) => requestErrorDetails(error).reason || '';

const walletErrorMessage = (error: unknown, fallback: string) => {
    const reason = errorReason(error);
    if (reason === WALLET_LOGIN_SESSION_REQUIRED) {
        return 'This operation requires an interactive Athena login session. API Keys cannot create, import, or reveal wallet keys.';
    }
    if (reason === WALLET_REAUTH_UNAVAILABLE) {
        return 'Private-key reauthentication is temporarily unavailable. No key material was returned.';
    }
    if (reason === WALLET_REAUTH_REQUIRED) {
        return fallback;
    }
    return requestErrorMessage(error, fallback);
};

const copyText = async (value: string) => {
    if (!value) {
        throw new Error('Nothing to copy');
    }
    await navigator.clipboard.writeText(value);
};

const hashWallet = (value: string) => {
    let hash = 2166136261;
    for (const character of value) {
        hash ^= character.codePointAt(0) || 0;
        hash = Math.imul(hash, 16777619);
    }
    return hash >>> 0;
};

const formatWalletTime = (value: string) => {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    return new Intl.DateTimeFormat(undefined, {dateStyle: 'medium', timeStyle: 'short'}).format(date);
};

const WalletAvatar = (props: {item: Pick<WalletItem, 'walletType' | 'address' | 'avatarKind' | 'avatarPresetId' | 'avatarUrl'>; size?: number}) => {
    const preset = walletPresetDetails[props.item.avatarPresetId as WalletAvatarPresetID];
    const palette = defaultAvatarPalettes[hashWallet(`${props.item.walletType}:${props.item.address}`) % defaultAvatarPalettes.length];
    const fallback = props.item.walletType === 'SOLANA' ? 'S' : 'E';
    const className = ['wallet-avatar', preset ? `wallet-avatar--${props.item.avatarPresetId}` : 'wallet-avatar--generated'].join(' ');
    const style = preset ? undefined : {background: `linear-gradient(145deg, ${palette[0]}, ${palette[1]})`};
    const uploaded = props.item.avatarKind === 'upload' && props.item.avatarUrl;
    return (
        <Avatar className={className} size={props.size || 52} src={uploaded || undefined} style={style}>
            {preset?.glyph || fallback}
        </Avatar>
    );
};

const GeneratedAvatarPreview = () => <Avatar className='wallet-avatar wallet-avatar--generated wallet-avatar--preview'>A</Avatar>;

const PresetAvatar = (props: {preset: WalletAvatarPresetID}) => {
    const detail = walletPresetDetails[props.preset];
    return <Avatar className={`wallet-avatar wallet-avatar--${props.preset}`}>{detail.glyph}</Avatar>;
};

const AvatarPresetPicker = (props: {value?: string; onChange?: (value: string) => void; disabled?: boolean; compact?: boolean}) => {
    const options = [{id: '', label: 'Generated'}, ...walletAvatarPresets.map(id => ({id, label: walletPresetDetails[id].label}))];
    return (
        <div className={props.compact ? 'wallet-avatar-picker wallet-avatar-picker--compact' : 'wallet-avatar-picker'} role='radiogroup' aria-label='Wallet avatar preset'>
            {options.map(option => {
                const selected = (props.value || '') === option.id;
                return (
                    <button
                        key={option.id || 'generated'}
                        type='button'
                        className={selected ? 'wallet-avatar-option wallet-avatar-option--selected' : 'wallet-avatar-option'}
                        role='radio'
                        aria-checked={selected}
                        disabled={props.disabled}
                        onClick={() => props.onChange?.(option.id)}>
                        {option.id ? <PresetAvatar preset={option.id as WalletAvatarPresetID} /> : <GeneratedAvatarPreview />}
                        <span>{option.label}</span>
                    </button>
                );
            })}
        </div>
    );
};

interface WalletFormValues {
    walletType: WalletType;
    remark: string;
    avatarPresetId?: string;
    privateKey?: string;
}

const WalletCreateImportModal = (props: {
    mode?: 'create' | 'import';
    submitting: boolean;
    onCancel: () => void;
    onSubmit: (mode: 'create' | 'import', values: WalletFormValues) => void;
}) => {
    const [form] = Form.useForm<WalletFormValues>();
    const walletType = Form.useWatch('walletType', form) || 'EVM';

    React.useEffect(() => {
        if (props.mode) {
            form.setFieldsValue({walletType: 'EVM', remark: '', avatarPresetId: '', privateKey: ''});
        } else {
            form.resetFields();
        }
    }, [form, props.mode]);

    const title = props.mode === 'import' ? 'Import Wallet' : 'Create Wallet';
    return (
        <Modal
            destroyOnHidden={true}
            open={Boolean(props.mode)}
            title={title}
            footer={null}
            maskClosable={!props.submitting}
            closable={!props.submitting}
            keyboard={!props.submitting}
            onCancel={props.onCancel}>
            <Form form={form} layout='vertical' preserve={false} requiredMark='optional' onFinish={values => props.mode && props.onSubmit(props.mode, values)}>
                <Form.Item name='walletType' label='Wallet type' rules={[{required: true, message: 'Choose a wallet type.'}]}>
                    <ChoiceGroup<WalletType> ariaLabel='Wallet type' className='choice-group--form' options={walletTypeOptions} disabled={props.submitting} />
                </Form.Item>
                <Form.Item
                    name='remark'
                    label='Remark'
                    validateTrigger={['onChange', 'onBlur']}
                    rules={[
                        {
                            validator: (_, value: string) => {
                                const message = remarkValidationMessage(value || '');
                                return message ? Promise.reject(new Error(message)) : Promise.resolve();
                            }
                        }
                    ]}
                    extra='Required. You can edit this later.'>
                    <Input
                        autoFocus={true}
                        placeholder='e.g. Treasury operations'
                        disabled={props.submitting}
                        showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/50`}}
                    />
                </Form.Item>
                {props.mode === 'import' && (
                    <Form.Item
                        name='privateKey'
                        label='Private key'
                        rules={[{required: true, whitespace: true, message: 'Enter the wallet private key.'}]}
                        extra={
                            walletType === 'SOLANA' ? 'Base58, a JSON byte array, or 32/64-byte hexadecimal material.' : 'A 32-byte hexadecimal private key, with or without 0x.'
                        }>
                        <Input.Password autoComplete='off' spellCheck={false} disabled={props.submitting} placeholder={walletType === 'SOLANA' ? 'Solana private key' : '0x…'} />
                    </Form.Item>
                )}
                <Form.Item name='avatarPresetId' label='Avatar' extra='Custom images can be uploaded after the wallet is saved.'>
                    <AvatarPresetPicker disabled={props.submitting} />
                </Form.Item>
                <div className='wallet-modal-actions'>
                    <Button disabled={props.submitting} onClick={props.onCancel}>
                        Cancel
                    </Button>
                    <Button type='primary' htmlType='submit' loading={props.submitting}>
                        {props.mode === 'import' ? 'Import wallet' : 'Create wallet'}
                    </Button>
                </div>
            </Form>
        </Modal>
    );
};

const WalletBackupModal = (props: {
    result?: CreateWalletResult;
    confirmed: boolean;
    onConfirmedChange: (value: boolean) => void;
    onDone: () => void;
    onCopy: (label: string, value: string) => void;
}) => (
    <Modal
        destroyOnHidden={true}
        open={Boolean(props.result)}
        title='Back Up Your Wallet'
        width={680}
        closable={false}
        keyboard={false}
        maskClosable={false}
        footer={
            <Button type='primary' disabled={!props.confirmed} onClick={props.onDone}>
                Done
            </Button>
        }>
        <Space direction='vertical' size='middle' className='wallet-secret-stack'>
            <Alert
                showIcon={true}
                type='warning'
                message='This is the only automatic display after creation'
                description='Store the private key somewhere secure. Anyone with this key can control the wallet. Athena will never ask you to share it.'
            />
            <div className='wallet-secret-field'>
                <Typography.Text type='secondary'>Address</Typography.Text>
                <Space.Compact block={true}>
                    <Input readOnly={true} value={props.result?.item.address || ''} />
                    <Tooltip title='Copy address'>
                        <Button aria-label='Copy wallet address' icon={<CopyOutlined />} onClick={() => props.onCopy('Address', props.result?.item.address || '')} />
                    </Tooltip>
                </Space.Compact>
            </div>
            <div className='wallet-secret-field'>
                <Typography.Text type='secondary'>Private key</Typography.Text>
                <Space.Compact block={true}>
                    <Input.Password readOnly={true} autoComplete='off' value={props.result?.privateKey || ''} />
                    <Tooltip title='Copy private key'>
                        <Button aria-label='Copy private key' icon={<CopyOutlined />} onClick={() => props.onCopy('Private key', props.result?.privateKey || '')} />
                    </Tooltip>
                </Space.Compact>
            </div>
            <Checkbox checked={props.confirmed} onChange={event => props.onConfirmedChange(event.target.checked)}>
                I have securely backed up this private key
            </Checkbox>
        </Space>
    </Modal>
);

interface RevealedWalletSecret {
    item: WalletItem;
    privateKey: string;
}

const WalletSecretModal = (props: {secret?: RevealedWalletSecret; onClose: () => void; onCopy: (label: string, value: string) => void}) => (
    <Modal
        destroyOnHidden={true}
        open={Boolean(props.secret)}
        title='View / Export Private Key'
        width={680}
        onCancel={props.onClose}
        footer={<Button onClick={props.onClose}>Close</Button>}>
        <Space direction='vertical' size='middle' className='wallet-secret-stack'>
            <Alert
                showIcon={true}
                type='warning'
                message='Keep this private key secret'
                description='Export means revealing or copying the key. Athena does not create a plaintext download file.'
            />
            <div className='wallet-secret-heading'>
                <WalletAvatar item={props.secret?.item || emptyWallet} size={46} />
                <span>
                    <strong>{props.secret?.item.remark}</strong>
                    <small>{props.secret?.item.walletType === 'SOLANA' ? 'Solana' : 'EVM'} wallet</small>
                </span>
            </div>
            <div className='wallet-secret-field'>
                <Typography.Text type='secondary'>Private key</Typography.Text>
                <Space.Compact block={true}>
                    <Input.Password readOnly={true} autoComplete='off' value={props.secret?.privateKey || ''} />
                    <Tooltip title='Copy private key'>
                        <Button aria-label='Copy private key' icon={<CopyOutlined />} onClick={() => props.onCopy('Private key', props.secret?.privateKey || '')} />
                    </Tooltip>
                </Space.Compact>
            </div>
        </Space>
    </Modal>
);

const emptyWallet: WalletItem = {
    id: 0,
    walletType: 'EVM',
    address: '',
    remark: '',
    source: 'created',
    avatarKind: 'default',
    avatarPresetId: '',
    avatarUrl: '',
    revision: 0,
    createdAt: '',
    updatedAt: ''
};

const WalletDetailDrawer = (props: {
    item?: WalletItem;
    canWrite: boolean;
    operation?: 'remark' | 'avatar';
    revealing: boolean;
    reauthStage?: string;
    onClose: () => void;
    onCopy: (label: string, value: string) => void;
    onSaveRemark?: (remark: string) => void;
    onAvatarPreset?: (presetID: string) => void;
    onAvatarUpload?: (file: File) => void;
    onAvatarReset?: () => void;
    onReveal?: () => void;
}) => {
    const [editingRemark, setEditingRemark] = React.useState(false);
    const [remark, setRemark] = React.useState('');
    const remarkError = remarkValidationMessage(remark);

    React.useEffect(() => {
        setEditingRemark(false);
        setRemark(props.item?.remark || '');
    }, [props.item?.id, props.item?.remark, props.item?.revision]);

    const uploadAvatar = (file: File) => {
        if (!acceptedAvatarTypes.has(file.type)) {
            props.onAvatarUpload?.(file);
            return Upload.LIST_IGNORE;
        }
        props.onAvatarUpload?.(file);
        return Upload.LIST_IGNORE;
    };

    const item = props.item;
    return (
        <Drawer
            rootClassName='wallet-detail-drawer'
            width={520}
            open={Boolean(item)}
            title='Wallet Details'
            onClose={props.onClose}
            extra={
                item ? (
                    <Tooltip title='Copy address'>
                        <Button type='text' aria-label='Copy wallet address' icon={<CopyOutlined />} onClick={() => props.onCopy('Address', item.address)} />
                    </Tooltip>
                ) : null
            }>
            {item && (
                <div className='wallet-detail'>
                    <div className='wallet-detail__identity'>
                        <WalletAvatar item={item} size={72} />
                        <div>
                            <Typography.Title level={3}>{item.remark}</Typography.Title>
                            <Space size='small' wrap={true}>
                                <Tag color={item.walletType === 'SOLANA' ? 'purple' : 'blue'}>{item.walletType === 'SOLANA' ? 'Solana' : 'EVM'}</Tag>
                                <Tag>{item.source === 'imported' ? 'Imported' : 'Created in Athena'}</Tag>
                            </Space>
                        </div>
                    </div>

                    <section className='wallet-detail__section' aria-labelledby='wallet-remark-heading'>
                        <div className='wallet-detail__section-heading'>
                            <Typography.Title id='wallet-remark-heading' level={4}>
                                Remark
                            </Typography.Title>
                            {props.canWrite && !editingRemark && (
                                <Button type='text' size='small' icon={<EditOutlined />} onClick={() => setEditingRemark(true)}>
                                    Edit
                                </Button>
                            )}
                        </div>
                        {editingRemark ? (
                            <Form
                                layout='vertical'
                                onFinish={() => {
                                    if (!remarkError) {
                                        props.onSaveRemark?.(remark.trim());
                                    }
                                }}>
                                <Form.Item validateStatus={remarkError ? 'error' : undefined} help={remarkError || 'Shown on your wallet card.'}>
                                    <Input
                                        autoFocus={true}
                                        value={remark}
                                        disabled={props.operation === 'remark'}
                                        showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/50`}}
                                        onChange={event => setRemark(event.target.value)}
                                    />
                                </Form.Item>
                                <Space>
                                    <Button
                                        disabled={props.operation === 'remark'}
                                        onClick={() => {
                                            setRemark(item.remark);
                                            setEditingRemark(false);
                                        }}>
                                        Cancel
                                    </Button>
                                    <Button type='primary' htmlType='submit' loading={props.operation === 'remark'} disabled={Boolean(remarkError)}>
                                        Save remark
                                    </Button>
                                </Space>
                            </Form>
                        ) : (
                            <Typography.Paragraph>{item.remark}</Typography.Paragraph>
                        )}
                    </section>

                    <section className='wallet-detail__section' aria-labelledby='wallet-details-heading'>
                        <Typography.Title id='wallet-details-heading' level={4}>
                            Details
                        </Typography.Title>
                        <Descriptions column={1} size='small'>
                            <Descriptions.Item label='Address'>
                                <Typography.Text className='wallet-address-full' copyable={true}>
                                    {item.address}
                                </Typography.Text>
                            </Descriptions.Item>
                            <Descriptions.Item label='Type'>{item.walletType === 'SOLANA' ? 'Solana' : 'EVM'}</Descriptions.Item>
                            <Descriptions.Item label='Source'>{item.source === 'imported' ? 'Imported' : 'Created in Athena'}</Descriptions.Item>
                            <Descriptions.Item label='Created'>{formatWalletTime(item.createdAt)}</Descriptions.Item>
                            <Descriptions.Item label='Updated'>{formatWalletTime(item.updatedAt)}</Descriptions.Item>
                        </Descriptions>
                    </section>

                    {props.canWrite && (
                        <section className='wallet-detail__section' aria-labelledby='wallet-avatar-heading'>
                            <Typography.Title id='wallet-avatar-heading' level={4}>
                                Avatar
                            </Typography.Title>
                            <AvatarPresetPicker
                                compact={true}
                                value={item.avatarKind === 'preset' ? item.avatarPresetId : item.avatarKind === 'default' ? '' : undefined}
                                disabled={props.operation === 'avatar'}
                                onChange={props.onAvatarPreset}
                            />
                            <Space wrap={true} className='wallet-avatar-actions'>
                                <Upload
                                    accept='image/jpeg,image/png,image/webp'
                                    showUploadList={false}
                                    beforeUpload={file => uploadAvatar(file as File)}
                                    disabled={props.operation === 'avatar'}>
                                    <Button icon={<UploadOutlined />} loading={props.operation === 'avatar'} disabled={props.operation === 'avatar'}>
                                        Upload image
                                    </Button>
                                </Upload>
                                {item.avatarKind !== 'default' && (
                                    <Button icon={<DeleteOutlined />} disabled={props.operation === 'avatar'} onClick={props.onAvatarReset}>
                                        Use generated
                                    </Button>
                                )}
                            </Space>
                            <Typography.Paragraph type='secondary' className='wallet-avatar-help'>
                                JPEG, PNG, or non-animated WebP. Maximum 2 MiB. Images are private to your account.
                            </Typography.Paragraph>
                        </section>
                    )}

                    <section className='wallet-detail__section wallet-detail__secret' aria-labelledby='wallet-secret-heading'>
                        <Typography.Title id='wallet-secret-heading' level={4}>
                            Private key
                        </Typography.Title>
                        <Typography.Paragraph type='secondary'>
                            Viewing or exporting requires a fresh identity check. The resulting five-minute lease can be reused only by your current login session.
                        </Typography.Paragraph>
                        {props.canWrite ? (
                            <>
                                <Button type='primary' danger={true} icon={<EyeOutlined />} loading={props.revealing} onClick={props.onReveal}>
                                    View / export private key
                                </Button>
                                <span className='wallet-reauth-stage' role='status' aria-live='polite'>
                                    {props.reauthStage}
                                </span>
                            </>
                        ) : (
                            <Alert type='info' showIcon={true} message='Read-only wallet access does not include private keys.' />
                        )}
                    </section>
                </div>
            )}
        </Drawer>
    );
};

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

const phantomProvider = (): PhantomProvider | undefined => {
    const provider = (window as Window & {phantom?: {solana?: PhantomProvider}}).phantom?.solana;
    return provider?.isPhantom ? provider : undefined;
};

const rawBase64URL = (bytes: Uint8Array) =>
    window
        .btoa(String.fromCharCode(...bytes))
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '');

interface PendingSecretAction {
    action: 'reveal';
    walletId: number;
}

const readPendingSecretAction = (): PendingSecretAction | undefined => {
    const raw = window.sessionStorage.getItem(pendingWalletSecretActionKey);
    if (!raw) {
        return undefined;
    }
    window.sessionStorage.removeItem(pendingWalletSecretActionKey);
    try {
        const value = JSON.parse(raw) as Partial<PendingSecretAction>;
        const walletId = Number(value.walletId);
        return value.action === 'reveal' && Number.isInteger(walletId) && walletId > 0 ? {action: 'reveal', walletId} : undefined;
    } catch {
        return undefined;
    }
};

interface WalletWriteHandle {
    openCreate(): void;
    openImport(): void;
}

const WalletWriteSurface = React.forwardRef<
    WalletWriteHandle,
    {
        selected?: WalletItem;
        onSelectedChange: (item?: WalletItem) => void;
        onWalletChanged: (item: WalletItem) => void;
        onReload: () => void;
    }
>(function WalletWriteSurface(props, ref) {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [editorMode, setEditorMode] = React.useState<'create' | 'import'>();
    const [submitting, setSubmitting] = React.useState(false);
    const [backup, setBackup] = React.useState<CreateWalletResult>();
    const [backupConfirmed, setBackupConfirmed] = React.useState(false);
    const [secret, setSecret] = React.useState<RevealedWalletSecret>();
    const [revealingID, setRevealingID] = React.useState<number>();
    const [reauthStage, setReauthStage] = React.useState('');
    const [operation, setOperation] = React.useState<'remark' | 'avatar'>();
    const resumedRef = React.useRef(false);
    const identityKey = `${authorization.user.iss}:${authorization.user.accountId}:${authorization.revision}`;

    React.useImperativeHandle(
        ref,
        () => ({
            openCreate: () => setEditorMode('create'),
            openImport: () => setEditorMode('import')
        }),
        []
    );

    React.useEffect(() => {
        setEditorMode(undefined);
        setSubmitting(false);
        setBackup(undefined);
        setBackupConfirmed(false);
        setSecret(undefined);
        setRevealingID(undefined);
        setReauthStage('');
        setOperation(undefined);
    }, [identityKey]);

    React.useEffect(
        () => () => {
            setSecret(undefined);
            setBackup(undefined);
        },
        []
    );

    const notifyCopy = React.useCallback(
        async (label: string, value: string) => {
            try {
                await copyText(value);
                ctx.notifications.success(`${label} copied`);
            } catch {
                ctx.notifications.error(`Could not copy ${label.toLowerCase()}`, 'Select the value and copy it manually.');
            }
        },
        [ctx.notifications]
    );

    const establishSolanaLease = React.useCallback(async () => {
        const provider = phantomProvider();
        if (!provider) {
            ctx.notifications.error('Phantom is required', 'Install or enable Phantom in this browser to approve private-key access.');
            return false;
        }
        const expectedAddress = authorization.user.identity.solanaAddress;
        let address = '';
        let accountChanged = false;
        const onAccountChanged = (publicKey: PhantomPublicKey | null) => {
            if (address && (!publicKey || publicKey.toString() !== address)) {
                accountChanged = true;
            }
        };
        const removeListener = () => {
            try {
                if (provider.off) {
                    provider.off('accountChanged', onAccountChanged);
                } else {
                    provider.removeListener?.('accountChanged', onAccountChanged);
                }
            } catch {
                // Listener cleanup cannot make a completed signature trustworthy again.
            }
        };

        try {
            setReauthStage('Connecting to Phantom…');
            const connection = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
            address = connection.publicKey?.toString() || '';
            if (!address || address !== expectedAddress) {
                throw new Error('Connect the same Phantom account that you use to sign in to Athena.');
            }
            provider.on?.('accountChanged', onAccountChanged);

            setReauthStage('Preparing a private-key access message…');
            const challenge = await lease.runTask(() => services.wallet.createSolanaWalletSecretChallenge());
            if (challenge.status === 'discarded') {
                return false;
            }
            if (challenge.status === 'rejected') {
                throw challenge.error;
            }
            if (!challenge.value.message) {
                throw new Error('Athena returned an empty reauthentication message.');
            }

            setReauthStage('Approve the message in Phantom. No transaction or fee is involved…');
            const signed = await provider.signMessage(new TextEncoder().encode(challenge.value.message), 'utf8');
            if (accountChanged || provider.publicKey?.toString() !== expectedAddress) {
                throw new Error('The connected Phantom account changed before verification completed.');
            }

            setReauthStage('Verifying the signature…');
            const verified = await lease.runTask(() => services.wallet.verifySolanaWalletSecretSignature(rawBase64URL(signed.signature)));
            if (verified.status === 'discarded') {
                return false;
            }
            if (verified.status === 'rejected') {
                throw verified.error;
            }
            return true;
        } catch (error) {
            ctx.notifications.error('Could not confirm wallet identity', walletErrorMessage(error, error instanceof Error ? error.message : 'Phantom verification failed.'));
            return false;
        } finally {
            removeListener();
        }
    }, [authorization.user.identity.solanaAddress, ctx.notifications, lease]);

    const establishDevelopmentLease = React.useCallback(async () => {
        setReauthStage('Confirming the local development session…');
        const result = await lease.runTask(() => services.wallet.createDevelopmentWalletSecretLease());
        if (result.status === 'fulfilled') {
            return true;
        }
        if (result.status === 'rejected') {
            ctx.notifications.error('Could not confirm development session', walletErrorMessage(result.error, 'Local private-key access was denied.'));
        }
        return false;
    }, [ctx.notifications, lease]);

    const beginGoogleReauthentication = React.useCallback((id: number) => {
        const action: PendingSecretAction = {action: 'reveal', walletId: id};
        window.sessionStorage.setItem(pendingWalletSecretActionKey, JSON.stringify(action));
        setReauthStage('Opening Google for a fresh identity check…');
        window.location.assign(services.wallet.googleWalletSecretReauthenticationURL('/wallet'));
    }, []);

    const resolveWalletItem = React.useCallback(
        async (id: number) => {
            if (props.selected?.id === id) {
                return props.selected;
            }
            const result = await lease.runTask(() => services.wallet.getWallet(id));
            return result.status === 'fulfilled' ? result.value : undefined;
        },
        [lease, props.selected]
    );

    const reveal = React.useCallback(
        async (id: number, resumed = false) => {
            if (revealingID !== undefined) {
                return;
            }
            setSecret(undefined);
            setRevealingID(id);
            setReauthStage('Requesting private-key access…');

            const requestSecret = () => lease.runTask(() => services.wallet.revealPrivateKey(id));
            let result = await requestSecret();
            if (result.status === 'rejected' && errorReason(result.error) === WALLET_REAUTH_REQUIRED && !resumed) {
                switch (authorization.user.identity.provider) {
                    case AccountIdentityProvider.Google:
                        beginGoogleReauthentication(id);
                        return;
                    case AccountIdentityProvider.SolanaWallet:
                        if (await establishSolanaLease()) {
                            setReauthStage('Loading the private key…');
                            result = await requestSecret();
                        } else {
                            setRevealingID(undefined);
                            setReauthStage('');
                            return;
                        }
                        break;
                    case AccountIdentityProvider.Development:
                        if (await establishDevelopmentLease()) {
                            setReauthStage('Loading the private key…');
                            result = await requestSecret();
                        } else {
                            setRevealingID(undefined);
                            setReauthStage('');
                            return;
                        }
                        break;
                    default:
                        ctx.notifications.error('Reauthentication is unavailable', 'This login identity cannot approve private-key access.');
                        setRevealingID(undefined);
                        setReauthStage('');
                        return;
                }
            }

            if (result.status === 'fulfilled') {
                const item = await resolveWalletItem(id);
                if (!item) {
                    ctx.notifications.error('Could not load wallet details', 'The key was not kept in the page. Open the wallet and try again.');
                } else {
                    setSecret({item, privateKey: result.value});
                }
            } else if (result.status === 'rejected') {
                const callbackReason = new URLSearchParams(location.search).get('walletSecretReason');
                const fallback = resumed && callbackReason ? `Google reauthentication did not complete (${callbackReason}).` : 'The private key request failed.';
                ctx.notifications.error('Could not reveal private key', walletErrorMessage(result.error, fallback));
            }
            setRevealingID(undefined);
            setReauthStage('');
        },
        [
            authorization.user.identity.provider,
            beginGoogleReauthentication,
            ctx.notifications,
            establishDevelopmentLease,
            establishSolanaLease,
            lease,
            location.search,
            resolveWalletItem,
            revealingID
        ]
    );

    React.useEffect(() => {
        if (resumedRef.current) {
            return;
        }
        resumedRef.current = true;
        const pending = readPendingSecretAction();
        if (pending?.action === 'reveal') {
            void reveal(pending.walletId, true);
        }
    }, [reveal]);

    const submitWallet = async (mode: 'create' | 'import', values: WalletFormValues) => {
        setSubmitting(true);
        const input: CreateWalletInput = {
            walletType: values.walletType,
            remark: values.remark.trim(),
            avatarPresetId: values.avatarPresetId || ''
        };
        const result =
            mode === 'create'
                ? await lease.runTask(() => services.wallet.createWallet(input))
                : await lease.runTask(() => services.wallet.importWallet({...input, privateKey: values.privateKey?.trim() || ''}));
        if (result.status === 'discarded') {
            return;
        }
        setSubmitting(false);
        if (result.status === 'rejected') {
            ctx.notifications.error(mode === 'create' ? 'Wallet creation failed' : 'Wallet import failed', walletErrorMessage(result.error, 'Could not save this wallet.'));
            return;
        }
        setEditorMode(undefined);
        props.onReload();
        if (mode === 'create') {
            setBackupConfirmed(false);
            setBackup(result.value as CreateWalletResult);
        } else {
            const item = result.value as WalletItem;
            props.onWalletChanged(item);
            props.onSelectedChange(item);
            ctx.notifications.success('Wallet imported');
        }
    };

    const completeBackup = () => {
        if (!backup || !backupConfirmed) {
            return;
        }
        const item = backup.item;
        setBackup(undefined);
        setBackupConfirmed(false);
        props.onWalletChanged(item);
        props.onSelectedChange(item);
        ctx.notifications.success('Wallet created');
    };

    const applyWalletUpdate = async (kind: 'remark' | 'avatar', start: () => ReturnType<typeof services.wallet.updateRemark>, successMessage: string) => {
        setOperation(kind);
        const result = await lease.runTask(start);
        if (result.status === 'discarded') {
            return;
        }
        setOperation(undefined);
        if (result.status === 'fulfilled') {
            props.onWalletChanged(result.value);
            props.onReload();
            ctx.notifications.success(successMessage);
            return;
        }
        if (requestErrorDetails(result.error).status === 409 && props.selected) {
            const latest = await lease.runTask(() => services.wallet.getWallet(props.selected!.id));
            if (latest.status === 'fulfilled') {
                props.onWalletChanged(latest.value);
            }
            props.onReload();
            ctx.notifications.warning('Wallet changed elsewhere', 'The latest wallet details were loaded. Review them and try again.');
            return;
        }
        ctx.notifications.error(kind === 'remark' ? 'Could not update remark' : 'Could not update avatar', walletErrorMessage(result.error, 'The wallet update failed.'));
    };

    const saveRemark = (remark: string) => {
        if (!props.selected) {
            return;
        }
        void applyWalletUpdate('remark', () => services.wallet.updateRemark(props.selected!.id, remark, props.selected!.revision), 'Wallet remark updated');
    };

    const setAvatarPreset = (presetID: string) => {
        const item = props.selected;
        if (!item || (item.avatarKind === 'preset' && item.avatarPresetId === presetID) || (item.avatarKind === 'default' && !presetID)) {
            return;
        }
        if (!presetID) {
            void applyWalletUpdate('avatar', () => services.wallet.deleteAvatar(item.id, item.revision), 'Generated avatar restored');
            return;
        }
        void applyWalletUpdate('avatar', () => services.wallet.updateAvatarPreset(item.id, presetID, item.revision), 'Wallet avatar updated');
    };

    const uploadAvatar = (file: File) => {
        const item = props.selected;
        if (!item) {
            return;
        }
        if (!acceptedAvatarTypes.has(file.type)) {
            ctx.notifications.error('Unsupported avatar format', 'Choose a JPEG, PNG, or WebP image.');
            return;
        }
        if (file.size > 2 * 1024 * 1024) {
            ctx.notifications.error('Avatar is too large', 'Choose an image up to 2 MiB.');
            return;
        }
        void applyWalletUpdate('avatar', () => services.wallet.uploadAvatar(item.id, file, item.revision), 'Wallet avatar updated');
    };

    const resetAvatar = () => {
        const item = props.selected;
        if (item) {
            void applyWalletUpdate('avatar', () => services.wallet.deleteAvatar(item.id, item.revision), 'Generated avatar restored');
        }
    };

    return (
        <>
            <WalletDetailDrawer
                item={props.selected}
                canWrite={true}
                operation={operation}
                revealing={revealingID === props.selected?.id}
                reauthStage={reauthStage}
                onClose={() => props.onSelectedChange(undefined)}
                onCopy={(label, value) => void notifyCopy(label, value)}
                onSaveRemark={saveRemark}
                onAvatarPreset={setAvatarPreset}
                onAvatarUpload={uploadAvatar}
                onAvatarReset={resetAvatar}
                onReveal={() => props.selected && void reveal(props.selected.id)}
            />
            <WalletCreateImportModal mode={editorMode} submitting={submitting} onCancel={() => !submitting && setEditorMode(undefined)} onSubmit={submitWallet} />
            <WalletBackupModal
                result={backup}
                confirmed={backupConfirmed}
                onConfirmedChange={setBackupConfirmed}
                onDone={completeBackup}
                onCopy={(label, value) => void notifyCopy(label, value)}
            />
            <WalletSecretModal secret={secret} onClose={() => setSecret(undefined)} onCopy={(label, value) => void notifyCopy(label, value)} />
        </>
    );
});

const WalletCard = (props: {item: WalletItem; selected: boolean; onOpen: () => void; onCopy: () => void}) => {
    const openOnKeyboard = (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.target !== event.currentTarget) {
            return;
        }
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            props.onOpen();
        }
    };
    return (
        <Card
            className={props.selected ? 'wallet-card wallet-card--selected' : 'wallet-card'}
            role='button'
            tabIndex={0}
            aria-label={`Open ${props.item.remark} wallet`}
            aria-current={props.selected || undefined}
            onClick={props.onOpen}
            onKeyDown={openOnKeyboard}>
            <div className='wallet-card__topline'>
                <WalletAvatar item={props.item} size={54} />
                <div className='wallet-card__title'>
                    <Typography.Text strong={true}>{props.item.remark}</Typography.Text>
                    <Tag color={props.item.walletType === 'SOLANA' ? 'purple' : 'blue'}>{props.item.walletType === 'SOLANA' ? 'Solana' : 'EVM'}</Tag>
                </div>
            </div>
            <div className='wallet-card__address'>
                <Typography.Text ellipsis={{tooltip: props.item.address}}>{props.item.address}</Typography.Text>
                <Tooltip title='Copy address'>
                    <Button
                        type='text'
                        size='small'
                        aria-label={`Copy ${props.item.remark} address`}
                        icon={<CopyOutlined />}
                        onClick={event => {
                            event.stopPropagation();
                            props.onCopy();
                        }}
                    />
                </Tooltip>
            </div>
            <div className='wallet-card__footer'>
                <span>{props.item.source === 'imported' ? 'Imported' : 'Created in Athena'}</span>
                <span aria-hidden='true'>Open details →</span>
            </div>
        </Card>
    );
};

const WalletGrid = (props: {data: AsyncState<ListWalletsResult>; selected?: WalletItem; onOpen: (item: WalletItem) => void; onCopy: (item: WalletItem) => void}) => {
    const items = props.data.data?.items || [];
    if (props.data.loading && !props.data.data) {
        return (
            <div className='wallet-grid' aria-label='Loading wallets'>
                {Array.from({length: 6}, (_, index) => (
                    <Card className='wallet-card wallet-card--loading' key={index}>
                        <Skeleton active={true} avatar={true} paragraph={{rows: 2}} />
                    </Card>
                ))}
            </div>
        );
    }
    if (!items.length) {
        return (
            <div className='wallet-empty'>
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No wallets match these filters.' />
            </div>
        );
    }
    return (
        <div className={props.data.loading ? 'wallet-grid wallet-grid--refreshing' : 'wallet-grid'}>
            {items.map(item => (
                <WalletCard key={item.id} item={item} selected={props.selected?.id === item.id} onOpen={() => props.onOpen(item)} onCopy={() => props.onCopy(item)} />
            ))}
        </div>
    );
};

export const WalletsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.Wallet);
    const writeHandle = React.useRef<WalletWriteHandle>(null);
    const {page, pageSize, setPage} = usePagedParams(12, walletPageSizes);
    const [query, setQuery] = useKeywordParam();
    const [walletType, setWalletType] = React.useState<WalletType>();
    const [selected, setSelected] = React.useState<WalletItem>();
    const data = useAsyncData(
        () => services.wallet.listWallets({page, pageSize, query: query.trim(), walletType}),
        [page, pageSize, query, walletType, authorization.user.accountId]
    );

    React.useEffect(() => {
        if (!selected || !data.data) {
            return;
        }
        const next = data.data.items.find(item => item.id === selected.id);
        if (next && next.revision >= selected.revision) {
            setSelected(next);
        }
    }, [data.data, selected?.id, selected?.revision]);

    React.useEffect(() => {
        setSelected(undefined);
        if (!canWrite) {
            window.sessionStorage.removeItem(pendingWalletSecretActionKey);
        }
    }, [authorization.user.accountId, authorization.user.iss, canWrite]);

    const notifyCopy = async (item: WalletItem) => {
        try {
            await copyText(item.address);
            ctx.notifications.success('Address copied');
        } catch {
            ctx.notifications.error('Could not copy address', 'Open the wallet details and copy the address manually.');
        }
    };

    const selectWalletType = (value: string) => {
        setWalletType(value === 'all' ? undefined : (value as WalletType));
        setPage(1, pageSize);
    };

    const updateSelectedWallet = (item: WalletItem) => {
        setSelected(current => (current?.id === item.id ? item : current));
    };

    const writeMenu: MenuProps['items'] = [
        {key: 'create', label: 'Create wallet', icon: <PlusOutlined />},
        {key: 'import', label: 'Import wallet', icon: <ImportOutlined />}
    ];
    const writeActions = canWrite ? (
        <>
            <Space className='wallet-header-actions wallet-header-actions--desktop'>
                <Button icon={<ImportOutlined />} onClick={() => writeHandle.current?.openImport()}>
                    Import
                </Button>
                <Button type='primary' icon={<PlusOutlined />} onClick={() => writeHandle.current?.openCreate()}>
                    Create Wallet
                </Button>
            </Space>
            <Dropdown
                menu={{
                    items: writeMenu,
                    onClick: info => (info.key === 'create' ? writeHandle.current?.openCreate() : writeHandle.current?.openImport())
                }}
                trigger={['click']}>
                <Button className='wallet-header-actions--mobile' type='primary' icon={<PlusOutlined />} aria-label='Wallet actions' />
            </Dropdown>
        </>
    ) : null;

    return (
        <AppPage
            title='Wallets'
            subtitle='Create and manage your private EVM and Solana wallets. Only your account can access them.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={writeActions}
            filters={
                <div className='wallet-filters'>
                    <SearchBar value={query} onChange={setQuery} placeholder='Search remark or address' />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by wallet type'
                        value={walletType || 'all'}
                        options={[{label: 'All', value: 'all'}, ...walletTypeOptions]}
                        onChange={selectWalletType}
                    />
                </div>
            }>
            <div className='wallet-page-summary'>
                <span>
                    <SafetyCertificateOutlined /> Wallet keys are encrypted at rest
                </span>
                {data.data && <Typography.Text type='secondary'>{data.data.total} wallets</Typography.Text>}
            </div>
            <WalletGrid data={data} selected={selected} onOpen={setSelected} onCopy={item => void notifyCopy(item)} />
            {(data.data?.total || 0) > pageSize && (
                <Pagination
                    className='wallet-pagination'
                    current={page}
                    pageSize={pageSize}
                    total={data.data?.total || 0}
                    pageSizeOptions={walletPageSizes}
                    showSizeChanger={true}
                    showTotal={total => `${total} wallets`}
                    onChange={setPage}
                />
            )}
            {canWrite ? (
                <SensitiveWriteScope module={AccountDataModule.Wallet}>
                    <WalletWriteSurface ref={writeHandle} selected={selected} onSelectedChange={setSelected} onWalletChanged={updateSelectedWallet} onReload={data.reload} />
                </SensitiveWriteScope>
            ) : (
                <WalletDetailDrawer
                    item={selected}
                    canWrite={false}
                    revealing={false}
                    onClose={() => setSelected(undefined)}
                    onCopy={(label, value) => {
                        void copyText(value).then(
                            () => ctx.notifications.success(`${label} copied`),
                            () => ctx.notifications.error(`Could not copy ${label.toLowerCase()}`)
                        );
                    }}
                />
            )}
        </AppPage>
    );
};
