import {ApiOutlined, ExportOutlined, RightOutlined} from '@ant-design/icons';
import {Button, Typography} from 'antd';
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
            <div className='athena-help-panel'>
                <Section title='Resources'>
                    <section className='athena-help-resources' aria-label='Help resources'>
                        {mayConnectAI && (
                            <div className='athena-help-connect'>
                                <Typography.Text type='secondary'>Manage AI connections and API keys in Account Center.</Typography.Text>
                                <Button className='help-connect-ai-action' type='primary' icon={<ApiOutlined />} onClick={() => navigate('/account/security')}>
                                    Connect an AI
                                </Button>
                            </div>
                        )}
                        {props.help?.chatUrl && (
                            <a className='athena-help-resource' href={props.help.chatUrl} target='_blank' rel='noreferrer'>
                                <strong>{props.help.chatText || 'Contact support'}</strong>
                                <ExportOutlined />
                            </a>
                        )}
                        <a className='athena-help-resource' href={deploymentPath('llms.txt')} target='_blank' rel='noreferrer'>
                            <span>
                                <strong>LLM discovery (llms.txt)</strong>
                                <small>Discover Athena resources for AI clients.</small>
                            </span>
                            <ExportOutlined />
                        </a>
                        <a className='athena-help-resource' href={deploymentPath('docs/ai/safety.md')} target='_blank' rel='noreferrer'>
                            <span>
                                <strong>Full-Account AI Access</strong>
                                <small>Read the scope and responsibilities of AI access.</small>
                            </span>
                            <ExportOutlined />
                        </a>
                        <a className='athena-help-resource' href={deploymentPath('swagger-ui')}>
                            <span>
                                <strong>Swagger UI</strong>
                                <small>Browse the available API reference.</small>
                            </span>
                            <RightOutlined />
                        </a>
                        {Object.entries(props.help?.binaryUrls || {}).map(([label, url]) => (
                            <a className='athena-help-resource' key={label} href={url}>
                                <strong>{label}</strong>
                                <ExportOutlined />
                            </a>
                        ))}
                        <Typography.Paragraph className='athena-help-note' type='secondary'>
                            ATHENA desktop UI is optimized for browsing, search, detail inspection, and operational workflows.
                        </Typography.Paragraph>
                    </section>
                </Section>
            </div>
        </AppPage>
    );
};
