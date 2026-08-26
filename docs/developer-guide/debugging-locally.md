# Debugging a local Athena instance

## Prerequisites
1. [Development Environment](development-environment.md)   
2. [Toolchain Guide](toolchain-guide.md)
3. [Development Cycle](development-cycle.md)
4. [Running Locally](running-locally.md)

## Preface
Please make sure you are familiar with running Athena locally using [make run](running-locally.md#start-local-services).

When running Athena locally for manual tests, the quickest way to do so is to run all the Athena components together, as described in [Running Locally](running-locally.md), 

However, when you need to debug a single Athena component (for example, `api-server`, `repo-server`, etc), you will need to run this component separately in your IDE, using your IDE launch and debug configuration, while the other components will be running as described previously, using the local toolchain.

For the next steps, we will use Athena `api-server` as an example of running a component in an IDE.

## Configure your IDE

### Locate your component configuration in `Procfile`
The `Procfile` is used by Goreman when running Athena locally with the local toolchain. The file is located in the top-level directory in your cloned Athena repo folder, you can view it's latest version [here](https://github.com/useryege/athena/blob/master/Procfile). It contains all the needed component run configuration, and you will need to copy parts of this configuration to your IDE. 

Example for `api-server` configuration in `Procfile`:
``` text
api-server: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/api-server} FORCE_LOG_COLORS=1 ATHENA_SSH_DATA_PATH=${ATHENA_SSH_DATA_PATH:-/tmp/athena-local/ssh} ATHENA_GOOGLE_OIDC_REDIRECT_URI=${ATHENA_GOOGLE_OIDC_REDIRECT_URI:-http://localhost:4000/auth/google/callback} ATHENA_BINARY_NAME=athena-server go run ./cmd/main.go --redis localhost:${ATHENA_REDIS_PORT:-6379} --disable-auth=${ATHENA_SERVER_DISABLE_AUTH:-'false'} --address ${ATHENA_SERVER_LISTEN_ADDRESS:-127.0.0.1} --port ${ATHENA_SERVER_PORT:-8080}"
```
This configuration example will be used as the basis for the next steps.

> [!NOTE]
> The Procfile for a component may change with time. Please go through the Procfile and make sure you use the latest configuration for debugging.

### Configure component env variables
The component that you will run in your IDE for debugging (`api-server` in our case) will need env variables. Copy the env variables from `Procfile`, located in the `athena` root folder of your development branch. The env variables are located before `go run ./cmd/main.go` in the `sh -c` section of the component run command.
You can keep them in `.env` file and then have the IDE launch configuration point to that file. Obviously, you can adjust the env variables to your needs when debugging a specific configuration.

Example for an `api-server.env` file:
``` bash
ATHENA_BINARY_NAME=athena-server
ATHENA_GNUPGHOME=/tmp/athena-local/gpg/keys
ATHENA_GPG_DATA_PATH=/tmp/athena-local/gpg/source
ATHENA_GPG_ENABLED=false
ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP=1
ATHENA_SSH_DATA_PATH=/tmp/athena-local/ssh
ATHENA_TRACING_ENABLED=1
FORCE_LOG_COLORS=1
ATHENA_GOOGLE_OIDC_CLIENT_ID=<local-web-client-id>
ATHENA_GOOGLE_OIDC_CLIENT_SECRET=<local-web-client-secret>
ATHENA_GOOGLE_OIDC_REDIRECT_URI=http://localhost:4000/auth/google/callback
ATHENA_ADMIN_GOOGLE_EMAIL=<administrator-google-email>
ATHENA_JWT_SECRET=<at-least-32-byte-signing-secret>
... 
# and so on for the component-specific settings you are testing.
```

### Install DotENV / EnvFile plugin
Using the market place / plugin manager of your IDE. The below example configurations require the plugin to be installed.


### Configure component IDE launch configuration
#### VSCode example
Next, you will need to create a launch configuration, with the relevant args. Copy the args from `Procfile`, located in the `athena` root folder of your development branch. The args are located after `go run ./cmd/main.go` in the `sh -c` section of the component run command.
Example for an `api-server` launch configuration, based on our above example for `api-server` configuration in `Procfile`: 
``` json
    {
      "name": "api-server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "YOUR_CLONED_ARGO_CD_REPO_PATH/athena/cmd",
      "args": [
        "--loglevel",
        "debug",
        "--redis",
        "localhost:6379",
        "--port",
        "8080"
      ],
      "envFile": "YOUR_ENV_FILES_PATH/api-server.env", # Assuming you installed DotENV plugin
    }
```

#### Goland example
Next, you will need to create a launch configuration, with the relevant parameters. Copy the parameters from `Procfile`, located in the `athena` root folder of your development branch. The parameters are located after `go run ./cmd/main.go` in the `sh -c` section of the component run command.
Example for an `api-server` launch configuration snippet, based on our above example for `api-server` configuration in `Procfile`: 
``` xml 
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="api-server" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="athena" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="--loglevel debug --redis localhost:6379 --port 8080" />
    <EXTENSION ID="net.ashald.envfile"> <!-- Assuming you installed the EnvFile plugin-->
      <option name="IS_ENABLED" value="true" />
      <option name="IS_SUBST" value="false" />
      <option name="IS_PATH_MACRO_SUPPORTED" value="false" />
      <option name="IS_IGNORE_MISSING_FILES" value="false" />
      <option name="IS_ENABLE_EXPERIMENTAL_INTEGRATIONS" value="false" />
      <ENTRIES>
        <ENTRY IS_ENABLED="true" PARSER="runconfig" IS_EXECUTABLE="false" />
        <ENTRY IS_ENABLED="true" PARSER="env" IS_EXECUTABLE="false" PATH="<YOUR_ENV_FILES_PATH>/api-server.env" />
      </ENTRIES>
    </EXTENSION>
    <kind value="DIRECTORY" />
    <directory value="$PROJECT_DIR$/cmd" />
    <filePath value="$PROJECT_DIR$" />
    <method v="2" />
  </configuration>
</component>
```

> [!NOTE]
> As an alternative to importing the above file to Goland, you can create a Run/Debug Configuration using the official [Goland docs](https://www.jetbrains.com/help/go/go-build.html) and just copy the `parameters`, `directory` and `PATH` sections from the example above (specifying `Run kind` as `Directory` in the Run/Debug Configurations wizard)

## Run Athena without the debugged component
Next, we need to run all Athena components, except for the debugged component (cause we will run this component separately in the IDE).
Run the other components locally, then launch the debugged component from your IDE.

### Run the other components locally
#### Run with "make run"
`make run` runs all the components by default, but it is also possible to run it with a blacklist of components, enabling the separation we need.

So for the case of debugging the `api-server`, run:
`make run exclude=api-server` 

#### Run with "goreman start"
`goreman start` runs all the components by default, but it is also possible to run it with a whitelist of components, enabling separation as needed.

To debug the `api-server`, run:
`goreman start notification applicationset-controller redis controller ui` 

## Run Athena debugged component from your IDE
Finally, run the component you wish to debug from your IDE and make sure it does not have any errors.

## Important
When running Athena components separately, ensure components aren't creating conflicts - each component needs to be up exactly once, be it running locally with the local toolchain or running from your IDE. Otherwise you may get errors about ports not available or even debugging a process that does not contain your code changes. 
