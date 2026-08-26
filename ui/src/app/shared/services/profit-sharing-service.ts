import {readBoolean, readNumber, readString, readValue} from './api-values';
import requests from './requests';

export enum ProfitSharingRoundPhase {
    Draft = 'DRAFT',
    Collecting = 'COLLECTING',
    Voting = 'VOTING',
    Closed = 'CLOSED'
}

export type ProfitSharingProposalStatus = 'DRAFT' | 'SUBMITTED';

export interface ProfitSharingParticipant {
    accountId: string;
    username: string;
    displayName: string;
    baselineResponsibility: string;
    sortOrder: number;
    proposalStatus: ProfitSharingProposalStatus;
}

export interface ProfitSharingProposalItem {
    accountId: string;
    username: string;
    displayName: string;
    responsibility: string;
    shareBasisPoints?: number;
}

export interface ProfitSharingProposal {
    id: string;
    label: string;
    isOwn: boolean;
    authorAccountId: string;
    authorUsername: string;
    authorDisplayName: string;
    status: ProfitSharingProposalStatus;
    revision: number;
    voteCountVisible: boolean;
    voteCount: number;
    isFinal: boolean;
    items: ProfitSharingProposalItem[];
}

export interface ProfitSharingResult {
    proposalId: string;
    label: string;
    authorAccountId: string;
    authorUsername: string;
    authorDisplayName: string;
    voteCount: number;
    isWinner: boolean;
}

export interface ProfitSharingRound {
    slug: string;
    title: string;
    phase: ProfitSharingRoundPhase;
    revision: number;
    participantCount: number;
    submittedCount: number;
    votedCount: number;
    ballotNumber: number;
    winnerProposalId?: string;
    participants: ProfitSharingParticipant[];
    myProposal?: ProfitSharingProposal;
    proposals: ProfitSharingProposal[];
    myVoteProposalId?: string;
    results: ProfitSharingResult[];
}

export interface ProfitSharingParticipantDefinition {
    accountId: string;
    username: string;
    displayName: string;
    baselineResponsibility: string;
    sortOrder: number;
}

export interface ProfitSharingRoundDefinition {
    slug: string;
    title: string;
    participants: ProfitSharingParticipantDefinition[];
    expectedRevision?: number;
}

export interface ProfitSharingProposalDraft {
    expectedRevision: number;
    items: Array<{
        accountId: string;
        responsibility: string;
        shareBasisPoints?: number;
    }>;
}

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

const profitSharingReadScope = {feature: 'profit-sharing' as const, mode: 'read' as const};
const profitSharingWriteScope = {feature: 'profit-sharing' as const, mode: 'write' as const};

const parsePhase = (value: unknown): ProfitSharingRoundPhase => {
    if (value === 1 || value === '1') {
        return ProfitSharingRoundPhase.Draft;
    }
    if (value === 2 || value === '2') {
        return ProfitSharingRoundPhase.Collecting;
    }
    if (value === 3 || value === '3') {
        return ProfitSharingRoundPhase.Voting;
    }
    if (value === 4 || value === '4') {
        return ProfitSharingRoundPhase.Closed;
    }
    const normalized = String(value || '')
        .toUpperCase()
        .replace(/^PROFIT_SHARING_/, '')
        .replace(/^ROUND_PHASE_/, '');
    switch (normalized) {
        case ProfitSharingRoundPhase.Collecting:
            return ProfitSharingRoundPhase.Collecting;
        case ProfitSharingRoundPhase.Voting:
            return ProfitSharingRoundPhase.Voting;
        case ProfitSharingRoundPhase.Closed:
            return ProfitSharingRoundPhase.Closed;
        default:
            return ProfitSharingRoundPhase.Draft;
    }
};

const parseProposalStatus = (value: unknown): ProfitSharingProposalStatus => {
    if (value === 2 || value === '2') {
        return 'SUBMITTED';
    }
    const normalized = String(value || '')
        .toUpperCase()
        .replace(/^PROFIT_SHARING_/, '')
        .replace(/^PROPOSAL_STATUS_/, '');
    return normalized === 'SUBMITTED' ? 'SUBMITTED' : 'DRAFT';
};

const optionalNumber = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (value === undefined || value === null || value === '') {
        return undefined;
    }
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
};

const optionalString = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (value === undefined || value === null || value === '') {
        return undefined;
    }
    return String(value);
};

const participant = (item: any): ProfitSharingParticipant => ({
    accountId: readString(item, 'accountId', 'account_id'),
    username: readString(item, 'username'),
    displayName: readString(item, 'displayName', 'display_name'),
    baselineResponsibility: readString(item, 'baselineResponsibility', 'baseline_responsibility'),
    sortOrder: readNumber(item, 'sortOrder', 'sort_order', 'displayOrder', 'display_order') || 0,
    proposalStatus: readBoolean(item, 'submitted') ? 'SUBMITTED' : parseProposalStatus(readValue(item, 'proposalStatus', 'proposal_status'))
});

const proposalItem = (item: any): ProfitSharingProposalItem => {
    const directShare = optionalNumber(item, 'shareBasisPoints', 'share_basis_points');
    const basisPoints = optionalNumber(item, 'basisPoints', 'basis_points');
    const basisPointsSet = readBoolean(item, 'shareBasisPointsSet', 'share_basis_points_set', 'basisPointsSet', 'basis_points_set');
    return {
        accountId: readString(item, 'participantAccountId', 'participant_account_id'),
        username: readString(item, 'participantUsername', 'participant_username'),
        displayName: readString(item, 'participantDisplayName', 'participant_display_name'),
        responsibility: readString(item, 'responsibility'),
        shareBasisPoints: directShare ?? (basisPointsSet ? basisPoints ?? 0 : undefined)
    };
};

export const parseProfitSharingProposal = (item: any): ProfitSharingProposal => {
    const items = readValue(item, 'items');
    return {
        id: readString(item, 'id'),
        label: readString(item, 'label'),
        isOwn: readBoolean(item, 'isOwn', 'is_own'),
        authorAccountId: readString(item, 'authorAccountId', 'author_account_id'),
        authorUsername: readString(item, 'authorUsername', 'author_username'),
        authorDisplayName: readString(item, 'authorDisplayName', 'author_display_name'),
        status: parseProposalStatus(readValue(item, 'status')),
        revision: readNumber(item, 'revision') || 0,
        voteCountVisible: readBoolean(item, 'voteCountVisible', 'vote_count_visible'),
        voteCount: readNumber(item, 'voteCount', 'vote_count') || 0,
        isFinal: readBoolean(item, 'isFinal', 'is_final'),
        items: Array.isArray(items) ? items.map(proposalItem) : []
    };
};

const result = (item: any): ProfitSharingResult => ({
    proposalId: readString(item, 'proposalId', 'proposal_id'),
    label: readString(item, 'label'),
    authorAccountId: readString(item, 'authorAccountId', 'author_account_id'),
    authorUsername: readString(item, 'authorUsername', 'author_username'),
    authorDisplayName: readString(item, 'authorDisplayName', 'author_display_name'),
    voteCount: readNumber(item, 'voteCount', 'vote_count') || 0,
    isWinner: readBoolean(item, 'isWinner', 'is_winner')
});

export const parseProfitSharingRound = (item: any): ProfitSharingRound => {
    const summary = readValue(item, 'summary') || item;
    const activeBallot = readValue(item, 'activeBallot', 'active_ballot') || {};
    const participants = readValue(item, 'participants');
    const proposals = readValue(item, 'proposals') || readValue(activeBallot, 'candidates');
    const results = readValue(item, 'results');
    const myProposal = readValue(item, 'myProposal', 'my_proposal', 'ownProposal', 'own_proposal');
    const normalizedProposals: ProfitSharingProposal[] = Array.isArray(proposals) ? proposals.map(parseProfitSharingProposal) : [];
    const winnerProposalId = optionalString(item, 'winnerProposalId', 'winner_proposal_id', 'finalProposalId', 'final_proposal_id');
    const normalizedResults: ProfitSharingResult[] = Array.isArray(results)
        ? results.map(result)
        : normalizedProposals
              .filter(proposal => proposal.voteCountVisible || proposal.isFinal || proposal.id === winnerProposalId)
              .map(proposal => ({
                  proposalId: proposal.id,
                  label: proposal.label,
                  authorAccountId: proposal.authorAccountId,
                  authorUsername: proposal.authorUsername,
                  authorDisplayName: proposal.authorDisplayName,
                  voteCount: proposal.voteCount,
                  isWinner: proposal.isFinal || proposal.id === winnerProposalId
              }));
    return {
        slug: readString(summary, 'slug'),
        title: readString(summary, 'title'),
        phase: parsePhase(readValue(summary, 'phase')),
        revision: readNumber(summary, 'revision') || 0,
        participantCount:
            readNumber(summary, 'participantCount', 'participant_count') ||
            readNumber(activeBallot, 'participantCount', 'participant_count') ||
            (Array.isArray(participants) ? participants.length : 0),
        submittedCount: readNumber(summary, 'submittedCount', 'submitted_count') || 0,
        votedCount: readNumber(summary, 'votedCount', 'voted_count') || readNumber(activeBallot, 'votedCount', 'voted_count') || 0,
        ballotNumber:
            readNumber(item, 'ballotNumber', 'ballot_number') || readNumber(summary, 'activeBallotNumber', 'active_ballot_number') || readNumber(activeBallot, 'number') || 0,
        winnerProposalId,
        participants: Array.isArray(participants) ? participants.map(participant).sort((left, right) => left.sortOrder - right.sortOrder) : [],
        myProposal: myProposal ? parseProfitSharingProposal(myProposal) : undefined,
        proposals: normalizedProposals,
        myVoteProposalId: optionalString(item, 'myVoteProposalId', 'my_vote_proposal_id') || optionalString(activeBallot, 'currentVoteProposalId', 'current_vote_proposal_id'),
        results: normalizedResults
    };
};

const roundBody = (body: any) => parseProfitSharingRound(readValue(body, 'round') || body || {});

export class ProfitSharingService {
    public listRounds(): AbortablePromise<ProfitSharingRound[]> {
        const req = requests.get('/profit-sharing/rounds', profitSharingReadScope);
        const promise = req.then(res => {
            const values = readValue(res.body, 'rounds', 'items');
            return (Array.isArray(values) ? values : []).map(parseProfitSharingRound);
        }) as AbortablePromise<ProfitSharingRound[]>;
        promise.abort = () => req.abort();
        return promise;
    }

    public getRound(slug: string): AbortablePromise<ProfitSharingRound> {
        const req = requests.get(`/profit-sharing/rounds/${encodeURIComponent(slug)}`, profitSharingReadScope);
        const promise = req.then(res => roundBody(res.body)) as AbortablePromise<ProfitSharingRound>;
        promise.abort = () => req.abort();
        return promise;
    }

    public createRound(definition: ProfitSharingRoundDefinition): AbortablePromise<ProfitSharingRound> {
        const req = requests.post('/profit-sharing/rounds', profitSharingWriteScope).send({
            slug: definition.slug,
            title: definition.title,
            participants: definition.participants.map(participant => ({
                account_id: participant.accountId,
                username: participant.username,
                display_name: participant.displayName,
                sort_order: participant.sortOrder,
                baseline_responsibility: participant.baselineResponsibility
            }))
        });
        const promise = req.then(res => roundBody(res.body)) as AbortablePromise<ProfitSharingRound>;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateRound(slug: string, definition: ProfitSharingRoundDefinition): AbortablePromise<ProfitSharingRound> {
        const req = requests.put(`/profit-sharing/rounds/${encodeURIComponent(slug)}`, profitSharingWriteScope).send({
            slug: definition.slug,
            title: definition.title,
            expected_revision: definition.expectedRevision,
            participants: definition.participants.map(participant => ({
                account_id: participant.accountId,
                username: participant.username,
                display_name: participant.displayName,
                sort_order: participant.sortOrder,
                baseline_responsibility: participant.baselineResponsibility
            }))
        });
        const promise = req.then(res => roundBody(res.body)) as AbortablePromise<ProfitSharingRound>;
        promise.abort = () => req.abort();
        return promise;
    }

    public openRound(slug: string, expectedRevision: number): AbortablePromise<void> {
        return this.roundAction(`/profit-sharing/rounds/${encodeURIComponent(slug)}:open`, expectedRevision);
    }

    public publishRound(slug: string, expectedRevision: number): AbortablePromise<void> {
        return this.roundAction(`/profit-sharing/rounds/${encodeURIComponent(slug)}:publish`, expectedRevision);
    }

    public closeBallot(slug: string, expectedRevision: number): AbortablePromise<void> {
        return this.roundAction(`/profit-sharing/rounds/${encodeURIComponent(slug)}/ballots/current:close`, expectedRevision);
    }

    public updateProposal(slug: string, proposal: ProfitSharingProposalDraft): AbortablePromise<ProfitSharingProposal | undefined> {
        const req = requests.put(`/profit-sharing/rounds/${encodeURIComponent(slug)}/proposal`, profitSharingWriteScope).send({
            expected_revision: proposal.expectedRevision,
            items: proposal.items.map(item => ({
                participant_account_id: item.accountId,
                responsibility: item.responsibility,
                share_basis_points: item.shareBasisPoints ?? 0,
                share_basis_points_set: item.shareBasisPoints !== undefined
            }))
        });
        const promise = req.then(res => {
            const value = readValue(res.body, 'proposal');
            return value ? parseProfitSharingProposal(value) : undefined;
        }) as AbortablePromise<ProfitSharingProposal | undefined>;
        promise.abort = () => req.abort();
        return promise;
    }

    public submitProposal(slug: string, expectedRevision: number): AbortablePromise<void> {
        return this.proposalAction(`/profit-sharing/rounds/${encodeURIComponent(slug)}/proposal:submit`, expectedRevision);
    }

    public reopenProposal(slug: string, expectedRevision: number): AbortablePromise<void> {
        return this.proposalAction(`/profit-sharing/rounds/${encodeURIComponent(slug)}/proposal:reopen`, expectedRevision);
    }

    public submitVote(slug: string, proposalId: string): AbortablePromise<void> {
        const req = requests.put(`/profit-sharing/rounds/${encodeURIComponent(slug)}/ballots/current/vote`, profitSharingWriteScope).send({proposal_id: proposalId});
        const promise = req.then(() => undefined) as AbortablePromise<void>;
        promise.abort = () => req.abort();
        return promise;
    }

    private roundAction(path: string, expectedRevision: number): AbortablePromise<void> {
        const req = requests.post(path, profitSharingWriteScope).send({expected_revision: expectedRevision});
        const promise = req.then(() => undefined) as AbortablePromise<void>;
        promise.abort = () => req.abort();
        return promise;
    }

    private proposalAction(path: string, expectedRevision: number): AbortablePromise<void> {
        const req = requests.post(path, profitSharingWriteScope).send({expected_revision: expectedRevision});
        const promise = req.then(() => undefined) as AbortablePromise<void>;
        promise.abort = () => req.abort();
        return promise;
    }
}
