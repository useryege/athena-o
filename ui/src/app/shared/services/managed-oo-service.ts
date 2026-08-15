import requests from './requests';
import {readNumber, readString} from './api-values';

export interface ManagedOOBaseItem {
    txHash: string;
    logIndex: number;
    blockNumber: number;
    blockHash: string;
    txIndex: number;
    contractAddress: string;
    topic: string;
    requester: string;
    proposer: string;
    identifier: string;
    requestTimestamp: number;
    ancillaryDataHex: string;
    ancillaryDataText: string;
    marketId: string;
    proposedPrice: string;
    rawTopics: string;
    rawData: string;
    fetchedAt: string;
    conditionId: string;
    eventSlug: string;
    marketSlug: string;
    question: string;
    polymarketUrl: string;
}

export interface ManagedOOProposalItem extends ManagedOOBaseItem {
    expirationTimestamp: number;
    currency: string;
}

export interface ManagedOODisputeItem extends ManagedOOBaseItem {
    disputer: string;
}

export interface ListManagedOOItemsResult<T extends ManagedOOBaseItem> {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
}

export interface ScanManagedOOBlockResult {
    blockNumber: number;
    proposalCount: number;
    disputeCount: number;
}

const normalizeBase = (item: any): ManagedOOBaseItem => ({
    txHash: readString(item, 'txHash', 'tx_hash'),
    logIndex: readNumber(item, 'logIndex', 'log_index') || 0,
    blockNumber: readNumber(item, 'blockNumber', 'block_number') || 0,
    blockHash: readString(item, 'blockHash', 'block_hash'),
    txIndex: readNumber(item, 'txIndex', 'tx_index') || 0,
    contractAddress: readString(item, 'contractAddress', 'contract_address'),
    topic: readString(item, 'topic'),
    requester: readString(item, 'requester'),
    proposer: readString(item, 'proposer'),
    identifier: readString(item, 'identifier'),
    requestTimestamp: readNumber(item, 'requestTimestamp', 'request_timestamp') || 0,
    ancillaryDataHex: readString(item, 'ancillaryDataHex', 'ancillary_data_hex'),
    ancillaryDataText: readString(item, 'ancillaryDataText', 'ancillary_data_text'),
    marketId: readString(item, 'marketId', 'market_id'),
    proposedPrice: readString(item, 'proposedPrice', 'proposed_price'),
    rawTopics: readString(item, 'rawTopics', 'raw_topics'),
    rawData: readString(item, 'rawData', 'raw_data'),
    fetchedAt: readString(item, 'fetchedAt', 'fetched_at'),
    conditionId: readString(item, 'conditionId', 'condition_id'),
    eventSlug: readString(item, 'eventSlug', 'event_slug'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    question: readString(item, 'question'),
    polymarketUrl: readString(item, 'polymarketUrl', 'polymarket_url')
});

const normalizeProposal = (item: any): ManagedOOProposalItem => ({
    ...normalizeBase(item),
    expirationTimestamp: readNumber(item, 'expirationTimestamp', 'expiration_timestamp') || 0,
    currency: readString(item, 'currency')
});

const normalizeDispute = (item: any): ManagedOODisputeItem => ({
    ...normalizeBase(item),
    disputer: readString(item, 'disputer')
});

export class ManagedOOService {
    public scanBlock(blockNumber: number): Promise<ScanManagedOOBlockResult> & {abort?: () => void} {
        const req = requests.post(`/managed-oo/blocks/${blockNumber}:scan`).send({});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                blockNumber: readNumber(body, 'blockNumber', 'block_number') || blockNumber,
                proposalCount: readNumber(body, 'proposalCount', 'proposal_count') || 0,
                disputeCount: readNumber(body, 'disputeCount', 'dispute_count') || 0
            };
        }) as Promise<ScanManagedOOBlockResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public listProposals(page = 1, pageSize = 20, blockNumber?: number): Promise<ListManagedOOItemsResult<ManagedOOProposalItem>> & {abort?: () => void} {
        return this.list('/managed-oo/proposals', normalizeProposal, page, pageSize, blockNumber);
    }

    public listDisputes(page = 1, pageSize = 20, blockNumber?: number): Promise<ListManagedOOItemsResult<ManagedOODisputeItem>> & {abort?: () => void} {
        return this.list('/managed-oo/disputes', normalizeDispute, page, pageSize, blockNumber);
    }

    private list<T extends ManagedOOBaseItem>(
        path: string,
        normalize: (item: any) => T,
        page: number,
        pageSize: number,
        blockNumber?: number
    ): Promise<ListManagedOOItemsResult<T>> & {abort?: () => void} {
        const query: Record<string, number> = {page, page_size: pageSize};
        if (blockNumber) {
            query.block_number = blockNumber;
        }
        const req = requests.get(path).query(query);
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalize),
                total: readNumber(body, 'total') || 0,
                page: readNumber(body, 'page') || page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || pageSize
            };
        }) as Promise<ListManagedOOItemsResult<T>> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
