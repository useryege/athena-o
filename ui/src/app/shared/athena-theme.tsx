import * as React from 'react';
import {App, ConfigProvider, theme} from 'antd';
import type {ThemeConfig} from 'antd';
import {StyleProvider, px2remTransformer} from '@ant-design/cssinjs';
import {createAthenaColorRoles} from './athena-color-roles';

const transformers = [px2remTransformer({rootValue: 16, mediaQuery: false})];

export function createAthenaTheme(): ThemeConfig {
    const style = getComputedStyle(document.documentElement);
    const read = (name: string) => style.getPropertyValue(`--athena-${name}`).trim();
    const roles = createAthenaColorRoles(read);
    return {
        cssVar: {key: 'athena-theme'},
        // 先派生完整深色 Map，再锁定最终角色；不能把已确认值仅当 seed 输入。
        algorithm: (seed, map) => ({...theme.darkAlgorithm(seed, map), ...roles}),
        token: {fontFamily: read('font'), fontSize: 14, controlHeight: 44, borderRadius: 10, motionDurationFast: '0.16s', motionDurationMid: '0.2s'},
        components: {
            Button: {
                primaryColor: read('bg'),
                dangerColor: read('bg'),
                primaryShadow: 'none',
                dangerShadow: 'none',
                defaultShadow: 'none',
                defaultColor: read('text'),
                defaultBg: read('panel'),
                defaultBorderColor: read('border-strong'),
                defaultHoverColor: read('text'),
                defaultHoverBg: read('panel-elevated'),
                defaultHoverBorderColor: read('border-strong'),
                defaultActiveColor: read('text'),
                defaultActiveBg: read('panel-elevated'),
                defaultActiveBorderColor: read('border-strong'),
                borderColorDisabled: read('border')
            },
            Input: {activeBorderColor: read('primary'), hoverBorderColor: read('primary'), activeShadow: 'none'},
            Segmented: {trackBg: read('panel'), itemSelectedBg: read('selected-bg'), itemSelectedColor: read('primary')},
            Pagination: {itemActiveBg: read('panel')},
            Alert: {defaultPadding: '16px 20px'}
        }
    };
}

export const AthenaThemeProvider = ({children}: {children: React.ReactNode}) => {
    const [config] = React.useState(createAthenaTheme);
    return (
        <StyleProvider transformers={transformers}>
            <ConfigProvider theme={config}>
                <App>{children}</App>
            </ConfigProvider>
        </StyleProvider>
    );
};
