# Athena UI

Web UI for Athena.


## Getting started

Install NVM and Yarn before continuing. For WSL, see the
[Athena installation guide](../docs/developer-guide/install-wsl.md). Other environments can
use the upstream [NVM installation instructions](https://github.com/nvm-sh/nvm#installing-and-updating)
and [Yarn installation instructions](https://classic.yarnpkg.com/en/docs/install).

From the repository root, install dependencies with the project Node.js version:

```bash
cd ui
nvm install
nvm use
yarn install
```

`.nvmrc` currently selects Node.js `24.14.1`; `package.json` accepts
`>=24.14.1 <25`. Do not change your global NVM default. Run `nvm use` from `ui`
in each shell that runs UI commands or starts the local stack. Docker UI builds also use
Node.js `24.14.1`.

Run `yarn start` to launch the Vite dev UI server. The member application is available at
`/` and the administrator application at `/admin/`. Run `yarn build` to bundle static
resources into the `./dist` directory.

To build a Docker image, run `IMAGE_NAMESPACE=yourimagerepo IMAGE_TAG=latest yarn docker`.

To do the same and push to a Docker registry, run `IMAGE_NAMESPACE=yourimagerepo IMAGE_TAG=latest DOCKER_PUSH=true yarn docker`.

## Pre-commit Checks

Make sure your code passes the lint checks:

```
yarn lint --fix
```

If you are using VSCode, add this configuration to `.vscode/settings.json` in the root of this repository to identify and fix lint issues automatically before you save file.

Install [Eslint Extension](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) in VSCode.

`.vscode/settings.json`
```json
{
  "eslint.format.enable": true,
    "editor.codeActionsOnSave": {
        "source.fixAll.eslint": "always"
    },
    "eslint.workingDirectories": [
        {
            "directory": "./ui",
            "!cwd": false
        }
    ],
    "eslint.experimental.useFlatConfig": true
}
```
