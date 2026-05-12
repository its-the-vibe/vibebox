# SKILL: Bank Statement Transaction Extractor

## Skill Purpose
Extract transactions from the "Your transactions" or "My transactions" section of a PNG bank statement, convert them to a pipe-separated (|) format with ISO-formatted dates, and save to a file.

## Skill Workflow
1. Accept a PNG file containing a bank statement as input.
2. Perform OCR to extract text from the image.
3. Identify the section labeled "Your transactions" or "My transactions".
4. Parse the transaction table, extracting:
   - Date
   - Description
   - Money In
   - Money Out
   - Balance
5. Convert all dates to ISO format (YYYY-MM-DD), inferring the year from the statement context if needed.
6. Output the data as a pipe-separated file (|), including a header row.
7. Save the output to a specified file (e.g., santander-YYYY-MM.tsv).

## Output Example
```
Date|Description|Money In|Money Out|Balance
2026-03-10|MAINTAINING THE ACCOUNT - MONTHLY FEE||3.00|737.26
... (etc)
```

## Notes
- Use pipe (|) as the delimiter to avoid issues with commas in descriptions.
- Infer the year from the statement period if not explicit in the date.
- Be robust to minor OCR and formatting inconsistencies.
