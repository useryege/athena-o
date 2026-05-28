import {WalletService} from './wallet-service';

const mockGet = jest.fn();
const mockPost = jest.fn();
const mockDelete = jest.fn();

jest.mock('./requests', () => ({
    __esModule: true,
    default: {
        get: (...args: any[]) => mockGet(...args),
        post: (...args: any[]) => mockPost(...args),
        delete: (...args: any[]) => mockDelete(...args)
    }
}));

const requestWithBody = (body: any) => {
    const request: any = {
        abort: jest.fn(),
        query: jest.fn(() => request),
        send: jest.fn(() => request),
        then: (resolve: any) => Promise.resolve(resolve({body}))
    };
    return request;
};

describe('wallet service', () => {
    beforeEach(() => {
        mockGet.mockReset();
        mockPost.mockReset();
        mockDelete.mockReset();
    });

    it('lists wallets with filters', async () => {
        const request = requestWithBody({
            items: [{id: '7', chain: 'ETH', address: '0xabc', derivation_path: "m/44'/60'/0'/0/0"}],
            total: '1',
            page: 2,
            page_size: 20
        });
        mockGet.mockReturnValue(request);

        const result = await new WalletService().listWallets({chain: 'ETH', query: 'main', page: 2, pageSize: 20});

        expect(mockGet).toHaveBeenCalledWith('/wallets');
        expect(request.query).toHaveBeenCalledWith({page: 2, page_size: 20, chain: 'ETH', query: 'main'});
        expect(result).toMatchObject({
            total: 1,
            page: 2,
            pageSize: 20,
            items: [{id: 7, chain: 'ETH', derivationPath: "m/44'/60'/0'/0/0"}]
        });
    });

    it('reveals details through an explicit query flag', async () => {
        const request = requestWithBody({item: {id: 3, private_key: 'secret', mnemonic: 'words'}});
        mockGet.mockReturnValue(request);

        const result = await new WalletService().getWallet(3, true);

        expect(mockGet).toHaveBeenCalledWith('/wallets/3');
        expect(request.query).toHaveBeenCalledWith({reveal_secrets: true});
        expect(result.privateKey).toBe('secret');
        expect(result.mnemonic).toBe('words');
    });

    it('imports private keys without putting secrets in the URL', async () => {
        const request = requestWithBody({item: {id: 5, chain: 'SOLANA'}});
        mockPost.mockReturnValue(request);

        await new WalletService().importPrivateKey('SOLANA', 'secret', 'ops');

        expect(mockPost).toHaveBeenCalledWith('/wallets/import-private-key');
        expect(request.send).toHaveBeenCalledWith({chain: 'SOLANA', private_key: 'secret', alias: 'ops'});
    });

    it('manages wallet blacklist through wallet-prefixed endpoints', async () => {
        mockGet.mockReturnValue(requestWithBody({items: [{wallet: '0xabc', note: 'seed', created_at: 'now'}]}));
        const list = await new WalletService().listWalletBlacklistEntries();
        expect(mockGet).toHaveBeenCalledWith('/wallet/blacklist');
        expect(list).toEqual([{wallet: '0xabc', note: 'seed', createdAt: 'now'}]);

        const addReq = requestWithBody({item: {wallet: '0xabc', note: 'seed'}});
        mockPost.mockReturnValue(addReq);
        await new WalletService().addWalletBlacklistEntry('0xabc', 'seed');
        expect(mockPost).toHaveBeenCalledWith('/wallet/blacklist');
        expect(addReq.send).toHaveBeenCalledWith({wallet: '0xabc', note: 'seed'});

        const updateReq = requestWithBody({item: {wallet: '0xabc', note: 'updated'}});
        mockPost.mockReturnValue(updateReq);
        await new WalletService().updateWalletBlacklistEntryNote('0xabc', 'updated');
        expect(mockPost).toHaveBeenCalledWith('/wallet/blacklist/0xabc/note');
        expect(updateReq.send).toHaveBeenCalledWith({wallet: '0xabc', note: 'updated'});

        const deleteReq = requestWithBody({});
        mockDelete.mockReturnValue(deleteReq);
        await new WalletService().deleteWalletBlacklistEntry('0xabc');
        expect(mockDelete).toHaveBeenCalledWith('/wallet/blacklist/0xabc');
    });
});
