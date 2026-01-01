package predefine

import (
	"io/fs"
)

type MigrationFS fs.FS

type FrontendFS fs.ReadFileFS
