import {CopyOutlined} from '@ant-design/icons';
import {App, Button, Drawer, Empty} from 'antd';
import * as React from 'react';

export interface ProjectJSONDrawerValue {
    title: string;
    value: string;
}

export const ProjectJSONDrawer = (props: {content?: ProjectJSONDrawerValue; onClose: () => void}) => {
    const {message} = App.useApp();
    const raw = props.content?.value || '';
    const open = Boolean(props.content);
    const wasOpen = React.useRef(false);
    const restoreFocusRef = React.useRef<HTMLElement | null>(null);

    if (open && !wasOpen.current && document.activeElement instanceof HTMLElement) {
        restoreFocusRef.current = document.activeElement;
    }

    React.useLayoutEffect(() => {
        wasOpen.current = open;
    }, [open]);

    const copy = async () => {
        try {
            if (!navigator.clipboard?.writeText) {
                throw new Error('Clipboard access is unavailable');
            }
            await navigator.clipboard.writeText(raw);
            message.success('JSON copied');
        } catch {
            message.error('Could not copy JSON');
        }
    };

    return (
        <Drawer
            className='project-json-drawer'
            title={props.content?.title || 'Raw JSON'}
            width='min(760px, calc(100vw - 24px))'
            open={open}
            onClose={props.onClose}
            keyboard={true}
            afterOpenChange={nextOpen => {
                if (!nextOpen) {
                    const target = restoreFocusRef.current;
                    restoreFocusRef.current = null;
                    window.requestAnimationFrame(() => {
                        if (target?.isConnected) {
                            target.focus();
                        }
                    });
                }
            }}
            extra={
                raw ? (
                    <Button icon={<CopyOutlined />} onClick={copy}>
                        Copy
                    </Button>
                ) : undefined
            }>
            {raw ? (
                <pre className='code-block project-json-viewer' tabIndex={0} aria-label={`${props.content?.title || 'Raw JSON'} content`}>
                    {raw}
                </pre>
            ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No JSON payload' />
            )}
        </Drawer>
    );
};
