package runner

import (
	"github.com/sarchlab/mgpusim/v4/amd/tracing/pageaccess"
)

// Re-export types and variables from pageaccess package for backward compatibility

// GlobalPageAccessTracer is the global page access tracer instance
// Deprecated: Use pageaccess.GlobalPageAccessTracer instead
var GlobalPageAccessTracer = &pageaccess.GlobalPageAccessTracer

// PageAccessTracer is re-exported from pageaccess package
// Deprecated: Use pageaccess.PageAccessTracer instead
type PageAccessTracer = pageaccess.PageAccessTracer

// PageAccessRecord is re-exported from pageaccess package
// Deprecated: Use pageaccess.PageAccessRecord instead
type PageAccessRecord = pageaccess.PageAccessRecord

// PageSharingSummary is re-exported from pageaccess package
// Deprecated: Use pageaccess.PageSharingSummary instead
type PageSharingSummary = pageaccess.PageSharingSummary

// PageMigrationRecord is re-exported from pageaccess package
// Deprecated: Use pageaccess.PageMigrationRecord instead
type PageMigrationRecord = pageaccess.PageMigrationRecord

// PageReplicationRecord is re-exported from pageaccess package
// Deprecated: Use pageaccess.PageReplicationRecord instead
type PageReplicationRecord = pageaccess.PageReplicationRecord

// NewPageAccessTracer is re-exported from pageaccess package
// Deprecated: Use pageaccess.NewPageAccessTracer instead
var NewPageAccessTracer = pageaccess.NewPageAccessTracer
