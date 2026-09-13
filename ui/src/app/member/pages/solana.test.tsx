import {Button, Input} from 'antd';
import renderer, {act} from 'react-test-renderer';
import {SolanaPage} from './solana';
import {ensureMemberBusinessServices, memberServices as services} from '../services';

let tree: renderer.ReactTestRenderer;

const project = {
    mint: 'SoLaNaMint+/address',
    tokenProgram: 'TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb',
    signature: 'transaction-signature',
    feePayer: 'payer-address',
    mintAuthority: 'mint-authority',
    freezeAuthority: '',
    decimals: 9,
    slot: '18446744073709551615',
    blockTime: '1720000000',
    discoveredAt: '1720000010'
};

const status = {
    status: 'catching_up',
    startSlot: '18446744073709551000',
    lastProcessedSlot: '18446744073709551010',
    latestFinalizedSlot: '18446744073709551020',
    lastSuccessAt: '1720000020',
    totalProjects: '1',
    lastError: ''
};

const flush = async () => {
    await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
    });
};

beforeEach(() => {
    ensureMemberBusinessServices();
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    jest.spyOn(services.solana, 'listProjects').mockResolvedValue({items: [project], totalSize: 26, page: 1, pageSize: 25});
    jest.spyOn(services.solana, 'getDiscoveryStatus').mockResolvedValue(status);
});

afterEach(() => {
    if (tree) {
        act(() => tree.unmount());
    }
    jest.restoreAllMocks();
});

test('loads saved candidates and scanner status with safely preserved large slots', async () => {
    await act(async () => {
        tree = renderer.create(<SolanaPage />);
    });
    await flush();

    expect(services.solana.listProjects).toHaveBeenCalledWith({page: 1, pageSize: 25, query: ''});
    expect(services.solana.getDiscoveryStatus).toHaveBeenCalledTimes(1);
    const text = JSON.stringify(tree.toJSON());
    expect(text).toContain('Solana');
    expect(text).toContain('Newly initialized token candidates');
    expect(tree.root.findAll(node => node.type === 'td' && node.children.includes('Token-2022'))).toHaveLength(1);
    expect(text).toContain('Table pagination');
    expect(text).not.toContain('Items per page');
    const expand = tree.root.findAll(node => String(node.props.className || '').includes('ant-table-row-expand-icon') && typeof node.props.onClick === 'function')[0];
    await act(async () => expand.props.onClick({stopPropagation: jest.fn()}));
    expect(JSON.stringify(tree.toJSON())).toContain('18446744073709551615');
});

test('shows the empty state and a recoverable request failure', async () => {
    jest.mocked(services.solana.listProjects).mockResolvedValueOnce({items: [], totalSize: 0, page: 1, pageSize: 25}).mockRejectedValueOnce(new Error('Solana read unavailable'));
    await act(async () => {
        tree = renderer.create(<SolanaPage />);
    });
    await flush();
    expect(JSON.stringify(tree.toJSON())).toContain('No discovered candidates yet');

    await act(async () => {
        tree.root.findByProps({'aria-label': 'Refresh data'}).props.onClick();
    });
    await flush();
    expect(JSON.stringify(tree.toJSON())).toContain('Solana read unavailable');
});

test('refreshes, searches, paginates, copies addresses, and only opens encoded Solscan links', async () => {
    const writeText = jest.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {clipboard: {writeText}});
    await act(async () => {
        tree = renderer.create(<SolanaPage />);
    });
    await flush();

    const query = tree.root.findAllByType(Input).find(item => item.props['aria-label'] === 'Mint address')!;
    await act(async () => query.props.onChange({target: {value: 'Mint+query'}}));
    await act(async () => query.props.onPressEnter());
    expect(services.solana.listProjects).toHaveBeenLastCalledWith({page: 1, pageSize: 25, query: 'Mint+query'});

    await act(async () => tree.root.findByProps({'aria-label': 'Refresh data'}).props.onClick());
    expect(services.solana.listProjects).toHaveBeenCalledTimes(3);
    expect(services.solana.getDiscoveryStatus).toHaveBeenCalledTimes(2);

    const next = tree.root.findAll(node => node.type === 'button' && node.parent?.props.title === 'Next Page')[0];
    await act(async () => next.parent!.props.onClick());
    expect(services.solana.listProjects).toHaveBeenLastCalledWith({page: 2, pageSize: 25, query: 'Mint+query'});

    const copy = tree.root.findAllByType(Button).find(item => item.props['aria-label'] === `Copy mint ${project.mint}`)!;
    await act(async () => copy.props.onClick());
    expect(writeText).toHaveBeenCalledWith(project.mint);
    const links = tree.root.findAll(node => node.type === 'a' && node.props.href);
    expect(links.map(link => link.props.href)).toContain(`https://solscan.io/token/${encodeURIComponent(project.mint)}`);
    expect(links.map(link => link.props.href)).toContain(`https://solscan.io/tx/${encodeURIComponent(project.signature)}`);
});


test('distinguishes a Mint query without matches and bounds its input', async () => {
    await act(async () => { tree = renderer.create(<SolanaPage />); });
    await flush();
    const input = tree.root.findAllByType(Input).find(item => item.props['aria-label'] === 'Mint address')!;
    expect(input.props.maxLength).toBe(128);
    jest.mocked(services.solana.listProjects).mockResolvedValueOnce({items: [], totalSize: 0, page: 1, pageSize: 25});
    await act(async () => input.props.onChange({target: {value: 'unmatched'}}));
    await act(async () => input.props.onPressEnter());
    await flush();
    expect(JSON.stringify(tree.toJSON())).toContain('No matching Mint');
    expect(JSON.stringify(tree.toJSON())).not.toContain('No discovered candidates yet');
});
