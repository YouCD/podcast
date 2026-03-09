package dist

import "embed"

//go:embed  * assets/*
var Dist embed.FS
