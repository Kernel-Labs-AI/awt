# Task Workflow

When you are working on a new task, follow this workflow closely.

## Starting a new task in awt

- When you begin working on a new task, use `awt task start --title "xxx" --agent "your_name"` with an appropriate title and your name. It will create a git worktree for you to work in.
- If there's an .env file in the root, use `awt task copy <task_id> .env` to make sure env variables are copied over.
- When you need to commit changes, use `awt task commit <task_id> -m 'Message'`, do not navigate into the cwd and use git manually.
- When the user asks you to handoff the task, commit changes then use the `awt task handoff <task_id>` command. This will push it to Github and open a PR.
- If the user needs you to run a command, always do `awt task exec <task_id> -- my command`
- At the end of each message, give the user your task ID so they can also run commands. 

If you need help with the commands, use `awt --help` or just look at the source in ./internal.

## Understand API surface

If you're implementing a new command, read USAGE.md before starting to work on this. We should keep the number of commands small. If we have the opportunity to combine some commands, we should. For example we have `commit` and `handoff` for pushing to GH. We should not add a `push` command, as it would be redundant. 

# Documentation

When you create a new command, always update ./USAGE.md to include instructions on how to use the new command. Make sure to document all arguments and flags for the command.

# Testing

ALWAYS run tests and the linter before telling me you are done with a change. Look at .github/workflows/ci.yml to understand what needs to pass in CI.