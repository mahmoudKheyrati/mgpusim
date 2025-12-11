package runner

import (
	"github.com/sarchlab/mgpusim/v4/amd/tracing/pageaccess"
)

// Re-export helper functions from pageaccess package for backward compatibility

// TrackMemoryAccess is re-exported from pageaccess package
// Deprecated: Use pageaccess.TrackMemoryAccess instead
var TrackMemoryAccess = pageaccess.TrackMemoryAccess

// TrackPageMigration is re-exported from pageaccess package
// Deprecated: Use pageaccess.TrackPageMigration instead
var TrackPageMigration = pageaccess.TrackPageMigration

// TrackPageReplication is re-exported from pageaccess package
// Deprecated: Use pageaccess.TrackPageReplication instead
var TrackPageReplication = pageaccess.TrackPageReplication
