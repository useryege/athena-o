import type {ColumnsType} from 'antd/es/table';
import {Link, useNavigate} from 'react-router-dom';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, SearchBar, TruncatedText, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {TokenAPIContractCode} from '../../shared/services/tokenapi-service';
import {short, useKeywordParam, usePagedParams} from './shared';

export const ContractCodesPage = () => {
    const navigate = useNavigate();
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
            <ResponsiveResourceList
                rowKey={item => item.codeHash || Math.random()}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <div onClick={() => navigate(`/token/contract-codes/${encodeURIComponent(item.codeHash || '')}`)}>
                        <CardTitle title={short(item.codeHash)} subtitle={<TruncatedText value={item.codeHash} copyable={true} />} />
                        <MetricRow
                            items={[
                                {label: 'Deployments', value: item.deploymentCount},
                                {label: 'Fetched', value: item.sourceCodeFetchedAt},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                    </div>
                )}
            />
        </AppPage>
    );
};
