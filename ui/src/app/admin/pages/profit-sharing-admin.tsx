import {ArrowLeftOutlined, DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined, SettingOutlined, TrophyOutlined} from '@ant-design/icons';
import {Alert, Button, Empty, Form, Input, Modal, Result, Select, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, ResourceTable, Section, useAsyncData} from '../../components';
import {Context, useAuthorization} from '../../shared/context';
import {adminServices as services} from '../services';
import {ProfitSharingRoundPhase} from '../../shared/services/profit-sharing-service';
import type {ProfitSharingParticipant, ProfitSharingParticipantDefinition, ProfitSharingRound} from '../../shared/services/profit-sharing-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {
    ProfitSharingPhaseTag,
    ProfitSharingRoundLink,
    ProfitSharingRoundRecord,
    ProfitSharingProposalGallery,
    ProfitSharingRoundMetrics,
    ProfitSharingSealedNotice,
    useProfitSharingConfirm,
    useProfitSharingUnsavedChanges
} from '../../shared/pages/profit-sharing-shared';

const blankParticipants = (): ProfitSharingParticipantDefinition[] =>
    Array.from({length: 5}, (_, index) => ({
        accountId: '',
        username: '',
        displayName: '',
        baselineResponsibility: '',
        sortOrder: index + 1
    }));

interface EligibleAccountOption {
    value: string;
    label: string;
    username: string;
    displayName: string;
}

const useEligibleAccountOptions = () => {
    const [search, setSearch] = React.useState('');
    const [knownOptions, setKnownOptions] = React.useState<Map<string, EligibleAccountOption>>(new Map());
    const deferredSearch = React.useDeferredValue(search.trim());
    const accounts = useAsyncData(() => services.adminAccounts.list({query: deferredSearch, page: 1, pageSize: 100, profitSharingEligibleOnly: true}), [deferredSearch]);
    React.useEffect(() => {
        if (!accounts.data) {
            return;
        }
        setKnownOptions(current => {
            const next = deferredSearch ? new Map(current) : new Map<string, EligibleAccountOption>();
            accounts.data.items.forEach(account => {
                const values = [account.profile.displayName, account.identity.verifiedEmail || account.identity.solanaAddress, `@${account.username}`].filter(Boolean);
                next.set(account.id, {
                    value: account.id,
                    label: Array.from(new Set(values)).join(' · '),
                    username: account.username,
                    displayName: account.profile.displayName
                });
            });
            return next;
        });
    }, [accounts.data]);
    return {...accounts, options: Array.from(knownOptions.values()), search: setSearch};
};

const snapshotAccountOptions = (participants: ProfitSharingParticipant[]): EligibleAccountOption[] =>
    participants.map(participant => ({
        value: participant.accountId,
        label: [participant.displayName, `@${participant.username}`].filter(Boolean).join(' · '),
        username: participant.username,
        displayName: participant.displayName
    }));

const mergeAccountOptions = (...groups: EligibleAccountOption[][]) => Array.from(new Map(groups.flat().map(option => [option.value, option])).values());

interface RoundDefinitionDraft {
    slug: string;
    title: string;
    participants: ProfitSharingParticipantDefinition[];
}

const definitionFor = (round: ProfitSharingRound): RoundDefinitionDraft => ({
    slug: round.slug,
    title: round.title,
    participants: round.participants.map((participant, index) => ({
        accountId: participant.accountId,
        username: participant.username,
        displayName: participant.displayName,
        baselineResponsibility: participant.baselineResponsibility,
        sortOrder: participant.sortOrder || index + 1
    }))
});

const normalizeDefinition = (definition: RoundDefinitionDraft): RoundDefinitionDraft => ({
    slug: (definition.slug || '').trim().toLowerCase(),
    title: (definition.title || '').trim(),
    participants: (definition.participants || []).map((participant, index) => ({
        accountId: (participant.accountId || '').trim(),
        username: (participant.username || '').trim(),
        displayName: (participant.displayName || '').trim(),
        baselineResponsibility: (participant.baselineResponsibility || '').trim(),
        sortOrder: index + 1
    }))
});

const definitionsEqual = (left: RoundDefinitionDraft, right: RoundDefinitionDraft) => JSON.stringify(normalizeDefinition(left)) === JSON.stringify(normalizeDefinition(right));

const hasDuplicateAccounts = (participants: Array<{accountId: string}>) => {
    const ids = participants.map(item => item.accountId.trim()).filter(Boolean);
    return new Set(ids).size !== ids.length;
};

const roundColumns: ColumnsType<ProfitSharingRound> = [
    {
        title: 'Round',
        render: round => <ProfitSharingRoundLink round={round} />
    },
    {title: 'Phase', width: 180, render: round => <ProfitSharingPhaseTag phase={round.phase} />},
    {title: 'Participants', align: 'right', className: 'athena-numeric-column', width: 130, dataIndex: 'participantCount'},
    {title: 'Submitted', align: 'right', className: 'athena-numeric-column', width: 140, render: round => `${round.submittedCount} / ${round.participantCount}`},
    {
        title: 'Voted',
        align: 'right',
        className: 'athena-numeric-column',
        width: 140,
        render: round =>
            round.phase === ProfitSharingRoundPhase.Voting || round.phase === ProfitSharingRoundPhase.Closed ? `${round.votedCount} / ${round.participantCount}` : '—'
    }
];

const RoundDefinitionFields = (props: {
    accountOptions: EligibleAccountOption[];
    accountsLoading?: boolean;
    disabled?: boolean;
    slugDisabled?: boolean;
    onAccountSearch: (query: string) => void;
}) => {
    const form = Form.useFormInstance<RoundDefinitionDraft>();
    const selectAccount = (index: number, accountID: string) => {
        const account = props.accountOptions.find(option => option.value === accountID);
        form.setFieldValue(['participants', index, 'username'], account?.username || '');
        form.setFieldValue(['participants', index, 'displayName'], account?.displayName || '');
    };
    return (
        <>
            <div className='profit-sharing-definition__identity'>
                <Form.Item name='title' label='Round title' rules={[{required: true, whitespace: true, max: 120}]}>
                    <Input disabled={props.disabled} placeholder='Profit Sharing · Phase 1' />
                </Form.Item>
                <Form.Item
                    name='slug'
                    label='Round URL slug'
                    rules={[
                        {required: true, whitespace: true},
                        {pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/, message: 'Use lowercase letters, numbers, and single hyphens.'}
                    ]}>
                    <Input disabled={props.disabled} readOnly={props.slugDisabled} addonBefore='/profit-sharing/' placeholder='phase-1' />
                </Form.Item>
            </div>
            <div className='profit-sharing-definition__participants-heading'>
                <div>
                    <Typography.Title level={3}>Participants</Typography.Title>
                    <Typography.Text type='secondary'>
                        Search for exactly five non-administrator accounts with sign-in and Profit Sharing access. Draft definitions may contain fewer.
                    </Typography.Text>
                </div>
            </div>
            <Form.List name='participants'>
                {(fields, {add, remove}) => (
                    <div className='profit-sharing-definition__participants'>
                        {fields.map((field, index) => (
                            <div className='profit-sharing-definition-participant' key={field.key}>
                                <span className='profit-sharing-definition-participant__order'>{index + 1}</span>
                                <Form.Item
                                    {...field}
                                    className='profit-sharing-definition-participant__account'
                                    name={[field.name, 'accountId']}
                                    label='Account'
                                    rules={[{required: true, whitespace: true}]}>
                                    <Select
                                        disabled={props.disabled}
                                        showSearch={true}
                                        filterOption={false}
                                        loading={props.accountsLoading}
                                        options={props.accountOptions}
                                        placeholder='Search an authorized member'
                                        onChange={value => selectAccount(field.name, value)}
                                        onSearch={props.onAccountSearch}
                                        onOpenChange={open => open && props.onAccountSearch('')}
                                    />
                                </Form.Item>
                                <Form.Item {...field} name={[field.name, 'username']} hidden={true}>
                                    <Input />
                                </Form.Item>
                                <Form.Item
                                    {...field}
                                    className='profit-sharing-definition-participant__name'
                                    name={[field.name, 'displayName']}
                                    label='Display name'
                                    rules={[{required: true, whitespace: true}]}>
                                    <Input readOnly={true} placeholder='Selected account display name' />
                                </Form.Item>
                                <Form.Item
                                    {...field}
                                    className='profit-sharing-definition-participant__responsibility'
                                    name={[field.name, 'baselineResponsibility']}
                                    label='Baseline responsibility'
                                    rules={[{required: true, whitespace: true, max: 500}]}>
                                    <Input.TextArea disabled={props.disabled} autoSize={{minRows: 1, maxRows: 4}} placeholder='Initial responsibility description' />
                                </Form.Item>
                                <Button
                                    className='profit-sharing-definition-participant__remove'
                                    type='text'
                                    danger={true}
                                    icon={<DeleteOutlined />}
                                    aria-label={`Remove participant ${index + 1}`}
                                    disabled={props.disabled}
                                    onClick={() => remove(field.name)}
                                />
                            </div>
                        ))}
                        <Button
                            block={true}
                            icon={<PlusOutlined />}
                            disabled={props.disabled || fields.length >= 5}
                            onClick={() => add({accountId: '', username: '', displayName: '', baselineResponsibility: '', sortOrder: fields.length + 1})}>
                            Add participant
                        </Button>
                    </div>
                )}
            </Form.List>
        </>
    );
};

const CreateRoundModal = (props: {
    open: boolean;
    accountOptions: EligibleAccountOption[];
    accountsLoading?: boolean;
    onAccountSearch: (query: string) => void;
    onClose: () => void;
    onCreated: (round: ProfitSharingRound) => void;
}) => {
    const ctx = React.useContext(Context);
    const confirm = useProfitSharingConfirm();
    const formID = React.useId();
    const [form] = Form.useForm<RoundDefinitionDraft>();
    const [submitting, setSubmitting] = React.useState(false);

    React.useEffect(() => {
        if (props.open) {
            form.setFieldsValue({title: '', slug: '', participants: blankParticipants()});
        }
    }, [form, props.open]);

    const submit = async (values: RoundDefinitionDraft) => {
        const definition = normalizeDefinition(values);
        if (hasDuplicateAccounts(definition.participants)) {
            ctx.notifications.error('Participant accounts must be unique');
            return;
        }
        setSubmitting(true);
        try {
            const created = await services.adminProfitSharing.createRound(definition);
            ctx.notifications.success('Profit-sharing round created', created.title || definition.title);
            form.resetFields();
            props.onCreated({...created, slug: created.slug || definition.slug, title: created.title || definition.title});
        } catch (error) {
            ctx.notifications.error('Could not create round', requestErrorMessage(error));
        } finally {
            setSubmitting(false);
        }
    };

    const close = () => {
        if (!form.isFieldsTouched()) {
            props.onClose();
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: 'Discard this new round?',
            content: 'The unsaved round definition will be lost.',
            okText: 'Discard',
            onOk: () => {
                form.resetFields();
                props.onClose();
            }
        });
    };

    return (
        <Modal
            className='profit-sharing-definition-modal'
            open={props.open}
            title='Create profit-sharing round'
            width={960}
            footer={
                <div className='profit-sharing-definition__footer'>
                    <Button disabled={submitting} onClick={close}>
                        Cancel
                    </Button>
                    <Button type='primary' htmlType='submit' form={formID} loading={submitting} icon={<PlusOutlined />}>
                        Create round
                    </Button>
                </div>
            }
            destroyOnHidden={true}
            closable={!submitting}
            onCancel={close}>
            <Form id={formID} form={form} layout='vertical' disabled={submitting} onFinish={submit}>
                <RoundDefinitionFields
                    accountOptions={props.accountOptions}
                    accountsLoading={props.accountsLoading}
                    disabled={submitting}
                    onAccountSearch={props.onAccountSearch}
                />
            </Form>
        </Modal>
    );
};

const ProfitSharingAdminRoundsPageContent = () => {
    const navigate = useNavigate();
    const rounds = useAsyncData(() => services.adminProfitSharing.listRounds(), []);
    const accounts = useEligibleAccountOptions();
    const [createOpen, setCreateOpen] = React.useState(false);
    const accountOptions = accounts.options;
    return (
        <div className='foundation-page'>
            <AppPage
                title='Profit Sharing Administration'
                subtitle='Create reusable rounds, monitor sealed submissions, publish proposals together, and close ballots.'
                loading={rounds.loading}
                error={rounds.error || accounts.error}
                onRefresh={() => {
                    rounds.reload();
                    accounts.reload();
                }}
                extra={
                    <Button type='primary' icon={<PlusOutlined />} disabled={accounts.loading || Boolean(accounts.error)} onClick={() => setCreateOpen(true)}>
                        New round
                    </Button>
                }>
                <Section title='All rounds'>
                    <ResourceTable
                        rowKey='slug'
                        label='Administer profit-sharing rounds'
                        items={rounds.data || []}
                        columns={roundColumns}
                        loading={rounds.loading}
                        compactRender={round => <ProfitSharingRoundRecord round={round} admin={true} />}
                        compactEmptyDescription='No profit-sharing rounds have been created'
                    />
                </Section>
                <CreateRoundModal
                    open={createOpen}
                    accountOptions={accountOptions}
                    accountsLoading={accounts.loading}
                    onAccountSearch={accounts.search}
                    onClose={() => setCreateOpen(false)}
                    onCreated={round => {
                        setCreateOpen(false);
                        navigate(`/profit-sharing/${encodeURIComponent(round.slug)}`);
                    }}
                />
            </AppPage>
        </div>
    );
};

const ParticipantStatusCard = ({participant}: {participant: ProfitSharingParticipant}) => (
    <article className='profit-sharing-participant-card'>
        <strong>{participant.displayName || `@${participant.username}`}</strong>
        <Typography.Text type='secondary'>@{participant.username}</Typography.Text>
        <p>{participant.baselineResponsibility}</p>
        <Tag color={participant.proposalStatus === 'SUBMITTED' ? 'success' : 'default'}>{participant.proposalStatus === 'SUBMITTED' ? 'Submitted' : 'Draft'}</Tag>
    </article>
);

const participantColumns: ColumnsType<ProfitSharingParticipant> = [
    {
        title: 'Participant',
        render: item => (
            <div className='profit-sharing-round-title'>
                <strong>{item.displayName || `@${item.username}`}</strong>
                {item.username && <Typography.Text type='secondary'>@{item.username}</Typography.Text>}
            </div>
        )
    },
    {title: 'Baseline responsibility', dataIndex: 'baselineResponsibility'},
    {
        title: 'Proposal status',
        width: 160,
        render: item => <Tag color={item.proposalStatus === 'SUBMITTED' ? 'success' : 'default'}>{item.proposalStatus === 'SUBMITTED' ? 'Submitted' : 'Draft'}</Tag>
    }
];

const AdminClosedResult = (props: {round: ProfitSharingRound}) => {
    const winner = props.round.results.find(item => item.isWinner || item.proposalId === props.round.winnerProposalId);
    return (
        <Section title='Final result'>
            <div className='profit-sharing-results'>
                {winner ? (
                    <Alert
                        type='success'
                        showIcon={true}
                        icon={<TrophyOutlined />}
                        title={`${winner.label || `Proposal ${winner.proposalId}`} won the ballot.`}
                        description={`${winner.authorDisplayName || (winner.authorUsername ? `@${winner.authorUsername}` : 'Unknown member')} · ${winner.voteCount} vote${winner.voteCount === 1 ? '' : 's'}`}
                    />
                ) : (
                    <Alert type='warning' showIcon={true} title='The ballot closed without a single winner.' />
                )}
                {props.round.proposals.length > 0 ? (
                    <ProfitSharingProposalGallery proposals={props.round.proposals} showAuthor={true} results={props.round.results} />
                ) : (
                    <Empty description='No final proposals are available' />
                )}
            </div>
        </Section>
    );
};

const ProfitSharingAdminRoundPageContent = () => {
    const params = useParams<{slug: string}>();
    const slug = params.slug || '';
    const navigate = useNavigate();
    const ctx = React.useContext(Context);
    const confirm = useProfitSharingConfirm();
    const roundData = useAsyncData(() => services.adminProfitSharing.getRound(slug), [slug]);
    const accounts = useEligibleAccountOptions();
    const round = roundData.data;
    const accountOptions = mergeAccountOptions(accounts.options, snapshotAccountOptions(round?.participants || []));
    const [form] = Form.useForm<RoundDefinitionDraft>();
    const watched = Form.useWatch([], form) as RoundDefinitionDraft | undefined;
    const [savedDefinition, setSavedDefinition] = React.useState<RoundDefinitionDraft>({slug: '', title: '', participants: []});
    const [saving, setSaving] = React.useState(false);
    const currentDefinition = watched || savedDefinition;
    const dirty = Boolean(round?.phase === ProfitSharingRoundPhase.Draft && !definitionsEqual(currentDefinition, savedDefinition));
    const eligibleAccountIDs = new Set(accounts.options.map(option => option.value));
    const rosterIsEligible = Boolean(
        round &&
            !accounts.loading &&
            !accounts.error &&
            round.participantCount === 5 &&
            round.participants.length === 5 &&
            !hasDuplicateAccounts(round.participants) &&
            round.participants.every(participant => eligibleAccountIDs.has(participant.accountId))
    );

    React.useEffect(() => {
        if (!round) {
            return;
        }
        const definition = definitionFor(round);
        setSavedDefinition(definition);
        form.setFieldsValue(definition);
    }, [form, round?.revision, round?.slug]);

    const discard = React.useCallback(() => form.setFieldsValue(savedDefinition), [form, savedDefinition]);
    useProfitSharingUnsavedChanges(dirty, 'The draft round definition has not been saved.', discard);

    const mutationError = (error: unknown, title: string) => {
        if (requestErrorDetails(error).status === 409) {
            ctx.notifications.warning('Round changed elsewhere', 'Your local changes were discarded and the current round is being loaded.');
            discard();
            roundData.reload();
            return;
        }
        ctx.notifications.error(title, requestErrorMessage(error));
    };

    const saveDefinition = async () => {
        if (!round || round.phase !== ProfitSharingRoundPhase.Draft || saving) {
            return;
        }
        let values: RoundDefinitionDraft;
        try {
            values = await form.validateFields();
        } catch {
            return;
        }
        const definition = normalizeDefinition(values);
        if (hasDuplicateAccounts(definition.participants)) {
            ctx.notifications.error('Participant accounts must be unique');
            return;
        }
        setSaving(true);
        try {
            const updated = await services.adminProfitSharing.updateRound(round.slug, {...definition, expectedRevision: round.revision});
            const nextSlug = updated.slug || definition.slug;
            setSavedDefinition(definition);
            form.setFieldsValue(definition);
            ctx.notifications.success('Round definition saved');
            if (nextSlug !== round.slug) {
                navigate(`/profit-sharing/${encodeURIComponent(nextSlug)}`, {replace: true});
            } else {
                roundData.reload();
            }
        } catch (error) {
            mutationError(error, 'Could not save round definition');
        } finally {
            setSaving(false);
        }
    };

    const advance = () => {
        if (!round || saving) {
            return;
        }
        const config =
            round.phase === ProfitSharingRoundPhase.Draft
                ? {
                      title: 'Open proposal collection?',
                      content: 'The participant roster will be locked and every participant can begin editing a proposal.',
                      okText: 'Open collection',
                      action: () => services.adminProfitSharing.openRound(round.slug, round.revision),
                      success: 'Proposal collection opened'
                  }
                : round.phase === ProfitSharingRoundPhase.Collecting
                  ? {
                        title: 'Publish all proposals?',
                        content: 'Every submitted proposal will be revealed at the same time and voting will begin. This cannot be undone.',
                        okText: 'Publish and open voting',
                        action: () => services.adminProfitSharing.publishRound(round.slug, round.revision),
                        success: 'Proposals published and voting opened'
                    }
                  : round.phase === ProfitSharingRoundPhase.Voting
                    ? {
                          title: 'Close this ballot?',
                          content:
                              'Votes in this ballot will be finalized. A tie automatically opens another anonymous runoff; authors and totals remain hidden until one proposal wins.',
                          okText: 'Close ballot',
                          action: () => services.adminProfitSharing.closeBallot(round.slug, round.revision),
                          success: 'Ballot closed and round status refreshed'
                      }
                    : undefined;
        if (!config) {
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: config.title,
            content: config.content,
            okText: config.okText,
            onOk: async () => {
                setSaving(true);
                try {
                    await config.action();
                    ctx.notifications.success(config.success);
                    roundData.reload();
                } catch (error) {
                    mutationError(error, 'Could not advance the round');
                    throw error;
                } finally {
                    setSaving(false);
                }
            }
        });
    };

    const canAdvance = Boolean(
        round &&
            !dirty &&
            !saving &&
            ((round.phase === ProfitSharingRoundPhase.Draft && rosterIsEligible) ||
                (round.phase === ProfitSharingRoundPhase.Collecting && round.participantCount > 0 && round.submittedCount === round.participantCount) ||
                (round.phase === ProfitSharingRoundPhase.Voting && round.participantCount > 0 && round.votedCount === round.participantCount))
    );
    const advanceLabel =
        round?.phase === ProfitSharingRoundPhase.Draft
            ? 'Open collection'
            : round?.phase === ProfitSharingRoundPhase.Collecting
              ? 'Publish proposals'
              : round?.phase === ProfitSharingRoundPhase.Voting
                ? 'Close ballot'
                : '';

    if (!slug) {
        return <Result status='warning' title='Round not specified' extra={<Button onClick={() => navigate('/profit-sharing')}>View all rounds</Button>} />;
    }

    return (
        <div className='foundation-page'>
            <AppPage
                title={round?.title || 'Profit Sharing Administration'}
                subtitle={round ? `Administer /${round.slug} · Revision ${round.revision}` : undefined}
                loading={roundData.loading}
                error={roundData.error || accounts.error}
                onRefresh={() => {
                    roundData.reload();
                    accounts.reload();
                }}
                extra={
                    <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/profit-sharing')}>
                        All rounds
                    </Button>
                }>
                {round && (
                    <>
                        <div className='profit-sharing-round-heading'>
                            <ProfitSharingPhaseTag phase={round.phase} />
                            <Typography.Text type='secondary'>Round revision {round.revision}</Typography.Text>
                        </div>
                        <ProfitSharingRoundMetrics round={round} />

                        <Section
                            title='Round control'
                            extra={
                                advanceLabel ? (
                                    <Button
                                        type='primary'
                                        icon={round.phase === ProfitSharingRoundPhase.Draft ? <EditOutlined /> : <SendOutlined />}
                                        loading={saving}
                                        disabled={!canAdvance}
                                        onClick={advance}>
                                        {advanceLabel}
                                    </Button>
                                ) : (
                                    <Tag color='success'>Complete</Tag>
                                )
                            }>
                            <div className='profit-sharing-admin-control'>
                                {round.phase === ProfitSharingRoundPhase.Draft && (
                                    <Alert
                                        type='info'
                                        showIcon={true}
                                        title='Finish the participant roster before opening collection.'
                                        description={
                                            dirty
                                                ? 'Save or discard the current definition before opening the round.'
                                                : rosterIsEligible
                                                  ? 'Opening collection locks the five-person roster and creates one proposal workspace per participant.'
                                                  : round.participantCount === 5
                                                    ? 'Every participant must remain signed-in eligible and authorized for Profit Sharing before collection can open.'
                                                    : `Exactly five eligible participants are required. This draft currently has ${round.participantCount}.`
                                        }
                                    />
                                )}
                                {round.phase === ProfitSharingRoundPhase.Collecting && (
                                    <Alert
                                        type={round.submittedCount === round.participantCount ? 'success' : 'info'}
                                        showIcon={true}
                                        title={round.submittedCount === round.participantCount ? 'Every participant has submitted.' : 'Waiting for all proposals.'}
                                        description={`${round.submittedCount} of ${round.participantCount} proposals are sealed. Publishing stays disabled until all are submitted.`}
                                    />
                                )}
                                {round.phase === ProfitSharingRoundPhase.Voting && (
                                    <Alert
                                        type={round.votedCount === round.participantCount ? 'success' : 'info'}
                                        showIcon={true}
                                        title={round.votedCount === round.participantCount ? 'Every participant has voted.' : 'Voting is in progress.'}
                                        description={`${round.votedCount} of ${round.participantCount} participants have voted. Closing stays disabled until all votes are recorded.`}
                                    />
                                )}
                                {round.phase === ProfitSharingRoundPhase.Closed && <Alert type='success' showIcon={true} title='This round is final and read-only.' />}
                                {round.phase === ProfitSharingRoundPhase.Collecting && <ProfitSharingSealedNotice />}
                            </div>
                        </Section>

                        {round.phase === ProfitSharingRoundPhase.Draft && (
                            <Section title='Round definition'>
                                <Form form={form} className='profit-sharing-definition' layout='vertical' disabled={saving}>
                                    <RoundDefinitionFields
                                        accountOptions={accountOptions}
                                        accountsLoading={accounts.loading}
                                        disabled={saving || Boolean(accounts.error)}
                                        slugDisabled={true}
                                        onAccountSearch={accounts.search}
                                    />
                                    <div className='profit-sharing-definition__footer'>
                                        <Typography.Text type={dirty ? 'warning' : 'secondary'}>{dirty ? 'Unsaved round definition' : 'Definition saved'}</Typography.Text>
                                        <Space wrap={true}>
                                            <Button disabled={!dirty || saving} onClick={discard}>
                                                Discard
                                            </Button>
                                            <Button type='primary' icon={<SettingOutlined />} loading={saving} disabled={!dirty || saving} onClick={() => void saveDefinition()}>
                                                Save definition
                                            </Button>
                                        </Space>
                                    </div>
                                </Form>
                            </Section>
                        )}

                        <Section title='Participant status'>
                            <ResourceTable
                                rowKey='accountId'
                                label='Profit-sharing participant submission status'
                                items={round.participants}
                                columns={participantColumns}
                                compactRender={participant => <ParticipantStatusCard participant={participant} />}
                                compactEmptyDescription='No participants configured'
                            />
                        </Section>

                        {round.phase === ProfitSharingRoundPhase.Voting && (
                            <Section title={`Published proposals · Ballot ${round.ballotNumber || 1}`}>
                                <Alert
                                    className='profit-sharing-admin-proposals__notice'
                                    type='info'
                                    showIcon={true}
                                    title='Authors and vote totals remain hidden until the round has a winner.'
                                />
                                {round.proposals.length > 0 ? (
                                    <ProfitSharingProposalGallery proposals={round.proposals} />
                                ) : (
                                    <Empty description='No published proposals are available' />
                                )}
                            </Section>
                        )}

                        {round.phase === ProfitSharingRoundPhase.Closed && <AdminClosedResult round={round} />}
                    </>
                )}
            </AppPage>
        </div>
    );
};

export const ProfitSharingAdminRoundsPage = () => {
    const authorization = useAuthorization();
    const {slug = ''} = useParams<{slug: string}>();
    return <ProfitSharingAdminRoundsPageContent key={JSON.stringify([authorization.user.accountId, authorization.user.iss, slug])} />;
};

export const ProfitSharingAdminRoundPage = () => {
    const authorization = useAuthorization();
    const {slug = ''} = useParams<{slug: string}>();
    return <ProfitSharingAdminRoundPageContent key={JSON.stringify([authorization.user.accountId, authorization.user.iss, slug])} />;
};
