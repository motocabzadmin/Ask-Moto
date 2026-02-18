package knowledgebase

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/moto/ask-moto/internal/core/domain"
)

// FileStore implements the KnowledgeBaseRepository port using in-memory storage loaded from files
type FileStore struct {
	mu        sync.RWMutex
	documents map[string]*domain.Document
	chunks    []*domain.Chunk
	version   string
}

// NewFileStore creates a new file-based KB store
func NewFileStore(version string) *FileStore {
	return &FileStore{
		documents: make(map[string]*domain.Document),
		chunks:    make([]*domain.Chunk, 0),
		version:   version,
	}
}

// LoadFromDirectory loads all markdown files from a directory
func (s *FileStore) LoadFromDirectory(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		doc, err := s.loadMarkdownFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}

		s.documents[doc.ID] = doc

		// Chunk the document
		chunks := s.chunkDocument(doc)
		s.chunks = append(s.chunks, chunks...)
	}

	return nil
}

// loadMarkdownFile reads and parses a markdown file into a Document
func (s *FileStore) loadMarkdownFile(path string) (*domain.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var content strings.Builder
	var title string
	var module string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first H1
		if lineNum == 1 && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Extract module from front matter or filename
		if strings.HasPrefix(line, "module:") {
			module = strings.TrimSpace(strings.TrimPrefix(line, "module:"))
			continue
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Derive module from filename if not in content
	if module == "" {
		base := filepath.Base(path)
		module = strings.TrimSuffix(base, ".md")
	}

	// Generate ID from filename
	id := strings.TrimSuffix(filepath.Base(path), ".md")

	return &domain.Document{
		ID:        id,
		Title:     title,
		Content:   content.String(),
		Module:    module,
		Version:   s.version,
		UpdatedAt: time.Now(),
	}, nil
}

// chunkDocument splits a document into searchable chunks
func (s *FileStore) chunkDocument(doc *domain.Document) []*domain.Chunk {
	chunks := make([]*domain.Chunk, 0)

	// Split by sections (## headers) or paragraphs
	sections := splitIntoSections(doc.Content)

	for i, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		chunk := &domain.Chunk{
			ID:         fmt.Sprintf("%s-chunk-%d", doc.ID, i),
			DocumentID: doc.ID,
			Content:    section,
			Module:     doc.Module,
			StartIndex: 0, // Could be calculated if needed
			EndIndex:   len(section),
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

// splitIntoSections splits content by markdown headers or double newlines
func splitIntoSections(content string) []string {
	var sections []string
	var current strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Start new section on headers
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			if current.Len() > 0 {
				sections = append(sections, current.String())
				current.Reset()
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	// Don't forget the last section
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}

	return sections
}

// GetDocument retrieves a document by ID
func (s *FileStore) GetDocument(id string) (*domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	return doc, nil
}

// GetAllDocuments returns all documents
func (s *FileStore) GetAllDocuments() []*domain.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docs := make([]*domain.Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	return docs
}

// GetDocumentsByModule returns documents filtered by module
func (s *FileStore) GetDocumentsByModule(module string) []*domain.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var docs []*domain.Document
	for _, doc := range s.documents {
		if doc.Module == module {
			docs = append(docs, doc)
		}
	}
	return docs
}

// GetAllChunks returns all chunks
func (s *FileStore) GetAllChunks() []*domain.Chunk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.chunks
}

// AddDocument adds a document to the store
func (s *FileStore) AddDocument(doc *domain.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents[doc.ID] = doc
	chunks := s.chunkDocument(doc)
	s.chunks = append(s.chunks, chunks...)

	return nil
}
