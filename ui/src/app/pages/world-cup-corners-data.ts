export type WorldCupCornerStageKey = 'group-stage' | 'round-of-16' | 'quarter-finals' | 'semi-finals' | 'third-place' | 'final';

export interface WorldCupCornerMatch {
    id: number;
    stage: WorldCupCornerStageKey;
    homeTeam: string;
    awayTeam: string;
    homeScore: number;
    awayScore: number;
    homePenaltyScore?: number;
    awayPenaltyScore?: number;
    homeCorners90: number;
    awayCorners90: number;
    homeCornersFull: number;
    awayCornersFull: number;
}

export const worldCupCornerStages: Array<{key: WorldCupCornerStageKey; label: string; shortLabel: string; knockout: boolean}> = [
    {key: 'group-stage', label: 'Group Stage (32 → 16)', shortLabel: 'Group Stage', knockout: false},
    {key: 'round-of-16', label: 'Round of 16', shortLabel: 'Round of 16', knockout: true},
    {key: 'quarter-finals', label: 'Quarter-finals', shortLabel: 'Quarter-finals', knockout: true},
    {key: 'semi-finals', label: 'Semi-finals', shortLabel: 'Semi-finals', knockout: true},
    {key: 'third-place', label: 'Third-place Match', shortLabel: 'Third-place', knockout: true},
    {key: 'final', label: 'Final', shortLabel: 'Final', knockout: true}
];

const match = (
    id: number,
    stage: WorldCupCornerStageKey,
    homeTeam: string,
    awayTeam: string,
    score: [number, number],
    corners90: [number, number],
    cornersFull: [number, number] = corners90,
    penalties?: [number, number]
): WorldCupCornerMatch => ({
    id,
    stage,
    homeTeam,
    awayTeam,
    homeScore: score[0],
    awayScore: score[1],
    homePenaltyScore: penalties?.[0],
    awayPenaltyScore: penalties?.[1],
    homeCorners90: corners90[0],
    awayCorners90: corners90[1],
    homeCornersFull: cornersFull[0],
    awayCornersFull: cornersFull[1]
});

export const worldCupCornerMatches: WorldCupCornerMatch[] = [
    match(1, 'group-stage', 'Qatar', 'Ecuador', [0, 2], [1, 3]),
    match(2, 'group-stage', 'England', 'Iran', [6, 2], [8, 0]),
    match(3, 'group-stage', 'Senegal', 'Netherlands', [0, 2], [6, 7]),
    match(4, 'group-stage', 'United States', 'Wales', [1, 1], [5, 3]),
    match(5, 'group-stage', 'Argentina', 'Saudi Arabia', [1, 2], [9, 2]),
    match(6, 'group-stage', 'Denmark', 'Tunisia', [0, 0], [11, 9]),
    match(7, 'group-stage', 'Mexico', 'Poland', [0, 0], [6, 5]),
    match(8, 'group-stage', 'France', 'Australia', [4, 1], [8, 1]),
    match(9, 'group-stage', 'Morocco', 'Croatia', [0, 0], [0, 5]),
    match(10, 'group-stage', 'Germany', 'Japan', [1, 2], [6, 6]),
    match(11, 'group-stage', 'Spain', 'Costa Rica', [7, 0], [5, 0]),
    match(12, 'group-stage', 'Belgium', 'Canada', [1, 0], [4, 4]),
    match(13, 'group-stage', 'Switzerland', 'Cameroon', [1, 0], [11, 5]),
    match(14, 'group-stage', 'Uruguay', 'South Korea', [0, 0], [4, 3]),
    match(15, 'group-stage', 'Portugal', 'Ghana', [3, 2], [3, 3]),
    match(16, 'group-stage', 'Brazil', 'Serbia', [2, 0], [5, 4]),
    match(17, 'group-stage', 'Wales', 'Iran', [0, 2], [2, 7]),
    match(18, 'group-stage', 'Qatar', 'Senegal', [1, 3], [6, 6]),
    match(19, 'group-stage', 'Netherlands', 'Ecuador', [1, 1], [2, 5]),
    match(20, 'group-stage', 'England', 'United States', [0, 0], [3, 7]),
    match(21, 'group-stage', 'Tunisia', 'Australia', [0, 1], [5, 2]),
    match(22, 'group-stage', 'Poland', 'Saudi Arabia', [2, 0], [4, 5]),
    match(23, 'group-stage', 'France', 'Denmark', [2, 1], [6, 4]),
    match(24, 'group-stage', 'Argentina', 'Mexico', [2, 0], [4, 2]),
    match(25, 'group-stage', 'Japan', 'Costa Rica', [0, 1], [5, 0]),
    match(26, 'group-stage', 'Belgium', 'Morocco', [0, 2], [9, 1]),
    match(27, 'group-stage', 'Croatia', 'Canada', [4, 1], [5, 2]),
    match(28, 'group-stage', 'Spain', 'Germany', [1, 1], [6, 5]),
    match(29, 'group-stage', 'Cameroon', 'Serbia', [3, 3], [4, 3]),
    match(30, 'group-stage', 'South Korea', 'Ghana', [2, 3], [12, 5]),
    match(31, 'group-stage', 'Brazil', 'Switzerland', [1, 0], [8, 3]),
    match(32, 'group-stage', 'Portugal', 'Uruguay', [2, 0], [6, 2]),
    match(33, 'group-stage', 'Netherlands', 'Qatar', [2, 0], [4, 2]),
    match(34, 'group-stage', 'Ecuador', 'Senegal', [1, 2], [3, 6]),
    match(35, 'group-stage', 'Iran', 'United States', [0, 1], [1, 5]),
    match(36, 'group-stage', 'Wales', 'England', [0, 3], [1, 6]),
    match(37, 'group-stage', 'Tunisia', 'France', [1, 0], [7, 8]),
    match(38, 'group-stage', 'Australia', 'Denmark', [1, 0], [2, 6]),
    match(39, 'group-stage', 'Saudi Arabia', 'Mexico', [1, 2], [1, 8]),
    match(40, 'group-stage', 'Poland', 'Argentina', [0, 2], [1, 8]),
    match(41, 'group-stage', 'Canada', 'Morocco', [1, 2], [6, 2]),
    match(42, 'group-stage', 'Croatia', 'Belgium', [0, 0], [2, 4]),
    match(43, 'group-stage', 'Japan', 'Spain', [2, 1], [0, 2]),
    match(44, 'group-stage', 'Costa Rica', 'Germany', [2, 4], [1, 14]),
    match(45, 'group-stage', 'Ghana', 'Uruguay', [0, 2], [5, 2]),
    match(46, 'group-stage', 'South Korea', 'Portugal', [2, 1], [5, 4]),
    match(47, 'group-stage', 'Serbia', 'Switzerland', [2, 3], [2, 0]),
    match(48, 'group-stage', 'Cameroon', 'Brazil', [1, 0], [3, 11]),
    match(49, 'round-of-16', 'Netherlands', 'United States', [3, 1], [4, 5]),
    match(50, 'round-of-16', 'Argentina', 'Australia', [2, 1], [1, 3]),
    match(51, 'round-of-16', 'France', 'Poland', [3, 1], [7, 1]),
    match(52, 'round-of-16', 'England', 'Senegal', [3, 0], [3, 3]),
    match(53, 'round-of-16', 'Japan', 'Croatia', [1, 1], [5, 4], [8, 5], [1, 3]),
    match(54, 'round-of-16', 'Brazil', 'South Korea', [4, 1], [5, 4]),
    match(55, 'round-of-16', 'Morocco', 'Spain', [0, 0], [0, 4], [0, 8], [3, 0]),
    match(56, 'round-of-16', 'Portugal', 'Switzerland', [6, 1], [6, 6]),
    match(57, 'quarter-finals', 'Croatia', 'Brazil', [1, 1], [2, 5], [3, 7], [4, 2]),
    match(58, 'quarter-finals', 'Netherlands', 'Argentina', [2, 2], [2, 1], [2, 8], [3, 4]),
    match(59, 'quarter-finals', 'Morocco', 'Portugal', [1, 0], [3, 9]),
    match(60, 'quarter-finals', 'England', 'France', [1, 2], [5, 2]),
    match(61, 'semi-finals', 'Argentina', 'Croatia', [3, 0], [2, 4]),
    match(62, 'semi-finals', 'France', 'Morocco', [2, 0], [2, 3]),
    match(63, 'third-place', 'Croatia', 'Morocco', [2, 1], [6, 3]),
    match(64, 'final', 'Argentina', 'France', [3, 3], [4, 3], [6, 5], [4, 2])
];
