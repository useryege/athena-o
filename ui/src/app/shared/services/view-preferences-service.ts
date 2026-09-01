import deepMerge from 'deepmerge';
import {BehaviorSubject, Observable} from 'rxjs';

export type ThemeMode = 'system' | 'dark' | 'light';
export type ResolvedTheme = 'dark' | 'light';

export interface ViewPreferences {
    version: number;
    pageSizes: {[key: string]: number};
    sortOptions?: {[key: string]: string};
    hideBannerContent: string;
    hideSidebar: boolean;
    position: string;
    theme: ThemeMode;
}

const minVer = 6;

const DEFAULT_PREFERENCES: ViewPreferences = {
    version: minVer,
    pageSizes: {},
    hideBannerContent: '',
    hideSidebar: false,
    position: '',
    theme: 'system'
};

export class ViewPreferencesService {
    private preferencesSubj: BehaviorSubject<ViewPreferences>;
    private systemThemeQuery?: MediaQueryList;

    constructor(private readonly storageKey = 'athena.member.preferences') {}

    public init() {
        if (!this.preferencesSubj) {
            const preferences = this.loadPreferences();
            this.applyTheme(preferences.theme);
            this.preferencesSubj = new BehaviorSubject(preferences);
            this.systemThemeQuery = window.matchMedia?.('(prefers-color-scheme: dark)');
            this.systemThemeQuery?.addEventListener('change', this.onSystemThemeChange);
            window.addEventListener('storage', event => {
                if (event.key !== null && event.key !== this.storageKey) {
                    return;
                }
                const nextPreferences = this.loadPreferences();
                this.applyTheme(nextPreferences.theme);
                this.preferencesSubj.next(nextPreferences);
            });
        }
    }

    public getPreferences(): Observable<ViewPreferences> {
        return this.preferencesSubj;
    }

    public updatePreferences(change: Partial<ViewPreferences>) {
        const nextPref = Object.assign({}, this.preferencesSubj.getValue(), change, {version: minVer});
        window.localStorage.setItem(this.storageKey, JSON.stringify(nextPref));
        this.applyTheme(nextPref.theme);
        this.preferencesSubj.next(nextPref);
    }

    public syncServerTheme(theme: ThemeMode) {
        if (this.preferencesSubj.getValue().theme === theme) {
            this.applyTheme(theme);
            return;
        }
        this.updatePreferences({theme});
    }

    public resolvedTheme(theme = this.preferencesSubj?.getValue().theme || DEFAULT_PREFERENCES.theme): ResolvedTheme {
        if (theme === 'system') {
            return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        }
        return theme;
    }

    private applyTheme(theme: ThemeMode) {
        document.documentElement.dataset.theme = this.resolvedTheme(theme);
    }

    private onSystemThemeChange = () => {
        if (this.preferencesSubj?.getValue().theme !== 'system') {
            return;
        }
        const current = this.preferencesSubj.getValue();
        this.applyTheme(current.theme);
        this.preferencesSubj.next({...current});
    };

    private loadPreferences(): ViewPreferences {
        let preferences: ViewPreferences;
        const preferencesStr = window.localStorage.getItem(this.storageKey);
        if (preferencesStr) {
            try {
                const parsed = JSON.parse(preferencesStr);
                preferences = parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : DEFAULT_PREFERENCES;
            } catch (e) {
                preferences = DEFAULT_PREFERENCES;
            }
            if (!preferences.version || preferences.version < minVer) {
                preferences = DEFAULT_PREFERENCES;
            }
        } else {
            preferences = DEFAULT_PREFERENCES;
        }
        const merged = deepMerge(DEFAULT_PREFERENCES, preferences);
        if (!['system', 'light', 'dark'].includes(merged.theme)) {
            merged.theme = 'system';
        }
        return merged;
    }
}
