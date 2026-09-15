import * as React from 'react';
import {createRoot} from 'react-dom/client';
import {Alert, App, Button, Checkbox, Dropdown, Input, InputNumber, Modal, Radio, Select, Space, Switch, Table} from 'antd';
import {AthenaThemeProvider} from '../../src/app/shared/athena-theme';
import '../../src/assets/fonts.css';
import '../../src/app/styles/shared.css';

const Controls = () => {
    const {message, notification} = App.useApp();
    const [open, setOpen] = React.useState(false);
    return (
        <main style={{padding: '1.25rem', maxWidth: '50rem'}}>
            <section aria-label='Selection control states' style={{display: 'grid', gap: '1rem', justifyItems: 'start'}}>
                <Checkbox aria-label='Checkbox toggle'>Checkbox toggle</Checkbox>
                <Checkbox aria-label='Checkbox disabled checked' disabled defaultChecked>
                    Checkbox disabled checked
                </Checkbox>
                <Checkbox aria-label='Checkbox disabled off' disabled>
                    Checkbox disabled off
                </Checkbox>
                <Radio.Group aria-label='Radio choices' defaultValue='off'>
                    <Radio value='on'>Radio on</Radio>
                    <Radio value='off'>Radio off</Radio>
                </Radio.Group>
                <Radio aria-label='Radio disabled checked' disabled checked>
                    Radio disabled checked
                </Radio>
                <Radio aria-label='Radio disabled off' disabled>
                    Radio disabled off
                </Radio>
                <Switch aria-label='Switch toggle' />
                <Switch aria-label='Switch disabled checked' disabled defaultChecked />
                <Switch aria-label='Switch disabled off' disabled />
            </section>
            <p data-testid='body'>Account information</p>
            <Space wrap>
                <Button type='primary'>Save profile</Button>
                <Button type='primary' danger>
                    Remove avatar
                </Button>
                <Button type='primary' disabled>
                    Disabled primary
                </Button>
            </Space>
            <Input aria-label='Display name' defaultValue='Athena member' />
            <Input aria-label='Username' readOnly value='member-name' />
            <Input aria-label='Disabled input' disabled value='Unavailable' />
            {(['small', 'middle', 'large'] as const).map(size => (
                <section key={size} data-testid={`size-${size}`} style={{display: 'grid', gap: '.75rem', marginBlock: '1rem'}}>
                    <Button size={size}>Size {size}</Button>
                    <Input size={size} aria-label={`Input ${size}`} defaultValue='Editable value' />
                    <InputNumber size={size} aria-label={`Number ${size}`} defaultValue={12} style={{width: '100%'}} />
                    <Select size={size} aria-label={`Select ${size}`} defaultValue='Selected' options={[{value: 'Selected'}, {value: 'Another option'}]} />
                    <Table size={size} rowKey='id' pagination={false} columns={[{title: 'Value', dataIndex: 'value'}]} dataSource={[{id: 1, value: 'Table content'}]} />
                </section>
            ))}
            <Space wrap>
                <Button onClick={() => setOpen(true)}>Open modal</Button>
                <Dropdown menu={{items: [{key: 'profile', label: 'Profile option'}]}} trigger={['click']}>
                    <Button>Open dropdown</Button>
                </Dropdown>
                <Button onClick={() => message.info({content: 'Message body', duration: 0})}>Show message</Button>
                <Button onClick={() => notification.info({title: 'Notification title', description: 'Notification body', duration: 0})}>Show notification</Button>
            </Space>
            <Modal open={open} onCancel={() => setOpen(false)} onOk={() => setOpen(false)} title='Modal title'>
                <p>Modal body</p>
            </Modal>
            {(['success', 'warning', 'error', 'info'] as const).map(type => (
                <Alert key={type} type={type} showIcon title={`${type} status`} description='The latest account state is available.' />
            ))}
        </main>
    );
};
createRoot(document.getElementById('app')!).render(
    <AthenaThemeProvider>
        <Controls />
    </AthenaThemeProvider>
);
