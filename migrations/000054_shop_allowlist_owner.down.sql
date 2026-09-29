-- Only an entry that never logged in or ordered; a used account stays.
DELETE FROM shop_accounts
WHERE email = 'warrmag7@gmail.com' AND google_sub IS NULL
  AND NOT EXISTS (SELECT 1 FROM shop_orders o WHERE o.account_id = shop_accounts.id);
