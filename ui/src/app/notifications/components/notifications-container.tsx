import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {NotificationDetails} from './notification-details';
import {NotificationsList} from './notifications-list';

require('./notifications.scss');

export const NotificationsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={NotificationsList} />
        <Route exact={true} path={`${props.match.path}/:id`} component={NotificationDetails} />
        <Redirect to={props.match.path} />
    </Switch>
);
