import * as React from 'react';
import {Link, useLocation, useNavigate, useNavigationType} from 'react-router-dom';

const PROJECTS_PATH = '/token/projects';
const PROJECTS_RETURN_STORAGE_PREFIX = 'athena.member.token-projects.return.';
const PROJECTS_RETURN_HISTORY_FIELD = 'athenaProjectsReturnID';

interface ProjectsReturnSnapshot {
    projectsURL: string;
    locationKey: string;
    returnID: string;
    scrollY: number;
}

interface ProjectDetailLocationState {
    projectsReturn?: ProjectsReturnSnapshot;
}

const projectsReturnStorageKey = (locationKey: string) => `${PROJECTS_RETURN_STORAGE_PREFIX}${locationKey}`;

const isProjectsURL = (value: string) => {
    try {
        const base = new URL('https://athena.invalid');
        const parsed = new URL(value, base);
        return parsed.origin === base.origin && parsed.pathname === PROJECTS_PATH && `${parsed.pathname}${parsed.search}` === value;
    } catch {
        return false;
    }
};

const parseProjectsReturnSnapshot = (value: unknown): ProjectsReturnSnapshot | undefined => {
    if (!value || typeof value !== 'object') {
        return undefined;
    }
    const candidate = value as Partial<ProjectsReturnSnapshot>;
    if (
        typeof candidate.projectsURL !== 'string' ||
        !isProjectsURL(candidate.projectsURL) ||
        typeof candidate.locationKey !== 'string' ||
        !candidate.locationKey ||
        typeof candidate.returnID !== 'string' ||
        !candidate.returnID ||
        typeof candidate.scrollY !== 'number' ||
        !Number.isFinite(candidate.scrollY) ||
        candidate.scrollY < 0
    ) {
        return undefined;
    }
    return {
        projectsURL: candidate.projectsURL,
        locationKey: candidate.locationKey,
        returnID: candidate.returnID,
        scrollY: candidate.scrollY
    };
};

const writeProjectsReturnSnapshot = (snapshot: ProjectsReturnSnapshot) => {
    try {
        window.sessionStorage.setItem(projectsReturnStorageKey(snapshot.locationKey), JSON.stringify(snapshot));
    } catch {
        // Detail navigation remains usable when storage is unavailable.
    }
};

const readProjectsReturnSnapshot = (locationKey: string) => {
    try {
        const stored = window.sessionStorage.getItem(projectsReturnStorageKey(locationKey));
        return stored ? parseProjectsReturnSnapshot(JSON.parse(stored)) : undefined;
    } catch {
        return undefined;
    }
};

const consumeProjectsReturnSnapshot = (locationKey: string) => {
    try {
        window.sessionStorage.removeItem(projectsReturnStorageKey(locationKey));
    } catch {
        // A stale snapshot is harmless when storage is unavailable.
    }
};

const projectsReturnFromLocationState = (state: unknown) => {
    if (!state || typeof state !== 'object') {
        return undefined;
    }
    return parseProjectsReturnSnapshot((state as ProjectDetailLocationState).projectsReturn);
};

const createProjectsReturnID = () => `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;

const historyLocationKey = (historyState: Record<string, unknown>) => (typeof historyState.key === 'string' ? historyState.key : 'default');

const markProjectsHistoryEntry = (locationKey: string, returnID: string) => {
    try {
        const historyState = window.history.state;
        if (!historyState || typeof historyState !== 'object' || historyLocationKey(historyState) !== locationKey) {
            return;
        }
        window.history.replaceState({...historyState, key: locationKey, [PROJECTS_RETURN_HISTORY_FIELD]: returnID}, document.title);
    } catch {
        // URL return remains available when the browser history state cannot be annotated.
    }
};

const currentProjectsReturnID = (locationKey: string) => {
    const historyState = window.history.state;
    if (!historyState || typeof historyState !== 'object' || historyLocationKey(historyState) !== locationKey) {
        return undefined;
    }
    const value = historyState[PROJECTS_RETURN_HISTORY_FIELD];
    return typeof value === 'string' && value ? value : undefined;
};

const isUnmodifiedPrimaryClick = (event: React.MouseEvent<HTMLAnchorElement>) =>
    event.button === 0 &&
    !event.defaultPrevented &&
    !event.metaKey &&
    !event.ctrlKey &&
    !event.shiftKey &&
    !event.altKey &&
    (!event.currentTarget.target || event.currentTarget.target === '_self');

export const clearProjectsReturnSnapshots = () => {
    try {
        for (let index = window.sessionStorage.length - 1; index >= 0; index -= 1) {
            const key = window.sessionStorage.key(index);
            if (key?.startsWith(PROJECTS_RETURN_STORAGE_PREFIX)) {
                window.sessionStorage.removeItem(key);
            }
        }
    } catch {
        // Authentication cleanup must not fail when storage is unavailable.
    }
};

export const ProjectDetailLink = (props: {projectID: number; children?: React.ReactNode}) => {
    const location = useLocation();
    const navigate = useNavigate();
    const destination = `${PROJECTS_PATH}/${props.projectID}`;

    const navigateToProject = (event: React.MouseEvent<HTMLAnchorElement>) => {
        if (!isUnmodifiedPrimaryClick(event)) {
            return;
        }
        event.preventDefault();

        const projectsURL = `${location.pathname}${location.search}`;
        const returnID = createProjectsReturnID();
        const snapshot = parseProjectsReturnSnapshot({
            projectsURL,
            locationKey: location.key,
            returnID,
            scrollY: Math.max(0, window.scrollY)
        });
        if (!snapshot) {
            navigate(destination);
            return;
        }

        markProjectsHistoryEntry(location.key, returnID);
        writeProjectsReturnSnapshot(snapshot);
        navigate(destination, {state: {projectsReturn: snapshot} satisfies ProjectDetailLocationState});
    };

    return (
        <Link to={destination} onClick={navigateToProject}>
            {props.children || 'View details'}
        </Link>
    );
};

export const useProjectDetailReturn = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const returnSnapshot = projectsReturnFromLocationState(location.state);

    return React.useCallback(() => {
        if (returnSnapshot) {
            navigate(-1);
            return;
        }
        navigate(PROJECTS_PATH, {replace: true});
    }, [navigate, returnSnapshot]);
};

export const useScrollProjectDetailOnPush = () => {
    const navigationType = useNavigationType();

    React.useLayoutEffect(() => {
        if (navigationType === 'PUSH') {
            window.scrollTo({top: 0, left: 0, behavior: 'auto'});
        }
    }, [navigationType]);
};

export const useRestoreProjectsScroll = (dataReady: boolean) => {
    const location = useLocation();
    const navigationType = useNavigationType();

    React.useLayoutEffect(() => {
        if (!dataReady || navigationType !== 'POP') {
            return undefined;
        }

        const snapshot = readProjectsReturnSnapshot(location.key);
        const projectsURL = `${location.pathname}${location.search}`;
        const returnID = currentProjectsReturnID(location.key);
        if (!snapshot || !returnID || snapshot.returnID !== returnID || snapshot.locationKey !== location.key || snapshot.projectsURL !== projectsURL) {
            return undefined;
        }

        let animationFrame = 0;
        let attempts = 0;
        let cancelled = false;
        const restore = () => {
            animationFrame = window.requestAnimationFrame(() => {
                if (cancelled) {
                    return;
                }
                attempts += 1;
                window.scrollTo({top: snapshot.scrollY, left: 0, behavior: 'auto'});
                if (Math.abs(window.scrollY - snapshot.scrollY) <= 1 || attempts >= 3) {
                    consumeProjectsReturnSnapshot(location.key);
                    return;
                }
                restore();
            });
        };
        restore();

        return () => {
            cancelled = true;
            window.cancelAnimationFrame(animationFrame);
        };
    }, [dataReady, location.key, location.pathname, location.search, navigationType]);
};
