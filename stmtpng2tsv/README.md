# stmtpng2tsv

`stmtpng2tsv` extracts transactions from PNG bank statements and writes them to a pipe-delimited TSV file.

## Usage

```bash
go run . -input /path/to/statement.png -output /path/to/santander-2026-03.tsv
# or with positional input path
go run . /path/to/statement.png -output /path/to/santander-2026-03.tsv

# Explicit Santander format (default)
go run . -bank santander -input /path/to/statement.png -output /path/to/santander-2026-03.tsv

# SumUp format with multi-page PNG inputs
go run . -bank sumup -output /path/to/sumup-2026-03.tsv /path/to/statement-page1.png /path/to/statement-page2.png

# Using Gemini backend
export GEMINI_API_KEY=your_api_key
go run . -backend gemini -input /path/to/statement.png
```

If `-output` is omitted, a default `<input-name>-YYYY-MM.tsv` file is generated next to the first input file.

## Backends

- `copilot`: Uses GitHub Copilot SDK. Requires Copilot authentication.
- `gemini` (default): Uses Google Gemini models via `google.golang.org/genai`. Requires `GEMINI_API_KEY` environment variable.

## Requirements

- For Copilot: GitHub Copilot authentication must be available. Use an image-capable model (for example `gpt-4.1`).
- For Gemini: A valid Google Gemini API key. Default model is `gemini-flash-latest`.

## Build and test

```bash
make build
make test
```
