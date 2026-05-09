import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {ProjectDetails} from './project-details/project-details';
import {ProjectsList} from './projects-list/projects-list';

export const ProjectsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ProjectsList} />
        <Route exact={true} path={`${props.match.path}/:projectID`} component={ProjectDetails} />
    </Switch>
);
