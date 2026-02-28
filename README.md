# OpenBotKit

A safe way to build agentic personal assitant for engineers

## Reimbursement CLI

CLI tool to automate expense reimbursement tracking. Fetches invoices from Gmail, parses financial statements (credit card & bank PDFs), cross-references transactions, and generates a submission-ready reimbursement package.

### How it works

1. **fetch** — Pulls invoice/receipt emails from Gmail via OAuth for configured accounts
2. **parse** — Extracts transactions from financial statement PDFs (Scapia CC, Axis CC, Axis bank) and dashboard invoices (Freepik) into SQLite
3. **recon** — Matches email invoices against statement transactions (±5 day window), detects surcharges (rent fees + GST), and stores results in a reconciliation database
4. **package** — Generates a zip with `reconciled.csv`, unlocked statement PDFs, and invoice attachments ready to submit

### Setup

#### Prerequisites

- Go 1.24+
- `pdftotext` (from poppler: `brew install poppler`)
- `qpdf` (for unlocking password-protected PDFs: `brew install qpdf`)
- Gmail API credentials (`credentials.json`)

#### Configuration

Create `config.yaml`:

```yaml
pdf_passwords:
  axis_cc: "YOUR_PASSWORD"
```

#### Directory structure

```
files/                         # User-provided input (gitignored contents)
├── finance/
│   ├── bank/axis/             # Bank statement PDFs
│   └── creditcard/
│       ├── axis/              # Axis CC statement PDFs
│       └── scapia/            # Scapia CC statement PDFs
└── source/
    ├── dashboards/freepik/    # Dashboard invoice PDFs
    └── offline/officeparty/   # Scanned offline receipts + manifest.csv

data/                          # App-generated (gitignored)
├── attachments/               # Email attachments
├── credentials.db             # OAuth tokens
├── emails.db                  # Fetched emails
├── statements.db              # Parsed transactions
└── reconciliation.db          # Reconciliation results
```

### Usage

```bash
go build -o reimbursement ./cmd/

# Fetch emails from Gmail
./reimbursement fetch

# Parse financial statement PDFs
./reimbursement parse

# Run reconciliation
./reimbursement recon

# Generate reimbursement package (zip)
./reimbursement package
```

### Supported services

| Service | Source | Detection |
|---------|--------|-----------|
| Claude Code | Anthropic receipts via email | `anthropic.com` sender |
| GitHub | GitHub payment receipts via email | `noreply@github.com` sender |
| GoDaddy | Order confirmations via email | `godaddy.com` sender |
| Freepik | Email invoices + dashboard PDFs | `freepik.com` sender |
| WeWork | MyHQ/WeWork invoices via email | `myhq.in`, `wework.com` sender |
| Office Party | Offline scanned receipts | `manifest.csv` in offline dir |

### Reconciliation

Transactions are matched across sources:

- **RECONCILED** — confirmed by 2+ sources (e.g. email invoice + CC statement). INR amount taken from statement.
- **UNRECONCILED** — found in only 1 source. Awaiting additional CC statements or manual entry.

Axis CC surcharges (Rent Transaction Fee 1% + GST 18%) on WeWork transactions are automatically detected and included as separate line items tied to the parent transaction.
