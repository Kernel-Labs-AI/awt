# Starting a new task

- When you begin working on a new task, use `awt task start --title "xxx" --agent "your_name"` with an appropriate title and your name.
- First thing first run `awt task copy <task_id> .env` to make sure env variables are copied over.
- When you need to commit changes, use `awt task commit <task_id> -m 'Message'`, do not navigate into the cwd and use git manually.
- When the user asks you to handoff the task, commit changes then use the `awt task handoff <task_id>` command. 
- If the user needs you to run a command, always do `awt task exec <task_id> -- my command`
- Always give the user your task ID so they can also run commands. 

If you need help with the commands, just look at the source in ./internal.

# Documentation

When you create a new command, always update ./USAGE.md to include instructions on how to use the new command.

# Dependencies selection

When choosing what dependency to use to get something done, make sure to always look at 2-3 options. Always prefer projects that have recent updates. If a project hasn't been updated in >2 years, we cannot use it.

# Quality control

ALWAYS run tests before telling me you are done with a change. Look at .github/workflows/ci.yml to understand what needs to pass in CI.