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
    InputNumber,
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
import {AppPage, AsyncState, ChoiceGroup, SearchBar, useAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {AccountIdentityProvider} from '../../shared/models';
import {SensitiveWriteScope, useSensitiveWriteLease} from '../../shared/sensitive-write-scope';
import {memberServices as services} from '../services';
import {
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
} from '../../shared/services/wallet-service';
import {realmBoundResourceURL, requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {useKeywordParam, usePagedParams} from '../../shared/pages/shared';

const walletPageSizes = [12, 24, 48];
const maxWalletBatchSize = 10;
const pendingWalletSecretActionKey = 'athena.member.wallet-secret.pending-action';
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

const remarkValidationMessage = (value: string, required = true) => {
    const remark = value.trim();
    if (!remark) {
        return required ? 'A wallet remark is required.' : '';
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

interface ImportedPrivateKeyLine {
    privateKey: string;
    lineNumber: number;
}

const parseImportedPrivateKeys = (value: string): ImportedPrivateKeyLine[] =>
    value
        .split(/\r\n|\n|\r/)
        .map((line, index) => ({privateKey: line.trim(), lineNumber: index + 1}))
        .filter(line => Boolean(line.privateKey));

const findPrivateKeyFieldIndex = (value: unknown, depth = 0): number | undefined => {
    if (depth > 6 || !value) {
        return undefined;
    }
    if (typeof value === 'string') {
        const match = value.match(/privateKeys\[(\d+)]/);
        return match ? Number(match[1]) : undefined;
    }
    if (Array.isArray(value)) {
        for (const item of value) {
            const index = findPrivateKeyFieldIndex(item, depth + 1);
            if (index !== undefined) {
                return index;
            }
        }
        return undefined;
    }
    if (typeof value !== 'object') {
        return undefined;
    }
    for (const nested of Object.values(value as Record<string, unknown>)) {
        const index = findPrivateKeyFieldIndex(nested, depth + 1);
        if (index !== undefined) {
            return index;
        }
    }
    return undefined;
};

const importWalletErrorMessage = (error: unknown, lines: ImportedPrivateKeyLine[], fallback: string) => {
    const message = walletErrorMessage(error, fallback);
    const messageMatch = message.match(/privateKeys\[(\d+)]/);
    const errorRecord = error && typeof error === 'object' ? (error as Record<string, unknown>) : {};
    const response = errorRecord.response && typeof errorRecord.response === 'object' ? (errorRecord.response as Record<string, unknown>) : {};
    const index = messageMatch ? Number(messageMatch[1]) : findPrivateKeyFieldIndex(response.body) ?? findPrivateKeyFieldIndex(errorRecord.body);
    const sourceLine = index === undefined ? undefined : lines[index];
    if (!sourceLine) {
        return message;
    }
    if (messageMatch) {
        return message.replace(/privateKeys\[(\d+)]/g, (field, rawIndex: string) => {
            const line = lines[Number(rawIndex)];
            return line ? `Line ${line.lineNumber}` : field;
        });
    }
    return `Line ${sourceLine.lineNumber}: ${message}`;
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
    const uploaded = props.item.avatarKind === 'upload' && realmBoundResourceURL(props.item.avatarUrl);
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
    count?: number;
    remark?: string;
    avatarPresetId?: string;
    privateKeys?: string;
}

const WalletCreateImportModal = (props: {
    mode?: 'create' | 'import';
    submitting: boolean;
    onCancel: () => void;
    onSubmit: (mode: 'create' | 'import', values: WalletFormValues) => void;
}) => {
    const [form] = Form.useForm<WalletFormValues>();
    const walletType = Form.useWatch('walletType', form) || 'EVM';
    const createCount = Form.useWatch('count', form) || 1;
    const privateKeys = Form.useWatch('privateKeys', form) || '';
    const importedLines = React.useMemo(() => parseImportedPrivateKeys(privateKeys), [privateKeys]);
    const batchSize = props.mode === 'import' ? importedLines.length : createCount;
    const isBatch = batchSize > 1;

    React.useEffect(() => {
        if (props.mode) {
            form.setFieldsValue({walletType: 'EVM', count: 1, remark: '', avatarPresetId: '', privateKeys: ''});
        } else {
            form.resetFields();
        }
    }, [form, props.mode]);

    React.useEffect(() => {
        if (isBatch && form.getFieldValue('remark')) {
            form.setFieldValue('remark', '');
        }
    }, [form, isBatch]);

    const title = props.mode === 'import' ? 'Import Wallets' : 'Create Wallets';
    const submitLabel =
        props.mode === 'import'
            ? importedLines.length
                ? `Import ${importedLines.length} ${importedLines.length === 1 ? 'wallet' : 'wallets'}`
                : 'Import wallets'
            : `Create ${createCount} ${createCount === 1 ? 'wallet' : 'wallets'}`;
    return (
        <Modal
            destroyOnHidden={true}
            rootClassName='wallet-create-import-modal'
            open={Boolean(props.mode)}
            title={title}
            width={600}
            footer={null}
            mask={{closable: !props.submitting}}
            closable={!props.submitting}
            keyboard={!props.submitting}
            onCancel={props.onCancel}>
            <Form form={form} layout='vertical' preserve={false} requiredMark='optional' onFinish={values => props.mode && props.onSubmit(props.mode, values)}>
                <Form.Item name='walletType' label='Wallet type' rules={[{required: true, message: 'Choose a wallet type.'}]}>
                    <ChoiceGroup<WalletType> ariaLabel='Wallet type' className='choice-group--form' options={walletTypeOptions} disabled={props.submitting} />
                </Form.Item>
                {props.mode === 'create' && (
                    <Form.Item
                        name='count'
                        label='Quantity'
                        rules={[
                            {required: true, message: 'Enter how many wallets to create.'},
                            {type: 'number', min: 1, max: maxWalletBatchSize, message: `Create between 1 and ${maxWalletBatchSize} wallets at a time.`},
                            {
                                validator: (_, value?: number) =>
                                    value === undefined || Number.isInteger(value) ? Promise.resolve() : Promise.reject(new Error('Quantity must be a whole number.'))
                            }
                        ]}
                        extra={
                            <span className='wallet-batch-count' aria-live='polite' aria-atomic='true'>
                                {createCount} of {maxWalletBatchSize} wallets selected.
                            </span>
                        }>
                        <InputNumber autoFocus={true} min={1} max={maxWalletBatchSize} precision={0} step={1} disabled={props.submitting} />
                    </Form.Item>
                )}
                {props.mode === 'import' && (
                    <Form.Item
                        name='privateKeys'
                        label='Private keys'
                        validateTrigger={['onChange', 'onBlur']}
                        rules={[
                            {
                                validator: (_, value?: string) => {
                                    const lines = parseImportedPrivateKeys(value || '');
                                    if (!lines.length) {
                                        return Promise.reject(new Error('Enter at least one private key.'));
                                    }
                                    if (lines.length > maxWalletBatchSize) {
                                        return Promise.reject(new Error(`Import up to ${maxWalletBatchSize} private keys at a time.`));
                                    }
                                    return Promise.resolve();
                                }
                            }
                        ]}
                        extra={
                            <span className='wallet-private-keys-help'>
                                <span id='wallet-private-keys-guidance'>
                                    One private key per line; blank lines are ignored. Keys remain visible in this dialog, so protect your screen.{' '}
                                    {walletType === 'SOLANA'
                                        ? 'Keep each Base58 key, JSON byte array, or 32/64-byte hexadecimal value on one line.'
                                        : 'Use one 32-byte hexadecimal private key per line, with or without 0x.'}
                                </span>
                                <span
                                    id='wallet-private-keys-count'
                                    className={
                                        importedLines.length > maxWalletBatchSize ? 'wallet-private-keys-count wallet-private-keys-count--error' : 'wallet-private-keys-count'
                                    }
                                    aria-live='polite'
                                    aria-atomic='true'>
                                    {importedLines.length} of {maxWalletBatchSize} private keys ready.
                                </span>
                            </span>
                        }>
                        <Input.TextArea
                            className='wallet-private-keys-input'
                            autoFocus={true}
                            autoComplete='off'
                            autoCapitalize='none'
                            autoCorrect='off'
                            spellCheck={false}
                            wrap='off'
                            rows={8}
                            disabled={props.submitting}
                            aria-describedby='wallet-private-keys-guidance wallet-private-keys-count'
                            placeholder={walletType === 'SOLANA' ? 'One Solana private key per line' : '0x…\n0x…'}
                        />
                    </Form.Item>
                )}
                <Form.Item
                    name='remark'
                    label='Remark'
                    validateTrigger={['onChange', 'onBlur']}
                    rules={[
                        {
                            validator: (_, value?: string) => {
                                if (isBatch && value?.trim()) {
                                    return Promise.reject(new Error('Batch wallets use automatic sequential remarks.'));
                                }
                                const message = remarkValidationMessage(value || '', false);
                                return message ? Promise.reject(new Error(message)) : Promise.resolve();
                            }
                        }
                    ]}
                    extra={
                        isBatch
                            ? `Automatic sequential names will be assigned to all ${batchSize} wallets.`
                            : 'Optional. Leave blank to use the next default name, such as EVM-1 or SOL-1. You can edit it later.'
                    }>
                    <Input placeholder='Optional custom remark' disabled={props.submitting || isBatch} showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/50`}} />
                </Form.Item>
                <Form.Item
                    name='avatarPresetId'
                    label='Avatar'
                    extra={
                        isBatch
                            ? 'This avatar is shared by every wallet in this batch. Custom images can be uploaded later.'
                            : 'Custom images can be uploaded after the wallet is saved.'
                    }>
                    <AvatarPresetPicker disabled={props.submitting} />
                </Form.Item>
                <div className='wallet-modal-actions'>
                    <Button disabled={props.submitting} onClick={props.onCancel}>
                        Cancel
                    </Button>
                    <Button type='primary' htmlType='submit' loading={props.submitting}>
                        {submitLabel}
                    </Button>
                </div>
            </Form>
        </Modal>
    );
};

const WalletBackupModal = (props: {
    results?: CreateWalletResult[];
    confirmed: boolean;
    onConfirmedChange: (value: boolean) => void;
    onDone: () => void;
    onCopy: (label: string, value: string) => void;
}) => (
    <Modal
        destroyOnHidden={true}
        rootClassName='wallet-batch-backup-modal'
        open={Boolean(props.results?.length)}
        title={props.results?.length === 1 ? 'Back Up Your Wallet' : 'Back Up Your Wallets'}
        width={760}
        closable={false}
        keyboard={false}
        mask={{closable: false}}
        footer={
            <Button type='primary' disabled={!props.confirmed} onClick={props.onDone}>
                Done
            </Button>
        }>
        <Space orientation='vertical' size='middle' className='wallet-secret-stack wallet-batch-backup'>
            <Alert
                showIcon={true}
                type='warning'
                title='This is the only automatic display after creation'
                description={
                    props.results?.length === 1
                        ? 'Store this private key somewhere secure. Anyone with the key can control the wallet. Athena will never ask you to share it.'
                        : `Store all ${props.results?.length || 0} private keys somewhere secure. Anyone with a key can control its wallet. Athena will never ask you to share them.`
                }
            />
            <div className='wallet-batch-backup__toolbar'>
                <Typography.Text type='secondary'>Results are shown in creation order.</Typography.Text>
                <Button
                    icon={<CopyOutlined />}
                    disabled={!props.results?.length}
                    aria-label='Copy all private keys, one per line'
                    onClick={() => props.onCopy('All private keys', (props.results || []).map(result => result.privateKey).join('\n'))}>
                    Copy all private keys
                </Button>
            </div>
            <div className='wallet-batch-backup__list' role='list' aria-label='Created wallet private keys'>
                {(props.results || []).map((result, index) => (
                    <section className='wallet-batch-backup__item' role='listitem' key={result.item.id || index} aria-labelledby={`wallet-backup-${index}-heading`}>
                        <div className='wallet-secret-heading'>
                            <WalletAvatar item={result.item} size={42} />
                            <span>
                                <strong id={`wallet-backup-${index}-heading`}>
                                    {index + 1}. {result.item.remark}
                                </strong>
                                <small>{result.item.walletType === 'SOLANA' ? 'Solana' : 'EVM'} wallet</small>
                            </span>
                        </div>
                        <div className='wallet-secret-field'>
                            <Typography.Text type='secondary'>Address</Typography.Text>
                            <Space.Compact block={true}>
                                <Input readOnly={true} value={result.item.address} />
                                <Tooltip title='Copy address'>
                                    <Button
                                        aria-label={`Copy ${result.item.remark} address`}
                                        icon={<CopyOutlined />}
                                        onClick={() => props.onCopy('Address', result.item.address)}
                                    />
                                </Tooltip>
                            </Space.Compact>
                        </div>
                        <div className='wallet-secret-field'>
                            <Typography.Text type='secondary'>Private key</Typography.Text>
                            <Space.Compact block={true}>
                                <Input.Password readOnly={true} autoComplete='off' value={result.privateKey} />
                                <Tooltip title='Copy private key'>
                                    <Button
                                        aria-label={`Copy ${result.item.remark} private key`}
                                        icon={<CopyOutlined />}
                                        onClick={() => props.onCopy('Private key', result.privateKey)}
                                    />
                                </Tooltip>
                            </Space.Compact>
                        </div>
                    </section>
                ))}
            </div>
            <Checkbox checked={props.confirmed} onChange={event => props.onConfirmedChange(event.target.checked)}>
                {props.results?.length === 1 ? 'I have securely backed up this private key' : 'I have securely backed up every private key in this batch'}
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
        <Space orientation='vertical' size='middle' className='wallet-secret-stack'>
            <Alert
                showIcon={true}
                type='warning'
                title='Keep this private key secret'
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
            size={520}
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
                            <Alert type='info' showIcon={true} title='Read-only wallet access does not include private keys.' />
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
    const [backup, setBackup] = React.useState<CreateWalletResult[]>();
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
        const importedLines = parseImportedPrivateKeys(values.privateKeys || '');
        const batchSize = mode === 'create' ? values.count || 1 : importedLines.length;
        const input = {
            walletType: values.walletType,
            remark: batchSize === 1 ? values.remark?.trim() || '' : '',
            avatarPresetId: values.avatarPresetId || ''
        };
        const result =
            mode === 'create'
                ? await lease.runTask(() => services.wallet.batchCreateWallets({...input, count: batchSize}))
                : await lease.runTask(() => services.wallet.batchImportWallets({...input, privateKeys: importedLines.map(line => line.privateKey)}));
        if (result.status === 'discarded') {
            return;
        }
        setSubmitting(false);
        if (result.status === 'rejected') {
            const status = requestErrorDetails(result.error).status;
            if (status === undefined || status === 0 || status === 408 || status >= 500) {
                props.onReload();
                ctx.notifications.warning(
                    mode === 'create' ? 'Wallet creation result is unknown' : 'Wallet import result is unknown',
                    mode === 'create'
                        ? 'Athena could not confirm the final result. The wallet list was refreshed; check it before trying again. Any committed key can be revealed later after reauthentication.'
                        : 'Athena could not confirm the final result. The wallet list was refreshed; check it before trying again.'
                );
                return;
            }
            const fallback = batchSize === 1 ? 'Could not save this wallet.' : `Could not save this batch of ${batchSize} wallets.`;
            ctx.notifications.error(
                mode === 'create' ? 'Wallet creation failed' : 'Wallet import failed',
                mode === 'import' ? importWalletErrorMessage(result.error, importedLines, fallback) : walletErrorMessage(result.error, fallback)
            );
            return;
        }
        if (mode === 'create') {
            const results = result.value as CreateWalletResult[];
            const complete =
                results.length === batchSize && results.every(saved => saved.item.id > 0 && Boolean(saved.item.address) && Boolean(saved.item.remark) && Boolean(saved.privateKey));
            if (!complete) {
                props.onReload();
                ctx.notifications.warning(
                    'Wallet creation result is incomplete',
                    'Athena did not return every expected wallet and private key. The wallet list was refreshed; check it before trying again. Any committed key can be revealed later after reauthentication.'
                );
                return;
            }
            setEditorMode(undefined);
            props.onReload();
            setBackupConfirmed(false);
            setBackup(results);
        } else {
            const items = result.value as WalletItem[];
            const complete = items.length === batchSize && items.every(item => item.id > 0 && Boolean(item.address) && Boolean(item.remark));
            if (!complete) {
                props.onReload();
                ctx.notifications.warning(
                    'Wallet import result is incomplete',
                    'Athena did not return every expected wallet. The wallet list was refreshed; check it before trying again.'
                );
                return;
            }
            setEditorMode(undefined);
            props.onReload();
            ctx.notifications.success(`${items.length} ${items.length === 1 ? 'wallet' : 'wallets'} imported`);
        }
    };

    const completeBackup = () => {
        if (!backup || !backupConfirmed) {
            return;
        }
        const item = backup[0]?.item;
        setBackup(undefined);
        setBackupConfirmed(false);
        if (item) {
            props.onWalletChanged(item);
            props.onSelectedChange(item);
        }
        ctx.notifications.success(`${backup.length} ${backup.length === 1 ? 'wallet' : 'wallets'} created`);
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
                results={backup}
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
        {key: 'create', label: 'Create wallets', icon: <PlusOutlined />},
        {key: 'import', label: 'Import wallets', icon: <ImportOutlined />}
    ];
    const writeActions = canWrite ? (
        <>
            <Space className='wallet-header-actions wallet-header-actions--desktop'>
                <Button icon={<ImportOutlined />} onClick={() => writeHandle.current?.openImport()}>
                    Import Wallets
                </Button>
                <Button type='primary' icon={<PlusOutlined />} onClick={() => writeHandle.current?.openCreate()}>
                    Create Wallets
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
                    responsive={true}
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
