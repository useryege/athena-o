import {Alert, Button, Card, Empty, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {KeyValueGrid, ResourceTable, Section, TruncatedText, useAsyncData} from '../components';
import {formatBlockNumber} from '../shared/format';
import {DEFAULT_PAGE_SIZE} from '../shared/pagination';
import {services} from '../shared/services';
import {TokenProjectDetail, TokenProjectReportEvaluation, TokenProjectReportPairRisk, TokenReportRevision} from '../shared/services/token-service';
import {ProjectTimeValue as TimeValue} from './project-detail-values';
import {ProjectJSONDrawer, ProjectJSONDrawerValue} from './project-json-drawer';
import {chainAssetLabels} from './token-shared';

const reportMetadataColumns = {xs: 1, sm: 1, md: 2, lg: 3, xl: 3, xxl: 3} as const;

const createdTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'green' : undefined}>{value ? 'Created' : 'Not created'}</Tag>;
};

const riskTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'red' : 'green'}>{value ? 'Detected' : 'Clear'}</Tag>;
};

const completenessTag = (value?: string) => {
    if (!value) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value === 'complete' ? 'green' : 'gold'}>{value}</Tag>;
};

const evaluationTag = (value?: string) => {
    const colors: Record<string, string> = {pending: 'gold', running: 'blue', succeeded: 'green', failed: 'red'};
    return <Tag color={value ? colors[value] : undefined}>{value || 'No task'}</Tag>;
};

const outcomeTag = (value?: string) => (
    <Tag color={value === 'selected' ? 'green' : value === 'rejected' ? 'red' : value === 'deferred' ? 'gold' : undefined}>{value || 'No outcome'}</Tag>
);

const formatInteger = (value?: string) => (value ? value.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '-');

const pairItems = (pair?: TokenProjectReportPairRisk) => [
    {label: 'Created', value: createdTag(pair?.isCreated)},
    {label: 'Remove-liquidity risk', value: riskTag(pair?.isRemoveLiquidity)},
    {label: 'Mint risk', value: riskTag(pair?.isMint)},
    {label: 'Quote USDT', value: formatInteger(pair?.quoteUsdtValueInt)},
    {label: 'Last swap', value: <TimeValue value={pair?.lastSwapAt} />}
];

const ReportPairSnapshots = (props: {chainID?: number; wethPair?: TokenProjectReportPairRisk; usdtPair?: TokenProjectReportPairRisk; compact?: boolean}) => {
    const labels = chainAssetLabels(props.chainID);
    return (
        <div className='project-report-risk-snapshot'>
            {!props.wethPair && !props.usdtPair && (
                <Alert
                    type='warning'
                    title='Report risk snapshot unavailable'
                    description='This report revision does not contain a pair risk projection. Unknown values are preserved below.'
                    showIcon={true}
                />
            )}
            <div className={`project-report-pair-grid${props.compact ? ' project-report-pair-grid--compact' : ''}`}>
                <Card size='small' title={`${labels.wrapped} pair report snapshot`}>
                    {!props.wethPair && <Typography.Text type='secondary'>Risk unavailable</Typography.Text>}
                    <KeyValueGrid columns={props.compact ? 1 : 2} items={pairItems(props.wethPair)} />
                </Card>
                <Card size='small' title={`${labels.stable} pair report snapshot`}>
                    {!props.usdtPair && <Typography.Text type='secondary'>Risk unavailable</Typography.Text>}
                    <KeyValueGrid columns={props.compact ? 1 : 2} items={pairItems(props.usdtPair)} />
                </Card>
            </div>
        </div>
    );
};

const evaluationItems = (evaluation?: TokenProjectReportEvaluation) => [
    {label: 'Status', value: evaluationTag(evaluation?.status)},
    {label: 'Failed attempts', value: evaluation?.failedAttempts ?? '-'},
    {label: 'Last error', value: evaluation?.lastError ? <Typography.Text type='danger'>{evaluation.lastError}</Typography.Text> : '-'},
    {label: 'Updated', value: <TimeValue value={evaluation?.updatedAt} />},
    {label: 'Outcome', value: outcomeTag(evaluation?.outcome)},
    {label: 'Evaluated', value: <TimeValue value={evaluation?.evaluatedAt} />}
];

const RevisionCompactCard = (props: {revision: TokenReportRevision; currentRevision?: number; chainID?: number; openJSON: (content: ProjectJSONDrawerValue) => void}) => (
    <Card
        className='project-report-revision-card'
        size='small'
        title={
            <Space wrap={true} size={6}>
                <span>Report revision {props.revision.revision ?? '-'}</span>
                {props.revision.revision === props.currentRevision && <Tag color='blue'>Current</Tag>}
            </Space>
        }
        extra={completenessTag(props.revision.completenessStatus)}>
        <div className='project-report-revision-card__body'>
            <KeyValueGrid
                columns={2}
                items={[
                    {label: 'Observed block', value: formatBlockNumber(props.revision.observedBlockNumber)},
                    {label: 'Content hash', value: <TruncatedText value={props.revision.contentHash} copyable={true} />},
                    {label: 'Built', value: <TimeValue value={props.revision.builtAt} />},
                    {label: 'Created', value: <TimeValue value={props.revision.createdAt} />}
                ]}
            />
            <ReportPairSnapshots chainID={props.chainID} wethPair={props.revision.riskSummary?.wethPair} usdtPair={props.revision.riskSummary?.usdtPair} compact={true} />
            <Space wrap={true}>
                <Button
                    size='small'
                    disabled={!props.revision.evidenceJSON}
                    onClick={() => props.openJSON({title: `Report r${props.revision.revision} evidence`, value: props.revision.evidenceJSON || ''})}>
                    Evidence JSON
                </Button>
                <Button
                    size='small'
                    disabled={!props.revision.reportJSON}
                    onClick={() => props.openJSON({title: `Report r${props.revision.revision}`, value: props.revision.reportJSON || ''})}>
                    Report JSON
                </Button>
            </Space>
        </div>
    </Card>
);

export const ProjectReportTab = (props: {projectID: number; detail: TokenProjectDetail; refreshVersion: number}) => {
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(DEFAULT_PAGE_SIZE);
    const [drawer, setDrawer] = React.useState<ProjectJSONDrawerValue>();
    const history = useAsyncData(
        () => services.tokenapi.listReportRevisions({projectID: props.projectID, page, pageSize}),
        [props.projectID, page, pageSize, props.refreshVersion]
    );
    const report = props.detail.currentReport;
    const evaluation = props.detail.currentReportEvaluation;
    const chainID = props.detail.project?.chainID;
    const labels = chainAssetLabels(chainID);
    const columns: ColumnsType<TokenReportRevision> = [
        {
            title: 'Revision',
            fixed: 'left',
            width: 120,
            render: item => (
                <Space wrap={true} size={4}>
                    <span>{item.revision ?? '-'}</span>
                    {item.revision === report?.revision && <Tag color='blue'>Current</Tag>}
                </Space>
            )
        },
        {title: 'Completeness', width: 120, render: item => completenessTag(item.completenessStatus)},
        {title: 'Block', width: 120, render: item => formatBlockNumber(item.observedBlockNumber)},
        {title: 'Content Hash', width: 230, render: item => <TruncatedText value={item.contentHash} copyable={true} />},
        {title: 'Built', width: 175, render: item => <TimeValue value={item.builtAt} />},
        {title: 'Created', width: 175, render: item => <TimeValue value={item.createdAt} />},
        {
            title: `${labels.wrapped} Pair`,
            children: [
                {title: 'Created', width: 95, render: item => createdTag(item.riskSummary?.wethPair?.isCreated)},
                {title: 'Remove Liquidity', width: 120, render: item => riskTag(item.riskSummary?.wethPair?.isRemoveLiquidity)},
                {title: 'Mint', width: 90, render: item => riskTag(item.riskSummary?.wethPair?.isMint)},
                {title: 'Quote USDT', width: 135, render: item => formatInteger(item.riskSummary?.wethPair?.quoteUsdtValueInt)},
                {title: 'Last Swap', width: 175, render: item => <TimeValue value={item.riskSummary?.wethPair?.lastSwapAt} />}
            ]
        },
        {
            title: `${labels.stable} Pair`,
            children: [
                {title: 'Created', width: 95, render: item => createdTag(item.riskSummary?.usdtPair?.isCreated)},
                {title: 'Remove Liquidity', width: 120, render: item => riskTag(item.riskSummary?.usdtPair?.isRemoveLiquidity)},
                {title: 'Mint', width: 90, render: item => riskTag(item.riskSummary?.usdtPair?.isMint)},
                {title: 'Quote USDT', width: 135, render: item => formatInteger(item.riskSummary?.usdtPair?.quoteUsdtValueInt)},
                {title: 'Last Swap', width: 175, render: item => <TimeValue value={item.riskSummary?.usdtPair?.lastSwapAt} />}
            ]
        },
        {
            title: 'JSON',
            width: 210,
            render: item => (
                <Space wrap={true} size={4}>
                    <Button size='small' disabled={!item.evidenceJSON} onClick={() => setDrawer({title: `Report r${item.revision} evidence`, value: item.evidenceJSON || ''})}>
                        Evidence
                    </Button>
                    <Button size='small' disabled={!item.reportJSON} onClick={() => setDrawer({title: `Report r${item.revision}`, value: item.reportJSON || ''})}>
                        Report
                    </Button>
                </Space>
            )
        }
    ];

    return (
        <div className='project-detail-tab project-report-tab'>
            <Section title='Current report'>
                {!report ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No report has been built for this project' />
                ) : (
                    <div className='project-report-current'>
                        <KeyValueGrid
                            columns={reportMetadataColumns}
                            items={[
                                {label: 'Revision', value: report.revision ?? '-'},
                                {label: 'Completeness', value: completenessTag(report.completenessStatus)},
                                {label: 'Observed block', value: formatBlockNumber(report.observedBlockNumber)},
                                {label: 'Content hash', value: <TruncatedText value={report.contentHash} copyable={true} />},
                                {label: 'Built', value: <TimeValue value={report.builtAt} />},
                                {label: 'Created', value: <TimeValue value={report.createdAt} />}
                            ]}
                        />
                        <Card size='small' title='Current report evaluation'>
                            <KeyValueGrid columns={reportMetadataColumns} items={evaluationItems(evaluation)} />
                        </Card>
                    </div>
                )}
            </Section>
            <Section title='Report risk snapshot'>
                {!report ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No report risk snapshot is available because this project has no report' />
                ) : (
                    <ReportPairSnapshots chainID={chainID} wethPair={report.riskSummary?.wethPair} usdtPair={report.riskSummary?.usdtPair} />
                )}
            </Section>
            <Section title='Report & evidence JSON'>
                {!report ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No report JSON is available' />
                ) : (
                    <Space wrap={true}>
                        <Button disabled={!report.reportJSON} onClick={() => setDrawer({title: `Current report r${report.revision}`, value: report.reportJSON || ''})}>
                            Report JSON
                        </Button>
                        <Button disabled={!report.evidenceJSON} onClick={() => setDrawer({title: `Current report r${report.revision} evidence`, value: report.evidenceJSON || ''})}>
                            Evidence JSON
                        </Button>
                    </Space>
                )}
            </Section>
            <Section title='Revision history'>
                {history.error && !history.data ? (
                    <Alert type='error' title='Report revision history unavailable' description={history.error.message} showIcon={true} />
                ) : (
                    <>
                        {history.error && <Alert type='error' title='Report revision history refresh failed' description={history.error.message} showIcon={true} />}
                        <ResourceTable
                            label='Report revision history'
                            rowKey={item => item.reportRevisionID || `${item.revision}-${item.createdAt}`}
                            items={history.data?.items || []}
                            columns={columns}
                            loading={history.loading}
                            total={history.data?.total}
                            page={page}
                            pageSize={pageSize}
                            onPageChange={(nextPage, nextPageSize) => {
                                setPage(nextPage);
                                setPageSize(nextPageSize);
                            }}
                            scrollX={2510}
                            compactRender={item => <RevisionCompactCard revision={item} currentRevision={report?.revision} chainID={chainID} openJSON={setDrawer} />}
                            compactEmptyDescription='No report revisions have been built for this project'
                        />
                    </>
                )}
            </Section>
            <ProjectJSONDrawer content={drawer} onClose={() => setDrawer(undefined)} />
        </div>
    );
};
