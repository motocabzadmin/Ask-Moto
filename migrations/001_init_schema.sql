-- Ask Moto pgvector Schema Migration
-- Enables semantic vector search for the knowledge base

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    module VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chunks table with vector embeddings
CREATE TABLE IF NOT EXISTS chunks (
    id VARCHAR(255) PRIMARY KEY,
    document_id VARCHAR(255) REFERENCES documents(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    module VARCHAR(100) NOT NULL,
    embedding vector(1536),  -- OpenAI text-embedding-3-small dimension
    start_index INT DEFAULT 0,
    end_index INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast vector similarity search using IVFFlat
-- Note: This index requires at least 100 rows to be effective
-- For smaller datasets, exact search will be used
CREATE INDEX IF NOT EXISTS chunks_embedding_idx ON chunks
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Index for module-based filtering
CREATE INDEX IF NOT EXISTS chunks_module_idx ON chunks(module);

-- Index for document lookup
CREATE INDEX IF NOT EXISTS chunks_document_id_idx ON chunks(document_id);
