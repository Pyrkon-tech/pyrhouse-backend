-- Permanent shop access for the system owner's non-domain Google account (the warehouse login has the same
-- exception in security.allowedEmails). An allowlist entry, so it stays manageable in the panel (Dostępy).
INSERT INTO shop_accounts (email, access_source)
VALUES ('warrmag7@gmail.com', 'allowlist')
ON CONFLICT (email) DO NOTHING;
