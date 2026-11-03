ALTER TABLE memory_mutations
  ADD COLUMN purge_attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN purge_after DATETIME(3) NULL,
  ADD COLUMN purge_error VARCHAR(255) NULL,
  ADD COLUMN purged_at DATETIME(3) NULL,
  ADD KEY idx_memory_purge_due (purge_status, purge_after);
