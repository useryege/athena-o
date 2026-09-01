import {AccountThemeMode} from './models';
import type {ThemeMode} from './services/view-preferences-service';

export const accountThemeLabel = (theme: AccountThemeMode | ThemeMode) => {
    if (theme === AccountThemeMode.Dark || theme === 'dark') {
        return 'Dark';
    }
    if (theme === AccountThemeMode.Light || theme === 'light') {
        return 'Light';
    }
    return 'System';
};

export const serverThemeMode = (theme: ThemeMode): AccountThemeMode => {
    if (theme === 'dark') {
        return AccountThemeMode.Dark;
    }
    if (theme === 'light') {
        return AccountThemeMode.Light;
    }
    return AccountThemeMode.System;
};

export const localThemeMode = (theme: AccountThemeMode): ThemeMode => {
    if (theme === AccountThemeMode.Dark) {
        return 'dark';
    }
    if (theme === AccountThemeMode.Light) {
        return 'light';
    }
    return 'system';
};
