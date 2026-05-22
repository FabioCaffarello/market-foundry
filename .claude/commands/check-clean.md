---
name: check-clean
description: Verify working tree clean + baseline (make verify, make bootstrap) PASS.
---

Verify the repository is in a clean state before starting any work.

## Steps

1. Check git working tree is clean:

   ```bash
   git status --short
   ```

   Output must be empty. If not, report what is modified or untracked.

2. Check the branch is up to date with origin:

   ```bash
   git fetch origin
   git rev-list --count origin/main..HEAD
   ```

   Must be `0` (no unpushed commits).

3. Run baseline validation:

   ```bash
   make bootstrap
   make verify
   ```

   Both must PASS.

If any check fails, **stop and report**. Do not proceed with other
work until the owner clarifies how to resolve the dirty state.

Use this command at the start of every significant session, especially
before applying scoped changes.
