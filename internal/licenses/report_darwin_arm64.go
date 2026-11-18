//go:build !ios

package licenses

import _ "embed"

//go:embed reports/darwin_arm64.txt
var report string
