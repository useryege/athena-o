import {ApiOutlined, DownloadOutlined, FileTextOutlined, MessageOutlined, SafetyCertificateOutlined} from '@ant-design/icons';
import {Button, Space, Typography} from 'antd';
import {useNavigate} from 'react-router-dom';
import {AppPage, Section} from '../../components';
import {useAuthorization} from '../context';
import {AuthSettings} from '../models';
import {deploymentPath} from '../runtime-base';

export const HelpPage = (props: {help: AuthSettings['help']}) => {
    const authorization = useAuthorization();
    const navigate = useNavigate();
    const mayConnectAI = authorization.user.access.apiKeyEnabled && !authorization.isAdmin;

    return (
        <AppPage title='Help' subtitle='Operational reference links'>
            <Section title='Resources'>
                <Space className='help-resource-list' orientation='vertical'>
                    {mayConnectAI && (
                        <Button className='help-connect-ai-action' type='primary' icon={<ApiOutlined />} onClick={() => navigate('/account/security')}>
                            Connect an AI
                        </Button>
                    )}
                    {props.help?.chatUrl && (
                        <Button type='link' icon={<MessageOutlined />} href={props.help.chatUrl} target='_blank' rel='noreferrer'>
                            {props.help.chatText || 'Contact support'}
                        </Button>
                    )}
                    <Button type='link' icon={<FileTextOutlined />} href={deploymentPath('llms.txt')} target='_blank' rel='noreferrer'>
                        LLM discovery (llms.txt)
                    </Button>
                    <Button type='link' icon={<SafetyCertificateOutlined />} href={deploymentPath('docs/ai/safety.md')} target='_blank' rel='noreferrer'>
                        Full-Account AI Access
                    </Button>
                    <Button type='link' href={deploymentPath('swagger-ui')}>
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
};
