import {Select, Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenAPIProject} from '../shared/services/tokenapi-service';
import {fmtNumber, useKeywordParam, usePagedParams} from './shared';
import {ChainBadge} from './token-shared';

export const ProjectsPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [chainID, setChainID] = React.useState<number>();
    const [contract, setContract] = useKeywordParam('contract');
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const options = useAsyncData(() => services.tokenapi.getOptions(), []);
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
    const columns: ColumnsType<TokenAPIProject> = [
        {title: 'ID', dataIndex: 'projectID'},
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {
            title: 'Token',
            render: item => (
                <Space direction='vertical' size={0}>
                    <Typography.Text strong={true}>{item.symbol || '-'}</Typography.Text>
                    <Typography.Text type='secondary'>{item.name || '-'}</Typography.Text>
                </Space>
            )
        },
        {title: 'Contract', render: item => <TruncatedText value={item.contract} copyable={true} />},
        {title: 'Creator', render: item => <TruncatedText value={item.creator} copyable={true} />},
        {title: 'Block', render: item => fmtNumber(item.blockNumber)},
        {title: 'Tx Index', dataIndex: 'txIndex'},
        {title: 'Code Hash', render: item => <TruncatedText value={item.codeHash} copyable={true} />},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    const chainOptions = (options.data?.chains || []).map(item => ({
        value: item.chainID,
        label: `${item.chainName || item.chainID} (${item.chainID})`
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
                    <Select
                        allowClear={true}
                        aria-label='Filter by chain'
                        value={chainID}
                        placeholder='Chain'
                        style={{width: 220}}
                        options={chainOptions}
                        onChange={value => {
                            setChainID(value);
                            setPage(1, pageSize);
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
            />
        </AppPage>
    );
};
