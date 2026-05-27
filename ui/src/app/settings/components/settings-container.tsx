import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {AccountDetails} from './account-details/account-details';
import {AccountsList} from './accounts-list/accounts-list';
import {SettingsOverview} from './settings-overview/settings-overview';
import {AppearanceList} from './appearance-list/appearance-list';

export const SettingsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={SettingsOverview} />
        <Route exact={true} path={`${props.match.path}/accounts`} component={AccountsList} />
        <Route exact={true} path={`${props.match.path}/accounts/:name`} component={AccountDetails} />
        <Route exact={true} path={`${props.match.path}/appearance`} component={AppearanceList} />
        <Redirect path='*' to={`${props.match.path}`} />
    </Switch>
);
