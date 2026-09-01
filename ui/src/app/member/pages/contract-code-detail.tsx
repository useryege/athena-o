import {Tabs} from 'antd';
import {useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, TruncatedText, useAsyncData} from '../../components';
import {formatBeijingDateTime} from '../../shared/format';
import {memberServices as services} from '../services';
import {boolTag, fmtNumber} from '../../shared/pages/shared';

export const ContractCodeDetailPage = () => {
    const {codeHash = ''} = useParams();
    const decoded = decodeURIComponent(codeHash);
    const detail = useAsyncData(() => services.tokenapi.getContractCode(decoded), [decoded]);
    return (
        <AppPage title='Contract Code Detail' subtitle={<TruncatedText value={decoded} copyable={true} />} loading={detail.loading} error={detail.error} onRefresh={detail.reload}>
            <Section title='Summary'>
                <KeyValueGrid
                    items={[
                        {label: 'Code Hash', value: <TruncatedText value={decoded} copyable={true} />},
                        {label: 'Found', value: boolTag(!!detail.data)},
                        {label: 'Deployments', value: fmtNumber(detail.data?.deploymentCount)},
                        {label: 'Fetched', value: formatBeijingDateTime(detail.data?.sourceCodeFetchedAt) || '-'},
                        {label: 'Created', value: formatBeijingDateTime(detail.data?.createdAt) || '-'}
                    ]}
                />
            </Section>
            <Tabs items={[{key: 'source', label: 'Source', children: <pre className='code-block'>{detail.data?.sourceCode || 'No source available'}</pre>}]} />
        </AppPage>
    );
};
