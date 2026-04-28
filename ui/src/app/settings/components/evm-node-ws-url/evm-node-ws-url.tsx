import {NotificationType} from 'argo-ui';
import * as React from 'react';

import {DataLoader, ErrorNotification, Page} from '../../../shared/components';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';

require('./evm-node-ws-url.scss');

const EvmNodeWsURLEditor = ({initialEvmNodeWsURL, reload}: {initialEvmNodeWsURL: string; reload: () => void}) => {
    const ctx = React.useContext(Context);
    const [evmNodeWsURL, setEvmNodeWsURL] = React.useState(initialEvmNodeWsURL || '');
    const [isSaving, setIsSaving] = React.useState(false);

    React.useEffect(() => {
        setEvmNodeWsURL(initialEvmNodeWsURL || '');
    }, [initialEvmNodeWsURL]);

    const saveEvmNodeWsURL = async () => {
        const normalizedURL = evmNodeWsURL.trim();
        setIsSaving(true);
        try {
            const updatedEvmNodeWsURL = await services.blocksniffer.setEvmNodeWsURL(normalizedURL);
            setEvmNodeWsURL(updatedEvmNodeWsURL || normalizedURL);
            ctx.notifications.show({
                type: NotificationType.Success,
                content: 'EVM Node WS URL has been updated.'
            });
            reload();
        } catch (e) {
            ctx.notifications.show({
                content: <ErrorNotification title='Unable to update EVM Node WS URL' e={e} />,
                type: NotificationType.Error
            });
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <div className='evm-node-ws-url'>
            <div className='argo-container'>
                <div className='evm-node-ws-url__panel'>
                    <div className='argo-form-row'>
                        <label className='evm-node-ws-url__label'>EVM Node WS URL</label>
                        <input
                            className='argo-field'
                            type='text'
                            value={evmNodeWsURL}
                            placeholder='wss://example.com:8546'
                            onChange={event => setEvmNodeWsURL(event.target.value)}
                            disabled={isSaving}
                        />
                    </div>
                    <div className='evm-node-ws-url__actions'>
                        <button className='argo-button argo-button--base' onClick={saveEvmNodeWsURL} disabled={isSaving || evmNodeWsURL.trim() === ''}>
                            Save
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export const EvmNodeWsURL = () => {
    const loaderRef = React.useRef<DataLoader>();
    return (
        <Page
            title='EVM Node WS URL'
            toolbar={{
                breadcrumbs: [{title: 'Settings', path: '/settings'}, {title: 'EVM Node WS URL'}]
            }}>
            <DataLoader ref={loader => (loaderRef.current = loader)} load={() => services.blocksniffer.getEvmNodeWsURL()}>
                {evmNodeWsURL => <EvmNodeWsURLEditor initialEvmNodeWsURL={evmNodeWsURL} reload={() => loaderRef.current && loaderRef.current.reload()} />}
            </DataLoader>
        </Page>
    );
};
