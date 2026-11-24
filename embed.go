package bds

import "embed"

// StaticFS embarque les assets, le web statique et les contenus éditoriaux par
// défaut. En exécution, les répertoires disque (assets/, web/, contenu/)
// priment sur l'embarqué quand ils existent.
//
//go:embed assets web contenu
var StaticFS embed.FS
