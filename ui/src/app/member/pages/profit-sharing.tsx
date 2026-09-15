import {ArrowLeftOutlined, EditOutlined, SafetyCertificateOutlined, SendOutlined, TrophyOutlined} from '@ant-design/icons';
import {Alert, Button, Empty, Input, InputNumber, Progress, Result, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, ResourceTable, Section, useAsyncData} from '../../components';
import {Context, useAuthorization} from '../../shared/context';
import {memberServices as services} from '../services';
import {ProfitSharingRoundPhase} from '../../shared/services/profit-sharing-service';
import type {ProfitSharingProposalItem, ProfitSharingProposalStatus, ProfitSharingRound} from '../../shared/services/profit-sharing-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {
    formatShareBasisPoints,
    ProfitSharingPhaseTag,
    ProfitSharingRoundLink,
    ProfitSharingRoundRecord,
    ProfitSharingProposalGallery,
    ProfitSharingRoundMetrics,
    useProfitSharingConfirm,
    useProfitSharingUnsavedChanges
} from '../../shared/pages/profit-sharing-shared';

const proposalItemsFor = (round: ProfitSharingRound): ProfitSharingProposalItem[] => {
    const proposalByAccount = new Map((round.myProposal?.items || []).map(item => [item.accountId, item]));
    return round.participants.map(participant => {
        const item = proposalByAccount.get(participant.accountId);
        return {
            accountId: participant.accountId,
            username: participant.username,
            displayName: participant.displayName,
            responsibility: item?.responsibility ?? participant.baselineResponsibility,
            shareBasisPoints: item?.shareBasisPoints
        };
    });
};

const proposalItemsEqual = (left: ProfitSharingProposalItem[], right: ProfitSharingProposalItem[]) =>
    left.length === right.length &&
    left.every((item, index) => {
        const other = right[index];
        return Boolean(other) && item.accountId === other.accountId && item.responsibility === other.responsibility && item.shareBasisPoints === other.shareBasisPoints;
    });

const proposalIsComplete = (items: ProfitSharingProposalItem[]) =>
    items.length === 5 &&
    items.every(item => Boolean(item.responsibility.trim()) && item.shareBasisPoints !== undefined && item.shareBasisPoints >= 0) &&
    items.reduce((total, item) => total + (item.shareBasisPoints || 0), 0) === 10_000;

const roundColumns: ColumnsType<ProfitSharingRound> = [
    {
        title: 'Round',
        render: item => <ProfitSharingRoundLink round={item} />
    },
    {title: 'Phase', width: 180, render: item => <ProfitSharingPhaseTag phase={item.phase} />},
    {
        title: 'Proposals',
        align: 'right',
        className: 'athena-numeric-column',
        width: 140,
        render: item => `${item.submittedCount} / ${item.participantCount}`
    },
    {
        title: 'Votes',
        align: 'right',
        className: 'athena-numeric-column',
        width: 140,
        render: item => (item.phase === ProfitSharingRoundPhase.Draft || item.phase === ProfitSharingRoundPhase.Collecting ? '—' : `${item.votedCount} / ${item.participantCount}`)
    }
];

const ProfitSharingRoundsPageContent = () => {
    const rounds = useAsyncData(() => services.memberProfitSharing.listRounds(), []);
    return (
        <div className='foundation-page'>
            <AppPage
                title='Profit Sharing'
                subtitle='Submit and compare complete responsibility and profit-sharing proposals across reusable rounds.'
                loading={rounds.loading}
                error={rounds.error}
                stale={Boolean(rounds.error && rounds.data)}
                onRefresh={rounds.reload}>
                <Section title='Rounds'>
                    <ResourceTable
                        rowKey='slug'
                        label='Profit-sharing rounds'
                        items={rounds.data || []}
                        columns={roundColumns}
                        loading={rounds.loading}
                        hasData={rounds.data !== undefined}
                        compactRender={round => <ProfitSharingRoundRecord round={round} />}
                        compactEmptyDescription='No profit-sharing rounds are available'
                    />
                </Section>
            </AppPage>
        </div>
    );
};

const ProposalEditor = (props: {
    items: ProfitSharingProposalItem[];
    status: ProfitSharingProposalStatus;
    dirty: boolean;
    saving: boolean;
    onChange: (items: ProfitSharingProposalItem[]) => void;
    onSave: () => void;
    onSubmit: () => void;
    onReopen: () => void;
}) => {
    const total = props.items.reduce((sum, item) => sum + (item.shareBasisPoints || 0), 0);
    const complete = proposalIsComplete(props.items);
    const submitted = props.status === 'SUBMITTED';
    const update = (index: number, change: Partial<ProfitSharingProposalItem>) =>
        props.onChange(props.items.map((item, itemIndex) => (itemIndex === index ? {...item, ...change} : item)));
    return (
        <div className='profit-sharing-editor' aria-busy={props.saving || undefined}>
            <div className='profit-sharing-editor__intro'>
                <div>
                    <Typography.Title level={2}>{submitted ? 'Proposal submitted' : 'Build your proposal'}</Typography.Title>
                    <Typography.Text type='secondary'>
                        Define every member's responsibility and profit share. Drafts may be incomplete; submission requires exactly 100%.
                    </Typography.Text>
                </div>
                <Tag color={submitted ? 'success' : props.dirty ? 'warning' : 'default'}>{submitted ? 'Submitted' : props.dirty ? 'Unsaved draft' : 'Draft saved'}</Tag>
            </div>

            {submitted ? (
                <Alert
                    type='success'
                    showIcon={true}
                    title='Your proposal is sealed.'
                    description='You may reopen it while proposal collection remains open. Reopening marks it as a draft again.'
                    action={
                        <Button disabled={props.saving} loading={props.saving} onClick={props.onReopen}>
                            Reopen proposal
                        </Button>
                    }
                />
            ) : (
                <>
                    <div className='profit-sharing-editor__rows'>
                        <div className='profit-sharing-editor__header' aria-hidden='true'>
                            <span>Member</span>
                            <span>Responsibility</span>
                            <span>Share</span>
                        </div>
                        {props.items.map((item, index) => (
                            <article className='profit-sharing-editor-row' key={item.accountId}>
                                <div className='profit-sharing-editor-row__member'>
                                    <strong>{item.displayName || `@${item.username}`}</strong>
                                    {item.username && <small>@{item.username}</small>}
                                </div>
                                <label className='profit-sharing-editor-row__responsibility'>
                                    <span>Responsibility</span>
                                    <Input.TextArea
                                        aria-label={`Responsibility for ${item.displayName || item.username}`}
                                        value={item.responsibility}
                                        disabled={props.saving}
                                        maxLength={500}
                                        autoSize={{minRows: 2, maxRows: 5}}
                                        placeholder='Describe this member’s responsibilities'
                                        onChange={event => update(index, {responsibility: event.target.value})}
                                    />
                                </label>
                                <label className='profit-sharing-editor-row__share'>
                                    <span>Share</span>
                                    <InputNumber
                                        aria-label={`Share for ${item.displayName || item.username}`}
                                        value={item.shareBasisPoints === undefined ? null : item.shareBasisPoints / 100}
                                        disabled={props.saving}
                                        min={0}
                                        max={100}
                                        precision={2}
                                        step={0.25}
                                        addonAfter='%'
                                        placeholder='0.00'
                                        onChange={value => update(index, {shareBasisPoints: value === null ? undefined : Math.round(Number(value) * 100)})}
                                    />
                                </label>
                            </article>
                        ))}
                    </div>
                    <div
                        className={`profit-sharing-editor__total${total === 10_000 ? ' profit-sharing-editor__total--valid' : total > 10_000 ? ' profit-sharing-editor__total--invalid' : ''}`}>
                        <div>
                            <Typography.Text strong={true}>Total allocation</Typography.Text>
                            <Typography.Text type={total > 10_000 ? 'danger' : total === 10_000 ? 'success' : 'secondary'}>{formatShareBasisPoints(total)}</Typography.Text>
                        </div>
                        <Progress
                            aria-label='Total allocation'
                            percent={Math.min(100, total / 100)}
                            showInfo={false}
                            status={total > 10_000 ? 'exception' : total === 10_000 ? 'success' : 'active'}
                        />
                        <Typography.Text type='secondary'>
                            {complete ? 'Ready to submit.' : 'Every responsibility and share must be filled, and shares must total exactly 100%.'}
                        </Typography.Text>
                    </div>
                </>
            )}

            <div className='profit-sharing-editor__footer'>
                <Typography.Text type={props.dirty ? 'warning' : 'secondary'}>
                    {submitted ? 'Submitted proposals are sealed until voting opens.' : props.dirty ? 'You have unsaved changes.' : 'Your draft is saved.'}
                </Typography.Text>
                {!submitted && (
                    <Space wrap={true}>
                        <Button icon={<EditOutlined />} loading={props.saving} disabled={!props.dirty || props.saving} onClick={props.onSave}>
                            Save draft
                        </Button>
                        <Button type='primary' icon={<SendOutlined />} loading={props.saving} disabled={!complete || props.saving} onClick={props.onSubmit}>
                            Submit proposal
                        </Button>
                    </Space>
                )}
            </div>
        </div>
    );
};

const VotingPanel = (props: {round: ProfitSharingRound; onReload: () => void}) => {
    const ctx = React.useContext(Context);
    const confirm = useProfitSharingConfirm();
    const [selected, setSelected] = React.useState<string>();
    const [submitting, setSubmitting] = React.useState(false);

    React.useEffect(() => {
        setSelected(props.round.myVoteProposalId);
    }, [props.round.ballotNumber, props.round.myVoteProposalId, props.round.slug]);

    const selectedProposal = props.round.proposals.find(item => item.id === selected);
    const submitVote = () => {
        if (!selectedProposal || selectedProposal.isOwn || submitting) {
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: props.round.myVoteProposalId ? 'Update your vote?' : 'Submit your vote?',
            content: `Choose ${selectedProposal.label || `Proposal ${selectedProposal.id}`}. You may update your choice while voting remains open.`,
            okText: props.round.myVoteProposalId ? 'Update vote' : 'Submit vote',
            onOk: async () => {
                setSubmitting(true);
                try {
                    await services.memberProfitSharing.submitVote(props.round.slug, selectedProposal.id);
                    ctx.notifications.success('Vote recorded', 'Your current ballot choice has been saved.');
                    props.onReload();
                } catch (error) {
                    ctx.notifications.error('Could not record vote', requestErrorMessage(error));
                    throw error;
                } finally {
                    setSubmitting(false);
                }
            }
        });
    };

    return (
        <Section title={`Ballot ${props.round.ballotNumber || 1}`}>
            <div className='profit-sharing-voting'>
                <Alert
                    type='info'
                    showIcon={true}
                    title='Proposal authors and vote totals stay hidden until the round has a winner.'
                    description='Review every allocation. You may choose exactly one proposal, and your own proposal is not eligible.'
                />
                {props.round.proposals.length > 0 ? (
                    <ProfitSharingProposalGallery proposals={props.round.proposals} selectedProposalId={selected} selectable={true} onSelect={setSelected} />
                ) : (
                    <Empty description='No eligible proposals are available' />
                )}
                <div className='profit-sharing-voting__footer'>
                    <Typography.Text type='secondary'>
                        {props.round.myVoteProposalId ? 'Your existing vote remains valid until you replace it.' : 'No vote has been submitted yet.'}
                    </Typography.Text>
                    <Button
                        type='primary'
                        icon={<SafetyCertificateOutlined />}
                        loading={submitting}
                        disabled={!selectedProposal || selectedProposal.isOwn || selected === props.round.myVoteProposalId || submitting}
                        onClick={submitVote}>
                        {props.round.myVoteProposalId ? 'Update vote' : 'Submit vote'}
                    </Button>
                </div>
            </div>
        </Section>
    );
};

const ClosedPanel = (props: {round: ProfitSharingRound}) => {
    const winner = props.round.results.find(item => item.isWinner || item.proposalId === props.round.winnerProposalId);
    const orderedProposals = [...props.round.proposals].sort((left, right) => {
        if (left.id === props.round.winnerProposalId) {
            return -1;
        }
        if (right.id === props.round.winnerProposalId) {
            return 1;
        }
        return left.id.localeCompare(right.id);
    });
    return (
        <Section title='Final result'>
            <div className='profit-sharing-results'>
                {winner ? (
                    <Alert
                        type='success'
                        showIcon={true}
                        icon={<TrophyOutlined />}
                        title={`${winner.label || `Proposal ${winner.proposalId}`} is the selected proposal.`}
                        description={`Proposed by ${winner.authorDisplayName || (winner.authorUsername ? `@${winner.authorUsername}` : 'Unknown member')}. It received ${winner.voteCount} vote${winner.voteCount === 1 ? '' : 's'}.`}
                    />
                ) : (
                    <Alert type='warning' showIcon={true} title='This ballot closed without a single winning proposal.' />
                )}
                {orderedProposals.length > 0 ? (
                    <ProfitSharingProposalGallery proposals={orderedProposals} showAuthor={true} results={props.round.results} />
                ) : (
                    <Empty description='No final proposals are available' />
                )}
            </div>
        </Section>
    );
};

const ProfitSharingRoundPageContent = () => {
    const params = useParams<{slug: string}>();
    const slug = params.slug || '';
    const navigate = useNavigate();
    const authorization = useAuthorization();
    const ctx = React.useContext(Context);
    const confirm = useProfitSharingConfirm();
    const roundData = useAsyncData(() => services.memberProfitSharing.getRound(slug), [slug]);
    const round = roundData.data;
    const [draft, setDraft] = React.useState<ProfitSharingProposalItem[]>([]);
    const [saved, setSaved] = React.useState<ProfitSharingProposalItem[]>([]);
    const [proposalRevision, setProposalRevision] = React.useState(0);
    const [proposalStatus, setProposalStatus] = React.useState<ProfitSharingProposalStatus>('DRAFT');
    const [saving, setSaving] = React.useState(false);
    const dirty = !proposalItemsEqual(draft, saved);
    const source = React.useMemo(() => (round ? proposalItemsFor(round) : []), [round]);
    const sourceSignature = round ? JSON.stringify({slug: round.slug, revision: round.myProposal?.revision || 0, phase: round.phase, items: source}) : '';

    React.useEffect(() => {
        if (!round) {
            return;
        }
        setDraft(source);
        setSaved(source);
        setProposalRevision(round.myProposal?.revision || 0);
        setProposalStatus(round.myProposal?.status || 'DRAFT');
    }, [sourceSignature]);

    const discard = React.useCallback(() => setDraft(saved), [saved]);
    useProfitSharingUnsavedChanges(dirty, 'Your profit-sharing proposal draft has not been saved.', discard);

    const reloadAuthoritative = React.useCallback(() => {
        roundData.reload();
    }, [roundData.reload]);

    const handleMutationError = (error: unknown, title: string) => {
        if (requestErrorDetails(error).status === 409) {
            ctx.notifications.warning('Proposal changed elsewhere', 'Your local draft was discarded and the latest proposal is being loaded.');
            setDraft(saved);
            roundData.reload();
            return;
        }
        ctx.notifications.error(title, requestErrorMessage(error));
    };

    const persistDraft = async () => {
        if (!round || saving) {
            return undefined;
        }
        setSaving(true);
        try {
            const updated = await services.memberProfitSharing.updateProposal(round.slug, {
                expectedRevision: proposalRevision,
                items: draft.map(item => ({
                    accountId: item.accountId,
                    responsibility: item.responsibility,
                    shareBasisPoints: item.shareBasisPoints
                }))
            });
            const nextItems = updated?.items?.length
                ? source.map(sourceItem => {
                      const item = updated.items.find(candidate => candidate.accountId === sourceItem.accountId);
                      return item ? {...item, username: sourceItem.username, displayName: sourceItem.displayName} : sourceItem;
                  })
                : draft;
            const nextRevision = updated?.revision ?? proposalRevision + 1;
            setSaved(nextItems);
            setDraft(nextItems);
            setProposalRevision(nextRevision);
            setProposalStatus(updated?.status || 'DRAFT');
            ctx.notifications.success('Draft saved');
            roundData.reload();
            return nextRevision;
        } catch (error) {
            handleMutationError(error, 'Could not save proposal draft');
            return undefined;
        } finally {
            setSaving(false);
        }
    };

    const submitProposal = () => {
        if (!round || !proposalIsComplete(draft) || saving) {
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: 'Submit this proposal?',
            content: 'The proposal will be sealed. You can reopen it only while collection remains open.',
            okText: 'Submit proposal',
            onOk: async () => {
                let revision = proposalRevision;
                if (dirty) {
                    const savedRevision = await persistDraft();
                    if (savedRevision === undefined) {
                        throw new Error('The draft could not be saved.');
                    }
                    revision = savedRevision;
                }
                setSaving(true);
                try {
                    await services.memberProfitSharing.submitProposal(round.slug, revision);
                    setProposalStatus('SUBMITTED');
                    ctx.notifications.success('Proposal submitted', 'Your proposal is now sealed.');
                    roundData.reload();
                } catch (error) {
                    handleMutationError(error, 'Could not submit proposal');
                    throw error;
                } finally {
                    setSaving(false);
                }
            }
        });
    };

    const reopenProposal = () => {
        if (!round || saving) {
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: 'Reopen your proposal?',
            content: 'Your submission will return to draft status and must be submitted again before proposals are published.',
            okText: 'Reopen proposal',
            onOk: async () => {
                setSaving(true);
                try {
                    await services.memberProfitSharing.reopenProposal(round.slug, proposalRevision);
                    setProposalStatus('DRAFT');
                    ctx.notifications.info('Proposal reopened');
                    roundData.reload();
                } catch (error) {
                    handleMutationError(error, 'Could not reopen proposal');
                    throw error;
                } finally {
                    setSaving(false);
                }
            }
        });
    };

    const refresh = () => {
        if (!dirty) {
            roundData.reload();
            return;
        }
        confirm({
            className: 'profit-sharing-confirm',
            title: 'Discard your unsaved draft?',
            content: 'Refreshing will replace your local changes with the latest saved proposal.',
            okText: 'Discard and refresh',
            onOk: () => {
                discard();
                roundData.reload();
            }
        });
    };

    if (!slug) {
        return <Result status='warning' title='Round not specified' extra={<Button onClick={() => navigate('/profit-sharing')}>View all rounds</Button>} />;
    }

    const isParticipant = Boolean(round?.participants.some(item => item.accountId === authorization.user.accountId));
    return (
        <div className='foundation-page'>
            <AppPage
                title={round?.title || 'Profit Sharing Round'}
                subtitle={round ? `Round /${round.slug} · Revision ${round.revision}` : undefined}
                loading={roundData.loading}
                error={roundData.error}
                onRefresh={round ? refresh : roundData.reload}
                extra={
                    <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/profit-sharing')}>
                        All rounds
                    </Button>
                }>
                {round && (
                    <>
                        <div className='profit-sharing-round-heading'>
                            <ProfitSharingPhaseTag phase={round.phase} />
                            <Typography.Text type='secondary'>Each participant submits one complete proposal and may vote only for another participant's proposal.</Typography.Text>
                        </div>
                        <ProfitSharingRoundMetrics round={round} />

                        {round.phase === ProfitSharingRoundPhase.Draft && (
                            <Result status='info' title='This round is being prepared.' subTitle='Proposal collection has not opened yet.' />
                        )}

                        {round.phase === ProfitSharingRoundPhase.Collecting &&
                            (isParticipant ? (
                                <section className='section-panel profit-sharing-proposal-editor' aria-label='Your proposal'>
                                    <div className='section-panel__body'>
                                        <ProposalEditor
                                            items={draft}
                                            status={proposalStatus}
                                            dirty={dirty}
                                            saving={saving}
                                            onChange={setDraft}
                                            onSave={() => void persistDraft()}
                                            onSubmit={submitProposal}
                                            onReopen={reopenProposal}
                                        />
                                    </div>
                                </section>
                            ) : (
                                <Result status='403' title='You are not a participant in this round.' subTitle='Only configured participants can create a proposal.' />
                            ))}

                        {round.phase === ProfitSharingRoundPhase.Voting &&
                            (isParticipant ? (
                                <VotingPanel round={round} onReload={reloadAuthoritative} />
                            ) : (
                                <Result status='403' title='You are not eligible to vote in this round.' />
                            ))}

                        {round.phase === ProfitSharingRoundPhase.Closed && <ClosedPanel round={round} />}
                    </>
                )}
            </AppPage>
        </div>
    );
};

export const ProfitSharingRoundsPage = () => {
    const authorization = useAuthorization();
    const {slug = ''} = useParams<{slug: string}>();
    return <ProfitSharingRoundsPageContent key={JSON.stringify([authorization.user.accountId, authorization.user.iss, slug])} />;
};

export const ProfitSharingRoundPage = () => {
    const authorization = useAuthorization();
    const {slug = ''} = useParams<{slug: string}>();
    return <ProfitSharingRoundPageContent key={JSON.stringify([authorization.user.accountId, authorization.user.iss, slug])} />;
};
