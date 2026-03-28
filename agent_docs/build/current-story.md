# Current Story: M1-S4

## Story
Content hashing + staleness detection

## Acceptance Criteria
- [x] SHA-256 content hash computed for each indexed file
- [x] `file_metadata` stores `content_hash` and `last_indexed_at`
- [ ] `IsStale(filePath)` reads current file, computes hash, compares to stored hash
- [ ] Returns `true` if hashes differ or file not in metadata
- [ ] Returns `false` if hashes match
- [ ] Handles deleted files (file gone from disk = stale)
- [ ] Handles new files (not in metadata = stale)
- [ ] Unit tests cover: fresh file, modified file, deleted file, new file

## Current State
- hash.File() and hash.Bytes() work
- Indexer.IsStale() exists but needs proper absolute path handling
- Need staleness tests
