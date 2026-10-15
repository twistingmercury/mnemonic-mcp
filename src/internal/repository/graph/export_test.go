package graph

// ExportNewRepositoryWithFactory exposes newRepositoryWithFactory for testing.
var ExportNewRepositoryWithFactory = newRepositoryWithFactory

// newRepositoryWithFactory creates a Repository with a custom SessionFactory.
// This is used for unit testing with mocked sessions.
func newRepositoryWithFactory(factory SessionFactory) Repository {
	return &neo4jRepository{factory: factory}
}
