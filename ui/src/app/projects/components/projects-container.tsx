import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {ProjectDetails} from './project-details/project-details';
import {ProjectsList} from './projects-list/projects-list';

export const ProjectsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={ProjectsList} />
        <Redirect exact={true} from={`${props.match.path}/archived`} to={`${props.match.path}`} />
        <Redirect exact={true} from={`${props.match.path}/archived/:contract`} to={`${props.match.path}/:contract`} />
        <Route exact={true} path={`${props.match.path}/:contract`} component={ProjectDetails} />
    </Switch>
);
