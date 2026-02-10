package routing

import "time"

// ExportNewKeywordMatcherForTest exposes newKeywordMatcherForTest for external test packages.
var ExportNewKeywordMatcherForTest = newKeywordMatcherForTest

// ExportCleanExpiredPatterns exposes cleanExpiredPatterns for external test packages.
func ExportCleanExpiredPatterns(m *KeywordMatcher) {
	m.cleanExpiredPatterns()
}

// ExportPatternCacheLen returns the number of entries in the pattern cache.
func ExportPatternCacheLen(m *KeywordMatcher) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.patterns)
}

// ExportPatternLastUsed returns the lastUsed time for a cached pattern, or the
// zero value if the key is not present.
func ExportPatternLastUsed(m *KeywordMatcher, keyword string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if entry, ok := m.patterns[keyword]; ok {
		return entry.lastUsed
	}
	return time.Time{}
}
