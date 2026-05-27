import * as React from 'react';
import {FormField} from 'argo-ui';
import {FormApi} from 'react-form';
import {MapInputField} from '../../../shared/components';

export const EditAnnotations = (props: {formApi: FormApi}) => {
    return <FormField formApi={props.formApi} field='metadata.annotations' component={MapInputField} />;
};
