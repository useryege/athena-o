import * as React from 'react';
import {createRoot} from 'react-dom/client';
import {Alert, Button, Input, Space} from 'antd';
import {AthenaThemeProvider} from '../../src/app/shared/athena-theme';
import '../../src/assets/fonts.css';
import '../../src/app/styles/shared.css';

createRoot(document.getElementById('app')!).render(
    <AthenaThemeProvider>
        <main style={{padding: '1.25rem', maxWidth: '50rem'}}>
            <p data-testid='body'>Account information</p>
            <Space wrap>
                <Button type='primary'>Save profile</Button>
                <Button type='primary' danger>Remove avatar</Button>
            </Space>
            <Input aria-label='Display name' defaultValue='Athena member' />
            <Input aria-label='Username' readOnly value='member-name' />
            {(['success', 'warning', 'error', 'info'] as const).map(type => <Alert key={type} type={type} showIcon title={`${type} status`} description='The latest account state is available.' />)}
        </main>
    </AthenaThemeProvider>
);
