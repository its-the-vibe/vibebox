# goquery

[![CI](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml/badge.svg)](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml)

`goquery` is a CLI tool to run predefined SQL queries against Google Cloud BigQuery.

## Prerequisites

- Go 1.21+
- `GOOGLE_PROJECT_ID` environment variable set to your Google Cloud Project ID.
- Google Application Default Credentials (ADC) configured.

## Installation / Build

To build the binary, run:

```bash
make build
```

The binary will be located at `bin/goquery`.

## Usage

To run a predefined query:

```bash
./bin/goquery query monthly-balance-extremes
```

### Example Output

```
Year-Month | Max Balance | Max Date   | Min Balance | Min Date
----------------------------------------------------------------
2025-05    | 5230.50     | 2025-05-12 | 1120.00     | 2025-05-29
2025-04    | 4800.00     | 2025-04-01 | 950.25      | 2025-04-18
```

## Available Queries

- `monthly-balance-extremes`: Fetches monthly maximum and minimum balances from `transactions_ds.ledger`.

---

# vibebox

Toolbox containing vibetools.

## Tools

- [`stmtdate`](./stmtdate): Extracts PDF `creationDate`, computes statement month (`YYYY-MM`), and can rename files with that month suffix.
- [`stmtpng2tsv`](./stmtpng2tsv): Uses OCR + Gemini agent extraction (default) to convert PNG statement transactions into pipe-separated TSV output.
- `goquery`: CLI tool to run predefined BigQuery queries.
