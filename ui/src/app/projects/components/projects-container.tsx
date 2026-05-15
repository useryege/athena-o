import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {ArchivedProjectDetails} from './archived-project-details/archived-project-details';
import {ArchivedProjectsList} from './archived-projects-list/archived-projects-list';
import {ProjectDetails} from './project-details/project-details';
import {ProjectsList} from './projects-list/projects-list';

export const ProjectsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ProjectsList} />
        <Route exact={true} path={`${props.match.path}/archived`} component={ArchivedProjectsList} />
        <Route exact={true} path={`${props.match.path}/archived/:projectID`} component={ArchivedProjectDetails} />
        <Route exact={true} path={`${props.match.path}/:projectID`} component={ProjectDetails} />
    </Switch>
);
