# Managing Dependencies

## Notifications Engine (`github.com/useryege/notifications-engine`)

### Repository

[notifications-engine](https://github.com/useryege/notifications-engine)

### Pulling changes from `notifications-engine`

After your Notifications Engine PR has been merged, Athena needs to be updated to pull in the version of the notifications engine that contains your change. Here are the steps:

- Retrieve the SHA hash for your commit. You will use this in the next step.
- From the `athena` folder, run the following command

  `go get github.com/useryege/notifications-engine@<git-commit-sha>`

  If you get an error message `invalid version: unknown revision` then you got the wrong SHA hash

- Run:

  `go mod tidy`

- The following files are changed:

  - `go.mod`
  - `go.sum`

- If your notifications engine PR included docs changes, run `make codegen` or `make codegen-local`.

- Create an Athena PR with a `refactor:` type in its title for the above file changes.

## Athena UI Components (`github.com/useryege/argo-ui`)
### Contributing to Athena UI

Athena, along with Argo Workflows, uses shared React components from [Athena UI](https://github.com/useryege/argo-ui). Examples of some of these components include buttons, containers, form controls, 
and others. Although you can make changes to these files and run them locally, in order to have these changes added to the Athena repo, you will need to follow these steps. 

1. Fork and clone the [Athena UI repository](https://github.com/useryege/argo-ui).

2. `cd` into your `argo-ui` directory, and then run `yarn install`. 

3. Make your file changes.

4. Run `yarn start` to start a [storybook](https://storybook.js.org/) dev server and view the components in your browser. Make sure all your changes work as expected. 

5. Use [yarn link](https://classic.yarnpkg.com/en/docs/cli/link/) to link Athena UI package to your Athena repository. (Commands below assume that `argo-ui` and `athena` are both located within the same parent folder)

    * `cd argo-ui`
    * `yarn link`
    * `cd ../athena/ui`
    * `yarn link argo-ui`

    Once the `argo-ui` package has been successfully linked, test changes in your local development environment. 

6. Commit changes and open a PR to [Athena UI](https://github.com/useryege/argo-ui). 

7. Once your PR has been merged in Athena UI, `cd` into your `athena/ui` folder and run `yarn add git+https://github.com/useryege/argo-ui.git`. This will update the commit SHA in the `ui/yarn.lock` file to use the latest master commit for argo-ui. 

8. Submit changes to `ui/yarn.lock`in a PR to Athena. 
