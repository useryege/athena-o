import requests from './requests';
import {AccountDataModule} from '../access-modules';

const readScope = {module: AccountDataModule.WorldCupCorners, mode: 'read' as const};

export type WorldCupCornerStageKey = 'group-stage' | 'round-of-16' | 'quarter-finals' | 'semi-finals' | 'third-place' | 'final';

export interface WorldCupCornerStage {
    key: WorldCupCornerStageKey;
    label: string;
    shortLabel: string;
    knockout: boolean;
}

export interface WorldCupCornerMatch {
    id: number;
    stage: WorldCupCornerStageKey;
    homeTeam: string;
    awayTeam: string;
    homeScore: number;
    awayScore: number;
    hasPenaltyShootout: boolean;
    homePenaltyScore: number;
    awayPenaltyScore: number;
    homeCorners90: number;
    awayCorners90: number;
    homeCornersFull: number;
    awayCornersFull: number;
}

export interface WorldCupCornersDataset {
    stages: WorldCupCornerStage[];
    matches: WorldCupCornerMatch[];
}

const stage = (value: any): WorldCupCornerStage => ({
    key: (value?.key || '') as WorldCupCornerStageKey,
    label: value?.label || '',
    shortLabel: value?.shortLabel || '',
    knockout: Boolean(value?.knockout)
});

const match = (value: any): WorldCupCornerMatch => ({
    id: Number(value?.id ?? 0),
    stage: (value?.stage || '') as WorldCupCornerStageKey,
    homeTeam: value?.homeTeam || '',
    awayTeam: value?.awayTeam || '',
    homeScore: Number(value?.homeScore ?? 0),
    awayScore: Number(value?.awayScore ?? 0),
    hasPenaltyShootout: Boolean(value?.hasPenaltyShootout),
    homePenaltyScore: Number(value?.homePenaltyScore ?? 0),
    awayPenaltyScore: Number(value?.awayPenaltyScore ?? 0),
    homeCorners90: Number(value?.homeCorners90 ?? 0),
    awayCorners90: Number(value?.awayCorners90 ?? 0),
    homeCornersFull: Number(value?.homeCornersFull ?? 0),
    awayCornersFull: Number(value?.awayCornersFull ?? 0)
});

export class WorldCupCornersService {
    public getDataset(): Promise<WorldCupCornersDataset> & {abort?: () => void} {
        const req = requests.get('/world-cup-corners/dataset', readScope);
        const promise = req.then(res => ({
            stages: (res.body?.stages || []).map(stage),
            matches: (res.body?.matches || []).map(match)
        })) as Promise<WorldCupCornersDataset> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
