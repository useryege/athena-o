import renderer, {act} from 'react-test-renderer';
import {StyleProvider} from '@ant-design/cssinjs';
import {Button} from 'antd';
import {MemoryRouter} from 'react-router-dom';
import {AuthorizationCtx, Context} from '../../shared/context';
import {parseUserInfo} from '../../shared/models';
import {ensureMemberBusinessServices, memberServices as services} from '../services';
import {WormTradingCombinationsPage} from './worm-trading-combinations';
import cases from '../../../../e2e/theme-refactor/fixtures/worm-assets-combinations.json';

const replies = cases.find(item => item.id === 'worm-combinations')!.replies as any[];
const rawUser = replies[0].json.session.userInfo;
const list = replies.find(item => item.path.endsWith('/combinations')).json;
const item = list.items[0];
const authorization = (accountId = rawUser.accountId, iss = 'issuer-A', revision = 1, writable = true) => ({
    user: parseUserInfo({...rawUser, accountId, iss}),
    revision,
    canWrite: () => writable,
    canRead: () => true,
    access: () => 2,
    isAdmin: false,
    lastCheckedAt: 0,
    refresh: async () => {}
});
let tree: renderer.ReactTestRenderer;
let dialogs: {onOk: () => Promise<void>; onCancel?: () => void; destroy: jest.Mock}[];
const notifications = {success: jest.fn(), error: jest.fn(), warning: jest.fn(), info: jest.fn()};
const view = (auth = authorization(), visible = true) => (
    <StyleProvider mock='server'>
        <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
            <AuthorizationCtx.Provider value={auth}>
                <Context.Provider
                    value={{
                        notifications,
                        modal: {
                            confirm: options => {
                                const dialog = {...options, destroy: jest.fn()};
                                dialogs.push(dialog as any);
                                return dialog;
                            },
                            info: jest.fn(),
                            error: jest.fn()
                        },
                        navigation: {goto: jest.fn(), replace: jest.fn()},
                        baseHref: ''
                    }}
                >
                    {visible && <WormTradingCombinationsPage />}
                </Context.Provider>
            </AuthorizationCtx.Provider>
        </MemoryRouter>
    </StyleProvider>
);
const update = async (auth = authorization(), visible = true) => {
    await act(async () => (tree ? tree.update(view(auth, visible)) : (tree = renderer.create(view(auth, visible)))));
};
const open = () => {
    act(() =>
        tree.root
            .findAllByType(Button)
            .find(button => button.props.children === 'Delete')!
            .props.onClick()
    );
    return dialogs[dialogs.length - 1];
};
beforeEach(() => {
    dialogs = [];
    ensureMemberBusinessServices();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(services.wormTrading, 'listMarketCombinations').mockResolvedValue(list);
    jest.spyOn(services.wormTrading, 'deleteMarketCombination').mockResolvedValue(undefined);
    Object.values(notifications).forEach(fn => fn.mockClear());
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    tree = undefined;
    jest.restoreAllMocks();
});
for (const transition of ['unmount', 'issuer', 'account', 'revision', 'revoke-restore', 'A-B-A']) {
    test(`delete confirmation is destroyed and its retained onOk cannot write after ${transition}`, async () => {
        await update();
        const old = open();
        if (transition === 'unmount') await update(authorization(), false);
        if (transition === 'issuer') await update(authorization(rawUser.accountId, 'issuer-B'));
        if (transition === 'account' || transition === 'A-B-A') await update(authorization('account-B'));
        if (transition === 'revision') await update(authorization(rawUser.accountId, 'issuer-A', 2));
        if (transition === 'revoke-restore') await update(authorization(rawUser.accountId, 'issuer-A', 1, false));
        if (transition === 'revoke-restore' || transition === 'A-B-A') await update();
        // Keep and invoke the actual page callback even if the modal was destroyed.
        await act(async () => old.onOk());
        expect(services.wormTrading.deleteMarketCombination).not.toHaveBeenCalled();
        expect(old.destroy).toHaveBeenCalledTimes(1);
        expect(notifications.success).not.toHaveBeenCalled();
        expect(notifications.error).not.toHaveBeenCalled();
    });
}
test('one confirmation admits a single DELETE with the captured revision, including synchronous duplicate clicks', async () => {
    let resolve!: () => void;
    jest.mocked(services.wormTrading.deleteMarketCombination).mockReturnValue(new Promise<void>(done => (resolve = done)));
    await update();
    const dialog = open();
    let pending!: Promise<void>;
    act(() => {
        pending = dialog.onOk();
        void dialog.onOk();
    });
    expect(services.wormTrading.deleteMarketCombination).toHaveBeenCalledTimes(1);
    expect(services.wormTrading.deleteMarketCombination).toHaveBeenCalledWith(item.id, item.revision);
    await act(async () => {
        resolve();
        await pending;
    });
    await act(async () => dialog.onOk());
    expect(services.wormTrading.deleteMarketCombination).toHaveBeenCalledTimes(1);
    expect(notifications.success).toHaveBeenCalledWith('Combination deleted', item.name);
    expect(services.wormTrading.listMarketCombinations).toHaveBeenCalledTimes(2);
});
for (const status of [409, 503]) {
    test(`delete ${status} preserves the list and permits a fresh confirmation`, async () => {
        jest.mocked(services.wormTrading.deleteMarketCombination).mockRejectedValueOnce({status, response: {body: {message: 'Controlled failure'}}});
        await update();
        await act(async () => open().onOk());
        expect(notifications.error.mock.calls[0][0]).toBe(status === 409 ? 'Combination changed' : 'Could not delete combination');
        expect(services.wormTrading.listMarketCombinations).toHaveBeenCalledTimes(1);
        expect(JSON.stringify(tree.toJSON())).toContain(item.name);
        await act(async () => open().onOk());
        expect(services.wormTrading.deleteMarketCombination).toHaveBeenCalledTimes(2);
        expect(notifications.success).toHaveBeenCalledTimes(1);
    });
}
for (const result of ['success', 'failure']) {
    test(`late delete ${result} cannot publish or clear the replacement scope's pending state`, async () => {
        let settle!: () => void;
        jest.mocked(services.wormTrading.deleteMarketCombination).mockReturnValueOnce(
            new Promise<void>((resolve, reject) => (settle = () => (result === 'success' ? resolve() : reject(new Error('Old failure')))))
        );
        await update();
        let pending!: Promise<void>;
        act(() => {
            pending = open().onOk();
        });
        await update(authorization('account-B'));
        jest.mocked(services.wormTrading.deleteMarketCombination).mockReturnValueOnce(new Promise<void>(() => {}));
        act(() => {
            void open().onOk();
        });
        await act(async () => {
            settle();
            await pending;
        });
        expect(notifications.success).not.toHaveBeenCalled();
        expect(notifications.error).not.toHaveBeenCalled();
        expect(tree.root.findAllByType(Button).find(button => button.props.children === 'Delete')!.props.loading).toBe(true);
    });
}
test('cancelled confirmation cannot submit and a fresh confirmation can open', async () => {
    await update();
    const cancelled = open();
    act(() => cancelled.onCancel!());
    await act(async () => cancelled.onOk());
    expect(services.wormTrading.deleteMarketCombination).not.toHaveBeenCalled();
    await act(async () => open().onOk());
    expect(services.wormTrading.deleteMarketCombination).toHaveBeenCalledTimes(1);
});
test('closing an already submitted dialog retains single-flight until its request finishes', async () => {
    let resolve!: () => void;
    jest.mocked(services.wormTrading.deleteMarketCombination).mockReturnValueOnce(new Promise<void>(done => (resolve = done)));
    await update();
    const dialog = open();
    let pending!: Promise<void>;
    act(() => {
        pending = dialog.onOk();
        dialog.onCancel!();
        open();
    });
    expect(dialogs).toHaveLength(1);
    await act(async () => {
        resolve();
        await pending;
    });
    expect(tree.root.findAllByType(Button).find(button => button.props.children === 'Delete')!.props.loading).toBe(false);
});

test('retirement preserves readable combinations while Trading write access removes delete', async () => {
    await update(authorization(rawUser.accountId, 'issuer-A', 2, false));
    expect(JSON.stringify(tree.toJSON())).toContain(item.name);
    expect(tree.root.findAllByType(Button).filter(button => button.props.children === 'Delete')).toHaveLength(0);
    expect(services.wormTrading.deleteMarketCombination).not.toHaveBeenCalled();
});
