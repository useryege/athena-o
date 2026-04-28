import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {AccountDetails} from './account-details/account-details';
import {AccountsList} from './accounts-list/accounts-list';
import {EvmNodeWsURL} from './evm-node-ws-url/evm-node-ws-url';
import {SettingsOverview} from './settings-overview/settings-overview';
import {AppearanceList} from './appearance-list/appearance-list';

export const SettingsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={SettingsOverview} />
        {/* <Route exact={true} path={`${props.match.path}/repos`} component={ReposList} />
        <Route exact={true} path={`${props.match.path}/certs`} component={CertsList} />
        <Route exact={true} path={`${props.match.path}/gpgkeys`} component={GpgKeysList} />
        <Route exact={true} path={`${props.match.path}/clusters`} component={ClustersList} />
        <Route exact={true} path={`${props.match.path}/clusters/:server`} component={ClusterDetails} />
        <Route exact={true} path={`${props.match.path}/projects`} component={ProjectsList} />
        <Route exact={true} path={`${props.match.path}/projects/:name`} component={ProjectDetails} /> */}
        <Route exact={true} path={`${props.match.path}/accounts`} component={AccountsList} />
        <Route exact={true} path={`${props.match.path}/accounts/:name`} component={AccountDetails} />
        <Route exact={true} path={`${props.match.path}/appearance`} component={AppearanceList} />
        <Route exact={true} path={`${props.match.path}/evm-node-ws-url`} component={EvmNodeWsURL} />
        <Redirect path='*' to={`${props.match.path}`} />
    </Switch>
);
