package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/nexspence/terraform-provider-nexspence/internal/provider"
)

// version is set by goreleaser via -ldflags.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with Terraform plugin debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/nexspence/nexspence",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
