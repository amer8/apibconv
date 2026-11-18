//go:build !android

package licenses

import _ "embed"

//go:embed reports/linux_arm64.txt
var report string
