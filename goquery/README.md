# goquery

[![CI](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml/badge.svg)](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml)

`goquery` is a Go CLI for running predefined BigQuery queries.

## Prerequisites

- Go 1.25.8+
- Google Application Default Credentials configured
- `GOOGLE_PROJECT_ID` environment variable set

## Build and test

```bash
make build
make test
make lint
```

## Usage

```bash
export GOOGLE_PROJECT_ID=my-gcp-project
./bin/goquery query monthly-balance-extremes
```

Example output format:

```text
Year-Month | Max Balance | Max Date   | Min Balance | Min Date
----------------------------------------------------------------
2025-05    | 5230.50     | 2025-05-12 | 1120.00     | 2025-05-29
2025-04    | 4800.00     | 2025-04-01 | 950.25      | 2025-04-18
```
