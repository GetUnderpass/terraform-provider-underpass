// The Underpass Terraform provider (registry.terraform.io/getunderpass/underpass).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/getunderpass/terraform-provider-underpass/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set by goreleaser at release.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with support for debuggers like delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/getunderpass/underpass",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
