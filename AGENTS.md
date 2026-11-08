# AGENTS.md

Guidance for coding agents working in this repository. The project layout, commands and release process are in [CLAUDE.md](CLAUDE.md).

## This repository is public

Everything committed is public: code, tests, docs, commit messages, and pull request text and comments. Several tools here work on personal data, such as photos, locations and DNS, so check every change for personal data before committing or opening a pull request, and leave it out:

- Names, email addresses, phone numbers and addresses of real people. The exception is the owner's git identity, already present in the history.
- Data from the owner's own accounts and tools: real locations, coordinates, travel history, photos, file names, camera details, people's names or IDs from Immich, Google or similar.
- Details of the owner's setup: hostnames, IP addresses, server URLs, ports, account IDs, API keys and tokens.
- Local paths that identify the machine or user, such as home directories.
- Files the tools produce. Catalogs, `immich.json`, `config.json`, `api-key` and the like belong in each tool's data folder outside the repo.

Use placeholders instead: `example.org` or `example.com` for hosts, a generic private IP such as `192.168.1.20`, made-up names such as "Me" and "Friend", and the centre of a well-known city for coordinates.

Stage files by name, not with `git add -A` or `git add .`: the working tree often holds untracked files that are not part of the change. When unsure whether something is personal, leave it out and ask.
