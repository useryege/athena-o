import {AthenaApplicationService} from './athena-application-service';

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

describe('athena application service', () => {
    beforeEach(() => {
        mockGet.mockReset();
        mockPost.mockReset();
        mockDelete.mockReset();
    });

    it('manages wallet blacklist through application-prefixed endpoints', async () => {
        mockGet.mockReturnValue(requestWithBody({items: [{wallet: '0xabc', note: 'seed', created_at: 'now'}]}));
        const list = await new AthenaApplicationService().listWalletBlacklistEntries();
        expect(mockGet).toHaveBeenCalledWith('/application/wallet/blacklist');
        expect(list).toEqual([{wallet: '0xabc', note: 'seed', createdAt: 'now'}]);

        const addReq = requestWithBody({item: {wallet: '0xabc', note: 'seed'}});
        mockPost.mockReturnValue(addReq);
        await new AthenaApplicationService().addWalletBlacklistEntry('0xabc', 'seed');
        expect(mockPost).toHaveBeenCalledWith('/application/wallet/blacklist');
        expect(addReq.send).toHaveBeenCalledWith({wallet: '0xabc', note: 'seed'});

        const updateReq = requestWithBody({item: {wallet: '0xabc', note: 'updated'}});
        mockPost.mockReturnValue(updateReq);
        await new AthenaApplicationService().updateWalletBlacklistEntryNote('0xabc', 'updated');
        expect(mockPost).toHaveBeenCalledWith('/application/wallet/blacklist/0xabc/note');
        expect(updateReq.send).toHaveBeenCalledWith({wallet: '0xabc', note: 'updated'});

        const deleteReq = requestWithBody({});
        mockDelete.mockReturnValue(deleteReq);
        await new AthenaApplicationService().deleteWalletBlacklistEntry('0xabc');
        expect(mockDelete).toHaveBeenCalledWith('/application/wallet/blacklist/0xabc');
    });
});
