DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'unique_pyr_code'
    ) THEN
        ALTER TABLE items ADD CONSTRAINT unique_pyr_code UNIQUE (pyr_code);
    END IF;
END $$; 