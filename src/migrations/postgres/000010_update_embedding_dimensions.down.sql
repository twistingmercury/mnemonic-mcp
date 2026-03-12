-- src/migrations/postgres/000010_update_embedding_dimensions.down.sql
-- Reverses: restores pattern_chunks.embedding from vector(2000) to vector(1536).

-- Vector dimensions cannot be altered in place; must drop and recreate the HNSW index.
drop index if exists idx_pattern_chunks_embedding;

-- Restore the embedding column to the original dimension.
alter table pattern_chunks
    alter column embedding type vector(1536);

-- Recreate the HNSW index for the original dimension.
create index idx_pattern_chunks_embedding
    on pattern_chunks using hnsw (embedding vector_cosine_ops);

comment on index idx_pattern_chunks_embedding is
    'HNSW index for chunk vector similarity search; chosen over IVFFlat because it builds correctly on empty tables without centroid seeding';
