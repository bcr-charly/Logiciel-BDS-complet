// bds est le point d'entrée du serveur BDS ENSIIE : un binaire autonome
// (stdlib Go exclusivement), déployable tel quel (binaire statique ou image
// Docker scratch) sur la machine de l'asso ARISE.
package main

import (
	"flag"
	"log"

	"bds/internal/app"
	"bds/internal/pages"
)

func main() {
	confPath := flag.String("conf", "conf.txt", "chemin du fichier de configuration")
	port := flag.String("port", "", "port d'écoute (override conf.txt)")
	creerAdmin := flag.String("creer-admin", "", "crée/met à jour un admin : email:motdepasse")
	flag.Parse()

	if err := app.Init(*confPath, *port, *creerAdmin); err != nil {
		log.Fatal(err)
	}
	pages.RegisterAll(app.A)

	if err := app.A.Run(); err != nil {
		log.Fatal(err)
	}
}
