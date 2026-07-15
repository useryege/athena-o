import {Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, ChoiceGroup, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenProject} from '../shared/services/token-service';
import {fmtNumber, useKeywordParam, usePagedParams} from './shared';
import {ChainBadge, chainLabel} from './token-shared';

export const ProjectsPage = () => {
    const {params, setParams, page, pageSize, setPage} = usePagedParams();
    const chainID = Number(params.get('chainID') || params.get('chain_id')) || undefined;
    const [contract, setContract] = useKeywordParam('contract');
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const setChainFilter = (value?: number | null) => {
        const next = new URLSearchParams(params);
        if (value === undefined || value === null) {
            next.delete('chainID');
        } else {
            next.set('chainID', String(value));
        }
        next.delete('chain_id');
        next.delete('page_size');
        next.set('page', '1');
        next.set('pageSize', String(pageSize));
        setParams(next);
    };
    const options = useAsyncData(() => services.tokenapi.getRuntimeConfiguration(), []);
    const data = useAsyncData(
        () =>
            services.tokenapi.listProjects({
                page,
                pageSize,
                chainID,
                contract: contract || undefined,
                codeHash: codeHash || undefined
            }),
        [page, pageSize, chainID, contract, codeHash]
    );
    const columns: ColumnsType<TokenProject> = [
        {title: 'ID', dataIndex: 'projectID'},
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {
            title: 'Token',
            render: item => (
                <Space orientation='vertical' size={0}>
                    <Typography.Text strong={true}>{item.symbol || '-'}</Typography.Text>
                    <Typography.Text type='secondary'>{item.name || '-'}</Typography.Text>
                </Space>
            )
        },
        {title: 'Contract', render: item => <TruncatedText value={item.contract} copyable={true} />},
        {title: 'Tx Sender', render: item => <TruncatedText value={item.txSender} copyable={true} />},
        {title: 'Block', render: item => fmtNumber(item.blockNumber)},
        {title: 'Tx Index', dataIndex: 'txIndex'},
        {title: 'Code Hash', render: item => <TruncatedText value={item.codeHash} copyable={true} />},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    const chainOptions = (options.data?.chains || [])
        .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
        .map(item => ({
            value: item.chainID,
            label: chainLabel(item.chainID)
        }));
    return (
        <AppPage
            title='Projects'
            loading={data.loading || options.loading}
            error={data.error || options.error}
            onRefresh={() => {
                data.reload();
                options.reload();
            }}
            filters={
                <Space wrap={true}>
                    <ChoiceGroup<number | 'all'>
                        ariaLabel='Filter by chain'
                        value={chainID ?? 'all'}
                        options={[{label: 'All', value: 'all'}, ...chainOptions]}
                        onChange={value => {
                            setChainFilter(value === 'all' ? undefined : value);
                        }}
                    />
                    <SearchBar value={contract} onChange={setContract} placeholder='Contract' />
                    <SearchBar value={codeHash} onChange={setCodeHash} placeholder='Code hash' />
                </Space>
            }>
            <ResourceTable
                rowKey={item => item.projectID || `${item.chainID}-${item.contract}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                scrollX={1500}
            />
        </AppPage>
    );
};
