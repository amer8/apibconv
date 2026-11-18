//go:build !android

package licenses

import _ "embed"

//go:embed reports/linux_amd64.txt
var report string
