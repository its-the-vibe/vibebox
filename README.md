# vibebox

[![CI](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml/badge.svg)](https://github.com/its-the-vibe/vibebox/actions/workflows/ci.yaml)

Toolbox containing vibetools.

## Tools

- [`stmtdate`](./stmtdate): Extracts PDF `creationDate`, computes statement month (`YYYY-MM`), and can rename files with that month suffix.
- [`stmtpng2tsv`](./stmtpng2tsv): Uses OCR + Gemini agent extraction (default) to convert PNG statement transactions into pipe-separated TSV output.
- [`goquery`](./goquery): Runs predefined BigQuery SQL queries through a subcommand CLI (`goquery query <query-name>`).

## Development

Run commands across all sub-projects from the repository root:

```sh
make build   # build all sub-projects
make test    # run tests in all sub-projects
make tidy    # run go mod tidy in all sub-projects
```

To add a new sub-project, add its directory name to the `SUBDIRS` variable in the root `Makefile`.
