package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/nshreck/terraform-provider-ruckus/internal/provider"
)

func main() {
	if err := providerserver.Serve(
		context.Background(),
		provider.New,
		providerserver.ServeOpts{
			Address: "registry.terraform.io/nshreck/ruckus",
		},
	); err != nil {
		// Required for errcheck; a crash is appropriate here
		log.Fatalf("Error running Terraform provider server: %v", err)
	}
}
