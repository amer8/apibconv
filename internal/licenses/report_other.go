//go:build android || ios || (!linux && !darwin) || (!amd64 && !arm64)

package licenses

import _ "embed"

//go:embed reports/fallback.txt
var report string
