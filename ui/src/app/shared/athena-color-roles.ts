import type {GlobalToken} from 'antd';

export function createAthenaColorRoles(read: (name: string) => string): Partial<GlobalToken> {
    const roles = {
        colorPrimary: read('primary'),
        colorPrimaryHover: read('primary-hover'),
        colorPrimaryActive: read('primary-active'),
        colorPrimaryBg: read('selected-bg'),
        colorPrimaryBgHover: read('selected-bg'),
        colorPrimaryBorder: read('primary'),
        colorPrimaryBorderHover: read('primary-hover'),
        colorPrimaryText: read('primary'),
        colorPrimaryTextHover: read('primary-hover'),
        colorPrimaryTextActive: read('primary-active'),
        colorBgBase: read('bg'),
        colorBgLayout: read('bg'),
        colorBgContainer: read('panel'),
        colorBgElevated: read('panel-elevated'),
        colorBgContainerDisabled: read('panel-elevated'),
        colorBgTextHover: read('panel-elevated'),
        colorBgTextActive: read('selected-bg'),
        colorTextBase: read('text'),
        colorText: read('text'),
        colorTextSecondary: read('muted'),
        colorTextTertiary: read('muted'),
        colorTextQuaternary: read('muted'),
        colorTextDisabled: read('muted'),
        colorTextPlaceholder: read('muted'),
        colorTextLightSolid: read('bg'),
        colorBorder: read('border-strong'),
        colorBorderSecondary: read('border'),
        colorSplit: read('border'),
        colorLink: read('primary'),
        colorLinkHover: read('primary-hover'),
        colorLinkActive: read('primary-active')
    };
    for (const [role, name] of [
        ['Success', 'green'],
        ['Warning', 'amber'],
        ['Error', 'red'],
        ['Info', 'blue']
    ]) {
        const base = `color${role}`;
        Object.assign(roles, {
            [base]: read(name),
            [`${base}Bg`]: read(`${name}-bg`),
            [`${base}BgHover`]: read(`${name}-bg`),
            [`${base}Border`]: read(`${name}-border`),
            [`${base}BorderHover`]: read(`${name}-border`),
            [`${base}Text`]: read(name),
            [`${base}TextHover`]: read(name),
            [`${base}TextActive`]: read(name),
            [`${base}Hover`]: read(name),
            [`${base}Active`]: read(name)
        });
    }
    Object.assign(roles, {
        colorErrorHover: read('red-hover'),
        colorErrorActive: read('red-active'),
        colorErrorTextHover: read('red-hover'),
        colorErrorTextActive: read('red-active')
    });
    return roles;
}
