# stmtpng2tsv

`stmtpng2tsv` extracts transactions from a PNG bank statement and writes them to a pipe-delimited TSV file.

## Usage

```bash
go run . -input /path/to/statement.png -output /path/to/santander-2026-03.tsv
# or with positional input path
go run . /path/to/statement.png -output /path/to/santander-2026-03.tsv
```

If `-output` is omitted, a default `<input-name>-YYYY-MM.tsv` file is generated next to the input file.

## Requirements

- GitHub Copilot authentication must be available for Copilot SDK usage.
- Use an image-capable model (for example `gpt-4.1`), so extraction can run directly from the PNG attachment.

## Build and test

```bash
make build
make test
```
