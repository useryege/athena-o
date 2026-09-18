import {ExportOutlined} from '@ant-design/icons';
import {Empty, Typography} from 'antd';
import {AppPage, Section} from '../../components';
import {AuthSettings} from '../models';

export const HelpPage = (props: {help: AuthSettings['help']}) => {
    const binaryUrls = Object.entries(props.help?.binaryUrls || {});
    const hasResources = Boolean(props.help?.chatUrl || binaryUrls.length);

    return (
        <AppPage title='Help' subtitle='Operational reference links'>
            <div className='athena-help-panel'>
                <Section title='Resources'>
                    <section className='athena-help-resources' aria-label='Help resources'>
                        {!hasResources ? (
                            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No help resources configured' />
                        ) : (
                            <>
                                {props.help?.chatUrl && (
                                    <a className='athena-help-resource' href={props.help.chatUrl} target='_blank' rel='noreferrer'>
                                        <strong>{props.help.chatText || 'Contact support'}</strong>
                                        <ExportOutlined />
                                    </a>
                                )}
                                {binaryUrls.map(([label, url]) => (
                                    <a className='athena-help-resource' key={label} href={url}>
                                        <strong>{label}</strong>
                                        <ExportOutlined />
                                    </a>
                                ))}
                            </>
                        )}
                        <Typography.Paragraph className='athena-help-note' type='secondary'>
                            ATHENA desktop UI is optimized for browsing, search, detail inspection, and operational workflows.
                        </Typography.Paragraph>
                    </section>
                </Section>
            </div>
        </AppPage>
    );
};
