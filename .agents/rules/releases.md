# Releases and docs

Refines `~/.claude/rule-details/versioning-a-project.md` and `~/.claude/rule-details/writing-docs.md`.

- **REL1** The git tag is the version. Before 1.0 a breaking change to the exported API, including one a regeneration brings in, bumps the minor version and says so in the CHANGELOG with the migration; everything else is a patch.
- **REL2** Every release prepends a `## vX.Y.Z, YYYY-MM-DD` section to `CHANGELOG.md`. The annotated tag message is `vX.Y.Z`, a blank line, then that section.
- **REL3** `.agents/.codex-plugin/plugin.json` carries `version` and moves to the release version in the same commit. `.agents/.claude-plugin/plugin.json` has no `version`.
- **REL4** The agent skill in `.agents/skills/go-deribit/` changes together with any generated method or type it names, and every example in it must compile against the code.
- **REL5** Commits, tags and pushes go through `git-update.sh` only. Tag pushes publish the skill to ClawHub through the `clawhub` job in `pipeline.yml`.
