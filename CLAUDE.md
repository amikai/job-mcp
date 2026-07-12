# Commit conventions

- Follow [Conventional Commits](https://www.conventionalcommits.org/): `<type>: <subject>` with types `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. Subject in imperative mood, lowercase, no trailing period. Body states the motivation, not a restatement of the diff.
- Exception — roster updates use the `roster:` prefix instead: keep each PR to a single `companies.yaml` roster; batch all of a session's additions into one commit (squash if already split), titled `roster: add N companies to <Provider>` (first population) or `roster: enrich <Provider> companies` (existing).
