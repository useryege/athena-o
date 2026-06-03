import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {AccountDetails} from './account-details/account-details';
import {AccountsList} from './accounts-list/accounts-list';
import {ApplicationDiscovery} from './application-discovery/application-discovery';
import {SettingsOverview} from './settings-overview/settings-overview';
import {AppearanceList} from './appearance-list/appearance-list';

export const SettingsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={SettingsOverview} />
        <Route exact={true} path={`${props.match.path}/accounts`} component={AccountsList} />
        <Route exact={true} path={`${props.match.path}/accounts/:name`} component={AccountDetails} />
        <Route exact={true} path={`${props.match.path}/appearance`} component={AppearanceList} />
        <Route exact={true} path={`${props.match.path}/application-discovery`} component={ApplicationDiscovery} />
        <Redirect path='*' to={`${props.match.path}`} />
    </Switch>
);
