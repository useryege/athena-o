import {Space, Typography} from 'antd';
import {AppPage, Section} from '../components';

export const HelpPage = () => (
    <AppPage title='Help' subtitle='Operational reference links'>
        <Section title='Resources'>
            <Space orientation='vertical'>
                <a href='swagger-ui'>Swagger UI</a>
                <Typography.Text type='secondary'>ATHENA mobile UI is optimized for browsing, search, detail inspection, and common operations.</Typography.Text>
            </Space>
        </Section>
    </AppPage>
);
