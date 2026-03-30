# Use Case: Web Search

## Scenario 1 — Search and synthesize a complex event
User asks about a major news event they missed. The agent searches,
fetches pages, and synthesizes a coherent summary.

### Steps
1. User: "Hey what happened with the CrowdStrike outage? I keep hearing about it but missed the details"
   → Agent searches the web and returns a coherent summary of the incident

## Scenario 2 — Store hours with temporal context
User asks whether a store is open right now. The agent combines web
search results with the current date/time from the system prompt.

### Steps
1. User: "Is Costco open right now?"
   → Agent searches for Costco hours and gives an actionable answer

## Scenario 3 — Version check with personalized reasoning
User asks about a software version and whether they should upgrade.
The agent relates search results back to the user's stated context.

### Steps
1. User: "Can you check if there's a new version of Postgres out? I'm on 16.2 and wondering if I should upgrade"
   → Agent searches for latest PostgreSQL version and advises on upgrading from 16.2

## Scenario 4 — Real-time stock price
User asks for a current stock price. The agent retrieves real-time
financial data and returns a numeric value.

### Steps
1. User: "What's AAPL trading at right now?"
   → Agent searches and returns a dollar price for Apple stock
