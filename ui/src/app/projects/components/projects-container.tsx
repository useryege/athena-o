import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {ProjectsList} from './projects-list/projects-list';

export const ProjectsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ProjectsList} />
    </Switch>
);
