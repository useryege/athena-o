import {AccountDataModule} from '../access-modules';
import {readNumber, readString, readValue} from './api-values';
import requests from './requests';

const readScope = {module: AccountDataModule.Solana, mode: 'read' as const};

export interface SolanaProject {
    mint: string;
    tokenProgram: string;
    signature: string;
    feePayer: string;
    mintAuthority: string;
    freezeAuthority: string;
    decimals: number;
    slot: string;
    blockTime: string;
    discoveredAt: string;
    name: string;
    symbol: string;
    metadataStatus: string;
    metadataSource: string;
    metadataAccount: string;
    metadataObservedSlot: string;
    metadataUpdatedAt: string;
    issuanceSource: string;
    issuanceProgram: string;
    sourceStatus: string;
}

export interface SolanaProjectPage {
    items: SolanaProject[];
    totalSize: number;
    page: number;
    pageSize: number;
}

export interface SolanaDiscoveryStatus {
    status: string;
    startSlot: string;
    lastProcessedSlot: string;
    latestFinalizedSlot: string;
    lastSuccessAt: string;
    totalProjects: string;
    lastError: string;
}

export interface ListSolanaProjectsOptions {
    page: number;
    pageSize: number;
    query: string;
}

const readIntegerText = (item: unknown, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (typeof value === 'number') {
        return Number.isSafeInteger(value) ? String(value) : '';
    }
    if (typeof value === 'string' && /^-?\d+$/.test(value)) {
        return value;
    }
    return '';
};

const normalizeProject = (item: unknown): SolanaProject => ({
    mint: readString(item, 'mint'),
    tokenProgram: readString(item, 'tokenProgram', 'token_program'),
    signature: readString(item, 'signature'),
    feePayer: readString(item, 'feePayer', 'fee_payer'),
    mintAuthority: readString(item, 'mintAuthority', 'mint_authority'),
    freezeAuthority: readString(item, 'freezeAuthority', 'freeze_authority'),
    decimals: readNumber(item, 'decimals') || 0,
    slot: readIntegerText(item, 'slot'),
    blockTime: readIntegerText(item, 'blockTime', 'block_time'),
    discoveredAt: readIntegerText(item, 'discoveredAt', 'discovered_at'),
    name: readString(item, 'name'),
    symbol: readString(item, 'symbol'),
    metadataStatus: readString(item, 'metadataStatus', 'metadata_status'),
    metadataSource: readString(item, 'metadataSource', 'metadata_source'),
    metadataAccount: readString(item, 'metadataAccount', 'metadata_account'),
    metadataObservedSlot: readIntegerText(item, 'metadataObservedSlot', 'metadata_observed_slot'),
    metadataUpdatedAt: readIntegerText(item, 'metadataUpdatedAt', 'metadata_updated_at'),
    issuanceSource: readString(item, 'issuanceSource', 'issuance_source'),
    issuanceProgram: readString(item, 'issuanceProgram', 'issuance_program'),
    sourceStatus: readString(item, 'sourceStatus', 'source_status')
});

const normalizeStatus = (item: unknown): SolanaDiscoveryStatus => ({
    status: readString(item, 'status'),
    startSlot: readIntegerText(item, 'startSlot', 'start_slot'),
    lastProcessedSlot: readIntegerText(item, 'lastProcessedSlot', 'last_processed_slot'),
    latestFinalizedSlot: readIntegerText(item, 'latestFinalizedSlot', 'latest_finalized_slot'),
    lastSuccessAt: readIntegerText(item, 'lastSuccessAt', 'last_success_at'),
    totalProjects: readIntegerText(item, 'totalProjects', 'total_projects'),
    lastError: readString(item, 'lastError', 'last_error')
});

export class SolanaService {
    public listProjects(options: ListSolanaProjectsOptions): Promise<SolanaProjectPage> & {abort?: () => void} {
        const req = requests.get('/solana/projects', readScope).query(options);
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: Array.isArray(body.items) ? body.items.map(normalizeProject) : [],
                totalSize: readNumber(body, 'totalSize', 'total_size') || 0,
                page: readNumber(body, 'page') || options.page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || options.pageSize
            };
        }) as Promise<SolanaProjectPage> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public getDiscoveryStatus(): Promise<SolanaDiscoveryStatus> & {abort?: () => void} {
        const req = requests.get('/solana/status', readScope);
        const promise = req.then(res => normalizeStatus(res.body || {})) as Promise<SolanaDiscoveryStatus> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
