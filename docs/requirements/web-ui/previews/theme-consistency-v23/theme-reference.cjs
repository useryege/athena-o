// 可运行的设计参考；由 build.cjs 使用实际安装的 Ant Design 验证。
// 正式 Provider 按此映射实现 TypeScript 版本；read 只读取根 CSS token。
function createAthenaColorRoles(read) {
  const roles = {
    colorPrimary: read('primary'), colorPrimaryHover: read('primary-hover'), colorPrimaryActive: read('primary-active'),
    colorPrimaryBg: read('selected-bg'), colorPrimaryBgHover: read('selected-bg'),
    colorPrimaryBorder: read('primary'), colorPrimaryBorderHover: read('primary-hover'),
    colorPrimaryText: read('primary'), colorPrimaryTextHover: read('primary-hover'), colorPrimaryTextActive: read('primary-active'),
    colorBgBase: read('bg'), colorBgLayout: read('bg'), colorBgContainer: read('panel'), colorBgElevated: read('panel-elevated'),
    colorBgContainerDisabled: read('panel-elevated'), colorBgTextHover: read('panel-elevated'), colorBgTextActive: read('selected-bg'),
    colorTextBase: read('text'), colorText: read('text'), colorTextSecondary: read('muted'),
    colorTextTertiary: read('muted'), colorTextQuaternary: read('muted'), colorTextDisabled: read('muted'), colorTextPlaceholder: read('muted'),
    colorTextLightSolid: read('bg'), colorBorder: read('border-strong'), colorBorderSecondary: read('border'), colorSplit: read('border'),
    colorLink: read('primary'), colorLinkHover: read('primary-hover'), colorLinkActive: read('primary-active'),
  };
  for (const [role, name] of [['Success', 'green'], ['Warning', 'amber'], ['Error', 'red'], ['Info', 'blue']]) {
    const base = `color${role}`;
    Object.assign(roles, {
      [base]: read(name), [`${base}Bg`]: read(`${name}-bg`), [`${base}BgHover`]: read(`${name}-bg`),
      [`${base}Border`]: read(`${name}-border`), [`${base}BorderHover`]: read(`${name}-border`),
      [`${base}Text`]: read(name), [`${base}TextHover`]: read(name), [`${base}TextActive`]: read(name),
      [`${base}Hover`]: read(name), [`${base}Active`]: read(name),
    });
  }
  Object.assign(roles, {
    colorErrorHover: read('red-hover'), colorErrorActive: read('red-active'),
    colorErrorTextHover: read('red-hover'), colorErrorTextActive: read('red-active'),
  });
  return roles;
}
function createAthenaTheme(theme, read) {
  const roles = createAthenaColorRoles(read);
  return {
    cssVar: {key: 'athena-theme'},
    // 先派生完整深色 Map，再锁定最终角色；不能把已确认值仅当 seed 输入。
    algorithm: (seed, map) => ({...theme.darkAlgorithm(seed, map), ...roles}),
    token: {fontFamily: read('font'), fontSize: 14, controlHeight: 44, borderRadius: 10,
      motionDurationFast: '0.16s', motionDurationMid: '0.2s'},
    components: {
      Button: {primaryColor: read('bg'), dangerColor: read('bg'), primaryShadow: 'none', dangerShadow: 'none', defaultShadow: 'none',
        defaultColor: read('text'), defaultBg: read('panel'), defaultBorderColor: read('border-strong'),
        defaultHoverColor: read('text'), defaultHoverBg: read('panel-elevated'), defaultHoverBorderColor: read('border-strong'),
        defaultActiveColor: read('text'), defaultActiveBg: read('panel-elevated'), defaultActiveBorderColor: read('border-strong'),
        borderColorDisabled: read('border')},
      Input: {activeBorderColor: read('primary'), hoverBorderColor: read('primary'), activeShadow: 'none'},
      Segmented: {trackBg: read('panel'), itemSelectedBg: read('selected-bg'), itemSelectedColor: read('primary')},
      Pagination: {itemActiveBg: read('panel')},
      Alert: {defaultPadding: '16px 20px'},
    },
  };
}
module.exports = {createAthenaColorRoles, createAthenaTheme};
