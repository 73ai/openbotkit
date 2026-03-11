# Schedule Task Reference

## Overview

Scheduled tasks run automatically at specified times. The system supports two types:

- **Recurring**: Runs on a cron schedule (e.g., daily at 9am)
- **One-shot**: Runs once at a specific time (e.g., in 2 hours)

## Creating Schedules

### Recurring Tasks

Use `create_schedule` with:
- `type`: `"recurring"`
- `cron_expr`: A 5-field UTC cron expression (minute hour day-of-month month day-of-week)
- `task`: A self-contained prompt for the agent
- `timezone`: The user's timezone (e.g., `"America/New_York"`)
- `description`: Human-readable description

**Important**: Minimum frequency is 1 hour. Schedules that fire more frequently will be rejected.

### One-shot Tasks

Use `create_schedule` with:
- `type`: `"one_shot"`
- `scheduled_at`: UTC ISO 8601 datetime (e.g., `"2024-01-15T14:00:00Z"`)
- `task`: A self-contained prompt for the agent
- `timezone`: The user's timezone
- `description`: Human-readable description

## Crafting Task Prompts

The `task` field must be a **self-contained prompt** that a fresh agent can execute without any conversation context. Include:

1. What to do (the action)
2. Any specific parameters or constraints
3. How to format the result

**Good example**: "Look up the current USD to EUR exchange rate using web search. Report the rate, the percentage change from yesterday, and any notable trends."

**Bad example**: "Check the exchange rate" (too vague, missing context)

## Timezone Handling

1. Determine the user's timezone from their message or stored memories
2. Convert the user's local time to a UTC cron expression
3. Store the timezone so results can be displayed in the user's local time

**Example**: User says "every day at 9am" and is in America/New_York (UTC-5):
- UTC cron: `0 14 * * *` (9am ET = 2pm UTC)
- timezone: `"America/New_York"`

## Cron Expression Examples

| Schedule | Cron (UTC) |
|----------|-----------|
| Daily at 9am UTC | `0 9 * * *` |
| Every 2 hours | `0 */2 * * *` |
| Weekdays at 8am UTC | `0 8 * * 1-5` |
| First day of month at noon | `0 12 1 * *` |

## Managing Schedules

- `list_schedules`: Shows all schedules with IDs, status, next run time, and last error
- `delete_schedule`: Removes a schedule by ID

## Retry Behavior

- Failed tasks retry once after 15 minutes
- If the retry also fails, the user receives a failure notification
- One-shot tasks are marked completed after successful execution
