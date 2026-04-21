-- 0128_grants_new_tables.sql

GRANT SELECT, INSERT, UPDATE ON TABLE controlled_documents TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE profile_sequence_counters TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE user_process_areas TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE governance_events TO metaldocs_app;
