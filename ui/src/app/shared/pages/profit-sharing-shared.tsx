import {LockOutlined, TrophyOutlined} from '@ant-design/icons';
import {Alert, Card, Progress, Radio, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {Link, useBlocker} from 'react-router-dom';
import {Context, type ModalApi, type ModalHandle} from '../context';
import {ProfitSharingRoundPhase} from '../services/profit-sharing-service';
import type {ProfitSharingProposal, ProfitSharingProposalItem, ProfitSharingResult, ProfitSharingRound} from '../services/profit-sharing-service';

export const profitSharingPhaseLabel = (phase: ProfitSharingRoundPhase) => {
    switch (phase) {
        case ProfitSharingRoundPhase.Collecting:
            return 'Collecting proposals';
        case ProfitSharingRoundPhase.Voting:
            return 'Voting';
        case ProfitSharingRoundPhase.Closed:
            return 'Closed';
        default:
            return 'Draft';
    }
};

export const ProfitSharingPhaseTag = (props: {phase: ProfitSharingRoundPhase}) => {
    const color =
        props.phase === ProfitSharingRoundPhase.Collecting
            ? 'processing'
            : props.phase === ProfitSharingRoundPhase.Voting
              ? 'warning'
              : props.phase === ProfitSharingRoundPhase.Closed
                ? 'success'
                : 'default';
    return <Tag color={color}>{profitSharingPhaseLabel(props.phase)}</Tag>;
};

export const ProfitSharingRoundLink = ({round}: {round: ProfitSharingRound}) => (
    <div className='profit-sharing-round-title'>
        <Link className='foundation-record-link' to={`/profit-sharing/${encodeURIComponent(round.slug)}`}>
            {round.title}
        </Link>
        <span className='athena-identifier'>/{round.slug}</span>
    </div>
);

export const ProfitSharingRoundRecord = ({round, admin = false}: {round: ProfitSharingRound; admin?: boolean}) => (
    <article className='profit-sharing-round-card'>
        <ProfitSharingRoundLink round={round} />
        <dl>
            <div>
                <dt>Phase</dt>
                <dd>
                    <ProfitSharingPhaseTag phase={round.phase} />
                </dd>
            </div>
            {admin && (
                <div>
                    <dt>Participants</dt>
                    <dd>{round.participantCount}</dd>
                </div>
            )}
            <div>
                <dt>{admin ? 'Submitted' : 'Proposals'}</dt>
                <dd>
                    {round.submittedCount} / {round.participantCount}
                </dd>
            </div>
            <div>
                <dt>{admin ? 'Voted' : 'Votes'}</dt>
                <dd>
                    {round.phase === ProfitSharingRoundPhase.Voting || round.phase === ProfitSharingRoundPhase.Closed ? `${round.votedCount} / ${round.participantCount}` : '—'}
                </dd>
            </div>
        </dl>
    </article>
);

export const formatShareBasisPoints = (value?: number) => {
    if (value === undefined) {
        return '—';
    }
    return `${(value / 100).toLocaleString(undefined, {minimumFractionDigits: 2, maximumFractionDigits: 2})}%`;
};

export const ProfitSharingRoundMetrics = (props: {round: ProfitSharingRound}) => {
    const round = props.round;
    const submissionPercent = round.participantCount ? Math.round((round.submittedCount / round.participantCount) * 100) : 0;
    const votePercent = round.participantCount ? Math.round((round.votedCount / round.participantCount) * 100) : 0;
    return (
        <section className='profit-sharing-metrics' aria-label='Round progress'>
            <article className='profit-sharing-metric'>
                <div>
                    <Typography.Text type='secondary'>Phase</Typography.Text>
                    <strong>{profitSharingPhaseLabel(round.phase)}</strong>
                </div>
            </article>
            <article className='profit-sharing-metric'>
                <div>
                    <Typography.Text type='secondary'>Proposals submitted</Typography.Text>
                    <strong>
                        {round.submittedCount} / {round.participantCount}
                    </strong>
                    <Progress aria-label='Proposals submitted' percent={submissionPercent} showInfo={false} size='small' />
                </div>
            </article>
            <article className='profit-sharing-metric'>
                <div>
                    <Typography.Text type='secondary'>Votes submitted</Typography.Text>
                    <strong>
                        {round.votedCount} / {round.participantCount}
                    </strong>
                    <Progress aria-label='Votes submitted' percent={votePercent} showInfo={false} size='small' />
                </div>
            </article>
        </section>
    );
};

export const ProfitSharingAllocationList = (props: {items: ProfitSharingProposalItem[]}) => (
    <div className='profit-sharing-allocation-list' role='table' aria-label='Proposed responsibilities and shares'>
        <div className='profit-sharing-allocation-list__header' role='row'>
            <span role='columnheader'>Member</span>
            <span role='columnheader'>Responsibility</span>
            <span role='columnheader'>Share</span>
        </div>
        {props.items.map(item => (
            <div className='profit-sharing-allocation-list__row' role='row' key={item.accountId}>
                <span className='profit-sharing-allocation-list__member' role='cell'>
                    <strong>{item.displayName || `@${item.username}`}</strong>
                    {item.username && <small>@{item.username}</small>}
                </span>
                <span className='profit-sharing-allocation-list__responsibility' role='cell'>
                    {item.responsibility || '—'}
                </span>
                <strong className='profit-sharing-allocation-list__share' role='cell'>
                    {formatShareBasisPoints(item.shareBasisPoints)}
                </strong>
            </div>
        ))}
    </div>
);

const proposalResult = (proposal: ProfitSharingProposal, results: ProfitSharingResult[]) => results.find(item => item.proposalId === proposal.id);

export const ProfitSharingProposalCard = (props: {
    proposal: ProfitSharingProposal;
    selected?: boolean;
    selectable?: boolean;
    showAuthor?: boolean;
    results?: ProfitSharingResult[];
    onSelect?: (proposalId: string) => void;
}) => {
    const result = proposalResult(props.proposal, props.results || []);
    const disabled = Boolean(props.proposal.isOwn || !props.selectable);
    const select = () => {
        if (!disabled) {
            props.onSelect?.(props.proposal.id);
        }
    };
    return (
        <Card
            className={`profit-sharing-proposal-card${props.selected ? ' profit-sharing-proposal-card--selected' : ''}${result?.isWinner ? ' profit-sharing-proposal-card--winner' : ''}`}
            size='small'
            onClick={props.selectable ? select : undefined}>
            <div className='profit-sharing-proposal-card__heading'>
                <div>
                    {props.selectable ? (
                        <Radio value={props.proposal.id} disabled={props.proposal.isOwn} onClick={event => event.stopPropagation()}>
                            <Typography.Text strong={true}>{props.proposal.label || `Proposal ${props.proposal.id}`}</Typography.Text>
                        </Radio>
                    ) : (
                        <Typography.Title level={3}>{props.proposal.label || `Proposal ${props.proposal.id}`}</Typography.Title>
                    )}
                    {props.showAuthor && (
                        <Typography.Text type='secondary'>
                            Proposed by {props.proposal.authorDisplayName || (props.proposal.authorUsername ? `@${props.proposal.authorUsername}` : 'Unknown member')}
                        </Typography.Text>
                    )}
                </div>
                <Space size={6} wrap={true}>
                    {props.proposal.isOwn && <Tag color='processing'>Your proposal</Tag>}
                    {result?.isWinner && (
                        <Tag color='success' icon={<TrophyOutlined />}>
                            Winner
                        </Tag>
                    )}
                    {result && <Tag>{result.voteCount} votes</Tag>}
                </Space>
            </div>
            {props.proposal.isOwn && props.selectable && (
                <Alert className='profit-sharing-proposal-card__notice' type='info' showIcon={true} title='You cannot vote for your own proposal.' />
            )}
            <ProfitSharingAllocationList items={props.proposal.items} />
        </Card>
    );
};

export const ProfitSharingProposalGallery = (props: {
    proposals: ProfitSharingProposal[];
    selectedProposalId?: string;
    selectable?: boolean;
    showAuthor?: boolean;
    results?: ProfitSharingResult[];
    onSelect?: (proposalId: string) => void;
}) => {
    const cards = props.proposals.map(proposal => (
        <ProfitSharingProposalCard
            key={proposal.id}
            proposal={proposal}
            selected={proposal.id === props.selectedProposalId}
            selectable={props.selectable}
            showAuthor={props.showAuthor}
            results={props.results}
            onSelect={props.onSelect}
        />
    ));
    return props.selectable ? (
        <Radio.Group
            className='profit-sharing-proposal-gallery'
            aria-label='Choose a proposal'
            value={props.selectedProposalId}
            onChange={event => props.onSelect?.(String(event.target.value))}>
            {cards}
        </Radio.Group>
    ) : (
        <div className='profit-sharing-proposal-gallery'>{cards}</div>
    );
};

export const ProfitSharingSealedNotice = () => (
    <Alert
        type='info'
        showIcon={true}
        icon={<LockOutlined />}
        title='Proposals remain sealed while collection is open.'
        description='Only submission status is visible. Proposal contents are revealed together when voting opens.'
    />
);

// Confirmations belong to this mounted identity/round, including portal content.
export const useProfitSharingConfirm = () => {
    const {modal} = React.useContext(Context);
    const handles = React.useRef(new Set<ModalHandle>());
    React.useEffect(
        () => () => {
            handles.current.forEach(handle => handle.destroy());
            handles.current.clear();
        },
        []
    );
    return React.useCallback(
        (options: Parameters<ModalApi['confirm']>[0]) => {
            const handle = modal.confirm({
                ...options,
                className: 'profit-sharing-confirm',
                focusable: {trap: true, focusTriggerAfterClose: true},
                wrapProps: {
                    onKeyDownCapture: event => {
                        if (event.key !== 'Tab') return;
                        const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not([disabled])'));
                        const first = buttons[0];
                        const last = buttons[buttons.length - 1];
                        if (first && last && ((!event.shiftKey && document.activeElement === last) || (event.shiftKey && document.activeElement === first))) {
                            event.preventDefault();
                            (event.shiftKey ? last : first).focus();
                        }
                    }
                }
            });
            handles.current.add(handle);
            return handle;
        },
        [modal]
    );
};

export const useProfitSharingUnsavedChanges = (dirty: boolean, message: string, onDiscard: () => void) => {
    const confirm = useProfitSharingConfirm();
    const blocker = useBlocker(dirty);

    React.useEffect(() => {
        if (!dirty) {
            return;
        }
        const beforeUnload = (event: BeforeUnloadEvent) => {
            event.preventDefault();
            event.returnValue = '';
        };
        window.addEventListener('beforeunload', beforeUnload);
        return () => window.removeEventListener('beforeunload', beforeUnload);
    }, [dirty]);

    React.useEffect(() => {
        if (blocker.state !== 'blocked') {
            return;
        }
        let resolved = false;
        const handle = confirm({
            className: 'profit-sharing-confirm',
            title: 'Discard unsaved changes?',
            content: message,
            okText: 'Discard and leave',
            onOk: () => {
                resolved = true;
                onDiscard();
                blocker.proceed();
            },
            onCancel: () => {
                resolved = true;
                blocker.reset();
            }
        });
        return () => {
            if (!resolved) {
                handle.destroy();
            }
        };
    }, [blocker.state, confirm, message, onDiscard]);
};
