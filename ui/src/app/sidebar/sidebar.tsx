import {Tooltip} from 'argo-ui';
import {Boundary, Placement} from 'popper.js';
import {useData} from 'argo-ui/v2';
import * as React from 'react';
import {Context} from '../shared/context';
import {services, ViewPreferences} from '../shared/services';

require('./sidebar.scss');

export interface SidebarNavItem {
    path?: string;
    iconClassName?: string;
    title: string;
    tooltip?: string;
    children?: SidebarNavItem[];
}

interface SidebarProps {
    onVersionClick: () => void;
    navItems: SidebarNavItem[];
    pref: ViewPreferences;
}

export const SIDEBAR_TOOLS_ID = 'sidebar-tools';

export const useSidebarTarget = () => {
    const sidebarTarget = React.useRef(document.createElement('div'));

    React.useEffect(() => {
        const sidebar = document.getElementById(SIDEBAR_TOOLS_ID);
        sidebar.appendChild(sidebarTarget?.current);
        return () => {
            sidebarTarget.current?.remove();
        };
    }, []);

    return sidebarTarget;
};

export const Sidebar = (props: SidebarProps) => {
    const context = React.useContext(Context);
    const [version, loading, error] = useData(() => services.version.version());
    const locationPath = context.history.location.pathname;

    const tooltipProps = {
        placement: 'right' as Placement,
        popperOptions: {
            modifiers: {
                preventOverflow: {
                    boundariesElement: 'window' as Boundary
                }
            }
        }
    };

    const isActive = (item: SidebarNavItem): boolean => {
        if (item.path && (locationPath === item.path || locationPath.startsWith(`${item.path}/`))) {
            return true;
        }
        return (item.children || []).some(child => isActive(child));
    };

    const renderNavItem = (item: SidebarNavItem, child = false) => {
        if (child && props.pref.hideSidebar && !item.iconClassName) {
            return null;
        }

        const active = isActive(item);
        const isGroup = !child && (item.children || []).length > 0;
        const className = [
            'sidebar__nav-item',
            child ? 'sidebar__nav-item--child' : '',
            child && !item.iconClassName ? 'sidebar__nav-item--child-text' : '',
            isGroup ? 'sidebar__nav-item--group' : '',
            active ? 'sidebar__nav-item--active' : ''
        ]
            .filter(Boolean)
            .join(' ');
        const onClick = item.path ? () => context.history.push(item.path) : undefined;

        return (
            <Tooltip key={item.path || item.title} content={<div className='sidebar__tooltip'>{item?.tooltip || item.title}</div>} {...tooltipProps}>
                <div className={className} onClick={onClick}>
                    <div>
                        {item.iconClassName && <i className={item.iconClassName} />}
                        {!props.pref.hideSidebar && <span className='sidebar__nav-item-title'>{item.title}</span>}
                    </div>
                </div>
            </Tooltip>
        );
    };

    return (
        <div className={`sidebar ${props.pref.hideSidebar ? 'sidebar--collapsed' : ''}`}>
            <div className='sidebar__container'>
                <div className='sidebar__logo'>
                    <div onClick={() => services.viewPreferences.updatePreferences({...props.pref, hideSidebar: !props.pref.hideSidebar})} className='sidebar__collapse-button'>
                        <i className={`fas fa-arrow-${props.pref.hideSidebar ? 'right' : 'left'}`} />
                    </div>
                    {!props.pref.hideSidebar && (
                        <div className='sidebar__logo-container'>
                            <img
                                onClick={() => context.history.push('/')}
                                title={'Go to start page'}
                                src='assets/images/athenalogo.svg'
                                alt='Argo'
                                className='sidebar__logo__text-logo'
                            />
                            <div className='sidebar__version' onClick={props.onVersionClick}>
                                {loading ? 'Loading...' : error?.state ? 'Unknown' : version?.Version || 'Unknown'}
                            </div>
                        </div>
                    )}
                    <img onClick={() => context.history.push('/')} title={'Go to start page'} src='assets/images/logo.png' alt='Argo' className='sidebar__logo__character' />{' '}
                </div>

                {(props.navItems || []).map(item => (
                    <React.Fragment key={item.path || item.title}>
                        {renderNavItem(item)}
                        {(item.children || []).length > 0 && <div className='sidebar__nav-children'>{item.children.map(child => renderNavItem(child, true))}</div>}
                    </React.Fragment>
                ))}

                {props.pref.hideSidebar && (
                    <Tooltip content='Show Filters' {...tooltipProps}>
                        <div
                            onClick={() => services.viewPreferences.updatePreferences({...props.pref, hideSidebar: !props.pref.hideSidebar})}
                            className='sidebar__nav-item sidebar__filter-button'>
                            <div>
                                <i className={`fas fa-filter`} />
                            </div>
                        </div>
                    </Tooltip>
                )}
            </div>
            <div id={SIDEBAR_TOOLS_ID} />
        </div>
    );
};
