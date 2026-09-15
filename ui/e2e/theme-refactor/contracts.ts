export interface ThemeReply {
    method: string;
    path: string;
    realm: 'member' | 'admin';
    status: number;
    json: unknown;
    delayMs?: number;
}

export interface ThemeCase {
    id: string;
    route: string;
    realm: 'member' | 'admin';
    heading: string;
    replies: ThemeReply[];
    /** Anonymous registration authority belongs to its HttpOnly ticket, not the URL. */
    registrationTicket?: {realm: 'member' | 'admin'};
}

export interface ThemeLedger {
    requests: Array<{method: string; path: string; query: string; realm: string | undefined; body: unknown}>;
    unexpected: string[];
}
