# Overview

> [!WARNING]
> **As an Athena user, you probably don't want to be reading this section of the docs.**
>
> This part of the manual is aimed at helping people contribute to Athena, documentation, or to develop third-party applications that interact with Athena, e.g.
> 
> * A chat bot
> * A Slack integration

## Preface
#### Understand the [Code Contribution Preface](submit-your-pr.md#preface)
#### Follow the [Design-Led Backend Development Workflow](design-led-backend-development.md)
    
## Contributing to Athena documentation

This guide will help you get started quickly with contributing documentation changes, performing the minimum setup you'll need.   
For backend and frontend contributions, that require a full building-testing-running-locally cycle, please refer to [Contributing to Athena backend and frontend ](index.md#contributing-to-athena-backend-and-frontend) 

### Fork and clone Athena repository
- [Fork and clone Athena repository](development-environment.md#fork-and-clone)

### Submit your PR
- [Before submitting a PR](submit-your-pr.md#before-submitting-a-pr)
- [Choose a correct title for your PR](submit-your-pr.md#choose-a-correct-title-for-your-pr)
- [Perform the PR template checklist](submit-your-pr.md#perform-the-PR-template-checklist)

## Contributing to Athena Notifications documentation

This guide will help you get started quickly with contributing documentation changes, performing the minimum setup you'll need.
The notifications docs are located in [notifications-engine](https://github.com/useryege/notifications-engine) Git repository and require 2 pull requests: one for the `notifications-engine` repo and one for the `athena` repo.
For backend and frontend contributions, that require a full building-testing-running-locally cycle, please refer to [Contributing to Athena backend and frontend ](index.md#contributing-to-athena-backend-and-frontend) 

### Fork and clone Athena repository
- [Fork and clone Athena repository](development-environment.md#fork-and-clone-the-repository)

### Submit your PR to notifications-engine
- [Before submitting a PR](submit-your-pr.md#before-submitting-a-pr)
- [Choose a correct title for your PR](submit-your-pr.md#choose-a-correct-title-for-your-pr)
- [Perform the PR template checklist](submit-your-pr.md#perform-the-PR-template-checklist)

### Install Go on your machine
- [Install Go](development-environment.md#install-go)

### Submit your PR to athena
- [Contributing to notifications-engine](dependencies.md#notifications-engine-githubcomargoprojnotifications-engine)
- [Before submitting a PR](submit-your-pr.md#before-submitting-a-pr)
- [Choose a correct title for your PR](submit-your-pr.md#choose-a-correct-title-for-your-pr)
- [Perform the PR template checklist](submit-your-pr.md#perform-the-PR-template-checklist)

## Contributing to Athena backend and frontend 

This guide will help you set up your local development environment so you can run Athena, update generated files, and prepare production deployments.

As is the case with the development process, this document is under constant change. If you notice any error, or if you think this document is out-of-date, or if you think it is missing something: Feel free to submit a PR or submit a bug to our GitHub issue tracker.

### Set up your development environment
- [Install required tools (Git, Go, Docker, etc)](development-environment.md#install-required-tools)
- [Fork and clone Athena repository](development-environment.md#fork-and-clone-the-repository)
- [Install development tools](development-environment.md#install-development-tools)
- [Start local services](development-environment.md#local-services)

### Set up a development toolchain
- [Set up the local toolchain](toolchain-guide.md#local-toolchain)

### Perform the development cycle 
- [Start with the compact local development workflow](how-to-develop.md)
- [Follow the design-led backend workflow and phase gates](design-led-backend-development.md)
- How to contribute to documentation: maintain repository Markdown directly and follow the [requirements](../requirements/README.md), [backend technical design](../design/README.md), and [design-led workflow](design-led-backend-development.md) rules

### Run and debug Athena locally
- [Run Athena on your machine for manual testing](running-locally.md)
- [Debug Athena in an IDE on your machine](debugging-locally.md)
  
### Submit your PR
- [Before submitting a PR](submit-your-pr.md#before-submitting-a-pr)
- [Understand the Continuous Integration process](submit-your-pr.md#understand-the-continuous-integration-process)
- [Choose a correct title for your PR](submit-your-pr.md#choose-a-correct-title-for-your-pr)
- [Perform the PR template checklist](submit-your-pr.md#perform-the-PR-template-checklist)
- [Understand the CI automated builds & tests](submit-your-pr.md#automated-builds-&-tests)
- [Understand & make sure your PR meets the CI code test coverage requirements](submit-your-pr.md#code-test-coverage)

## Contributing to Athena dependencies
- [Contributing to athena-ui](dependencies.md#athena-ui-components-githubcomargoprojargo-ui)
- [Contributing to notifications-engine](dependencies.md#notifications-engine-githubcomargoprojnotifications-engine)
