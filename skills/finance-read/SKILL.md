---
name: finance-read
description: Look up stock prices, exchange rates, and market data
allowed-tools: Bash(obk finance *)
---

## Commands

### Stock quote
`obk finance quote AAPL` — get current price for one or more symbols
`obk finance quote AAPL GOOGL MSFT --json` — JSON output

### Exchange rate
`obk finance rate USD INR` — get current exchange rate
`obk finance rate EUR GBP --json` — JSON output

## Symbol formats
- Stocks: `AAPL`, `GOOGL`, `MSFT`, `TSLA`
- ETFs: `SPY`, `QQQ`, `VOO`
- Crypto: `BTC-USD`, `ETH-USD`
- Indices: `^GSPC` (S&P 500), `^DJI` (Dow Jones)
