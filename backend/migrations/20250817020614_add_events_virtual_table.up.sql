CREATE VIRTUAL TABLE "events_archive_fts" USING fts5 (
    id,
    title,
    description
);