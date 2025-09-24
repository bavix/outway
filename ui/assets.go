package ui

import "embed"

//go:embed dist/* dist/static/*
var Assets embed.FS
