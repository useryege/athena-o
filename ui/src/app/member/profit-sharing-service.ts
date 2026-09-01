import {readValue} from '../shared/services/api-values';
import type {AbortablePromise, ProfitSharingProposal, ProfitSharingProposalDraft} from '../shared/services/profit-sharing-service';
import {parseProfitSharingProposal, ProfitSharingReader} from '../shared/services/profit-sharing-service';
import requests from '../shared/services/requests';

const profitSharingWriteScope = {feature: 'profit-sharing' as const, mode: 'write' as const};

export class MemberProfitSharingService extends ProfitSharingReader {
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

    private proposalAction(path: string, expectedRevision: number): AbortablePromise<void> {
        const req = requests.post(path, profitSharingWriteScope).send({expected_revision: expectedRevision});
        const promise = req.then(() => undefined) as AbortablePromise<void>;
        promise.abort = () => req.abort();
        return promise;
    }
}
