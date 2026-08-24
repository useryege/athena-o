import {DownloadOutlined, MessageOutlined} from '@ant-design/icons';
import {Button, Space, Typography} from 'antd';
import {AppPage, Section} from '../components';
import {AuthSettings} from '../shared/models';

export const HelpPage = (props: {help: AuthSettings['help']}) => (
    <AppPage title='Help' subtitle='Operational reference links'>
        <Section title='Resources'>
            <Space className='help-resource-list' orientation='vertical'>
                {props.help?.chatUrl && (
                    <Button type='link' icon={<MessageOutlined />} href={props.help.chatUrl} target='_blank' rel='noreferrer'>
                        {props.help.chatText || 'Contact support'}
                    </Button>
                )}
                <Button type='link' href='swagger-ui'>
                    Swagger UI
                </Button>
                {Object.entries(props.help?.binaryUrls || {}).map(([label, url]) => (
                    <Button key={label} type='link' icon={<DownloadOutlined />} href={url}>
                        {label}
                    </Button>
                ))}
                <Typography.Text type='secondary'>ATHENA desktop UI is optimized for browsing, search, detail inspection, and operational workflows.</Typography.Text>
            </Space>
        </Section>
    </AppPage>
);
