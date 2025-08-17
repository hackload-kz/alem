CREATE VIRTUAL TABLE "events_archive_fts" USING fts5 (
    id,
    title,
    description
);

-- Note: The following commented-out section is for populating the FTS table.
-- INSERT INTO events_archive_fts (id, title, description)
-- SELECT id, title, description FROM events_archive;
