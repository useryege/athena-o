import type {ColumnsType} from 'antd/es/table';
import {Link} from 'react-router-dom';
import {AppPage, ResourceTable, SearchBar, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenAPIContractCode} from '../shared/services/tokenapi-service';
import {short, useKeywordParam, usePagedParams} from './shared';

export const ContractCodesPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const data = useAsyncData(() => services.tokenapi.listContractCodes({page, pageSize, codeHash: codeHash || undefined}), [page, pageSize, codeHash]);
    const columns: ColumnsType<TokenAPIContractCode> = [
        {title: 'Code Hash', render: item => <Link to={`/token/contract-codes/${encodeURIComponent(item.codeHash || '')}`}>{short(item.codeHash)}</Link>},
        {title: 'Deployments', dataIndex: 'deploymentCount'},
        {title: 'Fetched', dataIndex: 'sourceCodeFetchedAt'},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    return (
        <AppPage
            title='Contract Codes'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={<SearchBar value={codeHash} onChange={setCodeHash} placeholder='Code hash' />}>
            <ResourceTable
                rowKey={item => item.codeHash || Math.random()}
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
