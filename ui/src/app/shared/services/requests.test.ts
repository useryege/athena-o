import requests from './requests';

afterEach(() => {
    jest.restoreAllMocks();
});

test('onError emits new request errors without replaying old ones', () => {
    (requests.get('/before-subscribe') as any).emit('error', {status: 401});

    const observed: number[] = [];
    const subscription = requests.onError.subscribe(err => observed.push(err.status));
    expect(observed).toEqual([]);

    (requests.get('/after-subscribe') as any).emit('error', {status: 401});

    expect(observed).toEqual([401]);
    subscription.unsubscribe();
});
