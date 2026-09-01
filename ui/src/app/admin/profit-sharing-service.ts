import type {AbortablePromise, ProfitSharingRound, ProfitSharingRoundDefinition} from '../shared/services/profit-sharing-service';
import {ProfitSharingReader, roundBody} from '../shared/services/profit-sharing-service';
import requests from '../shared/services/requests';

const profitSharingWriteScope = {feature: 'profit-sharing' as const, mode: 'write' as const};

export class AdminProfitSharingService extends ProfitSharingReader {
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

    private roundAction(path: string, expectedRevision: number): AbortablePromise<void> {
        const req = requests.post(path, profitSharingWriteScope).send({expected_revision: expectedRevision});
        const promise = req.then(() => undefined) as AbortablePromise<void>;
        promise.abort = () => req.abort();
        return promise;
    }
}
