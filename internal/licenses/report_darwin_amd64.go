//go:build !ios

package licenses

import _ "embed"

//go:embed reports/darwin_amd64.txt
var report string
