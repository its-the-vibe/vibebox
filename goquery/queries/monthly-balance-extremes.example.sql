WITH ranked_balances AS (
  SELECT 
    FORMAT_DATE('%Y-%m', PARSE_DATE('%Y-%m-%d', date)) AS year_month,
    date,
    balance_gbp,
    ROW_NUMBER() OVER(
      PARTITION BY FORMAT_DATE('%Y-%m', PARSE_DATE('%Y-%m-%d', date)) 
      ORDER BY balance_gbp ASC, date ASC
    ) AS rank_min,
    ROW_NUMBER() OVER(
      PARTITION BY FORMAT_DATE('%Y-%m', PARSE_DATE('%Y-%m-%d', date)) 
      ORDER BY balance_gbp DESC, date ASC
    ) AS rank_max
  FROM 
    `transactions_ds.ledger`
  WHERE 
    date IS NOT NULL AND balance_gbp IS NOT NULL
)
SELECT
  max_b.year_month,
  max_b.balance_gbp AS max_balance,
  max_b.date AS max_balance_date,
  min_b.balance_gbp AS min_balance,
  min_b.date AS min_balance_date
FROM 
  (SELECT year_month, balance_gbp, date FROM ranked_balances WHERE rank_max = 1) max_b
JOIN 
  (SELECT year_month, balance_gbp, date FROM ranked_balances WHERE rank_min = 1) min_b
ON 
  max_b.year_month = min_b.year_month
ORDER BY 
  year_month DESC;
