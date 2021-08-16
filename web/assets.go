package web

import (
	"embed"
)

//go:embed static/* template/*
var Assets embed.FS
