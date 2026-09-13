<!--
Note on DCO:

If the DCO action in the integration test fails, one or more of your commits are not signed off. Please click on the *Details* link next to the DCO action for instructions on how to resolve this.
-->

Checklist:

* [ ] Either (a) I've created an [enhancement proposal](https://github.com/useryege/athena/issues/new/choose) and discussed it with the community, (b) this is a bug fix, or (c) this does not need to be in the release notes.
* [ ] The title of the PR states what changed and the related issue number (used for release notes when applicable).
* [ ] The title of the PR conforms to the repository convention: `feat|fix|docs|test|ci|chore: ...`
* [ ] I've included "Closes [ISSUE #]" or "Fixes [ISSUE #]" in the description to automatically close the associated issue.
* [ ] 按实际影响完成并验证了所需的 CLI、UI 与其他消费者；没有受影响消费者时已说明不适用。
* [ ] Does this PR require documentation updates?
* [ ] I've updated documentation as required by this PR.
* [ ] I have signed off all my commits as required by [DCO](https://developercertificate.org/).
* [ ] 我已按实际影响完成验证（文档改动可使用链接与静态检查；行为改动包含必要的测试或其他行为证据）。
* [ ] My build is green.
* [ ] My new feature complies with the current Athena project scope and maintenance expectations.
* [ ] 服务或服务边界有变更：我已引用适用的[服务开发规范 SDS-R1 至 SDS-R8](../docs/developer-guide/service-development-standards.md)，并在设计或 PR 描述中记录所有者、独立进程/部署、入口、最小依赖、授权、事务、RPC deadline、故障影响和验证证据；不适用时已说明原因。
* [ ] I have added a brief description of why this PR is necessary and/or what this PR solves.
* [ ] Optional. For bug fixes, I've indicated what older releases this fix should be cherry-picked into (this may or may not happen depending on risk/complexity).

<!-- Please see the Athena contributor docs and FAQ if you have questions about your pull request. -->
