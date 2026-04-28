import {NotificationType} from 'argo-ui';
import * as React from 'react';

import {DataLoader, ErrorNotification, Page} from '../../../shared/components';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';

require('./node-grpc-url.scss');

const NodeGrpcURLEditor = ({initialNodeGrpcURL, reload}: {initialNodeGrpcURL: string; reload: () => void}) => {
    const ctx = React.useContext(Context);
    const [nodeGrpcURL, setNodeGrpcURL] = React.useState(initialNodeGrpcURL || '');
    const [isSaving, setIsSaving] = React.useState(false);

    React.useEffect(() => {
        setNodeGrpcURL(initialNodeGrpcURL || '');
    }, [initialNodeGrpcURL]);

    const saveNodeGrpcURL = async () => {
        const normalizedURL = nodeGrpcURL.trim();
        setIsSaving(true);
        try {
            const updatedNodeGrpcURL = await services.blocksniffer.setNodeGrpcURL(normalizedURL);
            setNodeGrpcURL(updatedNodeGrpcURL || normalizedURL);
            ctx.notifications.show({
                type: NotificationType.Success,
                content: 'Node gRPC URL has been updated.'
            });
            reload();
        } catch (e) {
            ctx.notifications.show({
                content: <ErrorNotification title='Unable to update Node gRPC URL' e={e} />,
                type: NotificationType.Error
            });
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <div className='node-grpc-url'>
            <div className='argo-container'>
                <div className='node-grpc-url__panel'>
                    <div className='argo-form-row'>
                        <label className='node-grpc-url__label'>Node gRPC URL</label>
                        <input
                            className='argo-field'
                            type='text'
                            value={nodeGrpcURL}
                            placeholder='https://example.com:443'
                            onChange={event => setNodeGrpcURL(event.target.value)}
                            disabled={isSaving}
                        />
                    </div>
                    <div className='node-grpc-url__actions'>
                        <button className='argo-button argo-button--base' onClick={saveNodeGrpcURL} disabled={isSaving || nodeGrpcURL.trim() === ''}>
                            Save
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export const NodeGrpcURL = () => {
    const loaderRef = React.useRef<DataLoader>();
    return (
        <Page
            title='Node gRPC URL'
            toolbar={{
                breadcrumbs: [{title: 'Settings', path: '/settings'}, {title: 'Node gRPC URL'}]
            }}>
            <DataLoader ref={loader => (loaderRef.current = loader)} load={() => services.blocksniffer.getNodeGrpcURL()}>
                {nodeGrpcURL => <NodeGrpcURLEditor initialNodeGrpcURL={nodeGrpcURL} reload={() => loaderRef.current && loaderRef.current.reload()} />}
            </DataLoader>
        </Page>
    );
};
