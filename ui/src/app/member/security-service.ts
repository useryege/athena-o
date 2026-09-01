import type {Token} from '../shared/models';
import requests from '../shared/services/requests';

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

const apiKeyReadScope = {feature: 'api-key' as const, mode: 'read' as const};
const apiKeyWriteScope = {feature: 'api-key' as const, mode: 'write' as const};

const token = (value: any): Token => ({
    id: String(value?.id || ''),
    issuedAt: Number(value?.issuedAt ?? value?.issued_at ?? 0),
    expiresAt: Number(value?.expiresAt ?? value?.expires_at ?? 0)
});

export class MemberSecurityService {
    public listTokens(): AbortablePromise<Token[]> {
        const req = requests.get('/account/security/tokens', apiKeyReadScope);
        const promise = req.then(res => (res.body?.items || []).map(token)) as AbortablePromise<Token[]>;
        promise.abort = () => req.abort();
        return promise;
    }

    public createToken(tokenId: string, expiresIn: number): Promise<string> {
        return requests
            .post('/account/security/tokens', apiKeyWriteScope)
            .send({expiresIn, id: tokenId})
            .then(res => String(res.body?.token || ''));
    }

    public deleteToken(id: string): Promise<void> {
        return requests.delete(`/account/security/tokens/${encodeURIComponent(id)}`, apiKeyWriteScope).then(() => undefined);
    }
}
