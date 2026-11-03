ALTER TABLE memory_entries
  ADD COLUMN core BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN recheck_after DATETIME(3) NULL,
  ADD KEY idx_memory_scope_core (scope_id, core);
