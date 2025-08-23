CREATE VIRTUAL TABLE events_archive_fts USING fts5(
    id UNINDEXED,
    title,
    description,
    content='events_archive',
    content_rowid='id'
);

INSERT INTO events_archive_fts(events_archive_fts) VALUES('rebuild');
