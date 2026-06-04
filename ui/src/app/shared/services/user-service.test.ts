import requests from './requests';
import {UserService} from './user-service';

const originalFetch = global.fetch;

afterEach(() => {
    global.fetch = originalFetch;
    requests.setBaseHRef('/');
    jest.restoreAllMocks();
});

test('logout clears the local auth cookie without following the server redirect', async () => {
    const fetchMock = jest.fn().mockResolvedValue({status: 303, statusText: 'See Other'});
    global.fetch = fetchMock as any;

    await expect(new UserService().logout()).resolves.toBe(true);

    expect(fetchMock).toHaveBeenCalledWith('/auth/logout', {credentials: 'same-origin', redirect: 'manual'});
});

test('logout reports server failures', async () => {
    const fetchMock = jest.fn().mockResolvedValue({status: 500, statusText: 'Server Error'});
    global.fetch = fetchMock as any;

    await expect(new UserService().logout()).rejects.toThrow('Server Error');
});
