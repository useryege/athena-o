import {PlayCircleOutlined, StopOutlined} from '@ant-design/icons';
import {Button, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, StatusTag, useAsyncData} from '../components';
import {formatBeijingDateTime, formatBlockNumber} from '../shared/format';
import {services} from '../shared/services';
import {TokenChainCheckpoint} from '../shared/services/token-service';
import {boolTag} from './shared';
import {ChainBadge} from './token-shared';

type ChainProcessingStatus = 'running' | 'stopped';

export const ChainCheckpointsPage = () => {
    const data = useAsyncData(() => services.tokenapi.listChainCheckpoints(), []);
    const [updatingStatusByChainID, setUpdatingStatusByChainID] = React.useState<Record<number, ChainProcessingStatus>>({});
    const updateStatus = async (item: TokenChainCheckpoint, status: ChainProcessingStatus) => {
        if (item.chainID === undefined) {
            return;
        }
        const chainID = item.chainID;
        setUpdatingStatusByChainID(current => ({...current, [chainID]: status}));
        try {
            await services.tokenapi.updateChainCheckpoint(chainID, status);
            data.reload();
        } finally {
            setUpdatingStatusByChainID(current => {
                const next = {...current};
                delete next[chainID];
                return next;
            });
        }
    };
    const columns: ColumnsType<TokenChainCheckpoint> = [
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {title: 'Enabled', render: item => boolTag(item.enabled)},
        {title: 'Cursor', render: item => formatBlockNumber(item.cursorBlockNumber)},
        {title: 'Current Status', render: item => <StatusTag value={item.status} positive={item.status === 'running'} />},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'},
        {
            title: 'Actions',
            render: item => {
                const updatingStatus = item.chainID === undefined ? undefined : updatingStatusByChainID[item.chainID];
                const rowUpdating = updatingStatus !== undefined;
                const actionDisabled = item.chainID === undefined || rowUpdating;
                return (
                    <Space.Compact>
                        <Button
                            size='small'
                            icon={<PlayCircleOutlined />}
                            disabled={actionDisabled || item.status === 'running'}
                            loading={updatingStatus === 'running'}
                            aria-label={`Start chain ${item.chainID ?? ''} ingest`}
                            onClick={() => void updateStatus(item, 'running')}>
                            Start
                        </Button>
                        <Button
                            size='small'
                            danger={true}
                            icon={<StopOutlined />}
                            disabled={actionDisabled || item.status === 'stopped'}
                            loading={updatingStatus === 'stopped'}
                            aria-label={`Stop chain ${item.chainID ?? ''} ingest`}
                            onClick={() => void updateStatus(item, 'stopped')}>
                            Stop
                        </Button>
                    </Space.Compact>
                );
            }
        }
    ];
    return (
        <AppPage title='Chain Checkpoints' loading={data.loading} error={data.error} onRefresh={data.reload}>
            <ResourceTable rowKey={item => item.chainID ?? Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
        </AppPage>
    );
};
