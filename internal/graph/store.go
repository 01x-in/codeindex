package graph

// Store is the primary interface for the knowledge graph.
type Store interface {
	// Schema management
	Migrate() error
	Close() error

	// Write operations (used by indexer)
	UpsertNode(node Node) (int64, error)
	UpsertEdge(edge Edge) error
	SetFileMetadata(meta FileMetadata) error
	DeleteFileData(filePath string) error // removes all nodes/edges for a file
	SetIndexMetadata(key string, value string) error

	// Read operations (used by query engine)
	GetNode(id int64) (Node, error)
	FindNodesByName(name string) ([]Node, error)
	FindNodesByFile(filePath string) ([]Node, error)
	GetEdgesFrom(nodeID int64, kind string) ([]Edge, error)
	GetEdgesTo(nodeID int64, kind string) ([]Edge, error)
	GetFileMetadata(filePath string) (FileMetadata, error)
	GetAllFileMetadata() ([]FileMetadata, error)
	GetIndexMetadata(key string) (string, error)

	// Graph traversal
	GetNeighborhood(nodeID int64, depth int, edgeKinds []string) ([]Node, []Edge, error)

	// CTE-based traversal (single SQL query, avoids N+1)
	GetCallersCTE(nodeIDs []int64, maxDepth int) ([]CallerChainEntry, error)
	GetNeighborhoodCTE(nodeIDs []int64, maxDepth int, edgeKinds []string) ([]Node, []Edge, error)
}

// CallerChainEntry represents one entry in a caller chain returned by CTE traversal.
type CallerChainEntry struct {
	CallerNode Node
	Depth      int
}
