# stmtdate

`stmtdate` extracts a PDF `creationDate`, computes the statement month (one month earlier), and prints it as `YYYY-MM`.

## Usage

```bash
go run . -file /path/to/Statement.pdf
# or
 go run . /path/to/Statement.pdf
```

Output example:

```text
2026-03
```

Rename in place by appending the statement month:

```bash
go run . -rename /path/to/Statement.pdf
```

Example rename:

`Statement.pdf` → `Statement-2026-03.pdf`

## Build and test

```bash
make build
make test
```
