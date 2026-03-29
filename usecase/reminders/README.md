# Use Case: Reminders

## Scenario 1 — One-shot reminder
User asks to be reminded to do something at a specific time.
The assistant creates a one-shot schedule and can list it back.

### Steps
1. User: "Remind me to call the dentist tomorrow at 3pm"
   → Agent creates a one-shot schedule
2. User: "What reminders do I have?"
   → Agent lists the dentist reminder with correct time

## Scenario 2 — Recurring reminder with execution
User asks for a daily EUR/USD exchange rate update. We verify both
the schedule creation and that the stored task executes correctly.

### Steps
1. User: "Tell me the EUR/USD exchange rate on telegram every morning at 10am"
   → Agent creates a recurring schedule with daily 10am cron
2. Simulate execution: run the stored task through a fresh agent with web tools
   → Agent searches for exchange rate and responds with EUR/USD rate
