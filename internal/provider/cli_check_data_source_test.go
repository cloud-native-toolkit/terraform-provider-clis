// Copyright (c) 2025 Cloud-Native Toolkit
// SPDX-License-Identifier: MIT

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccExampleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.clis_check.clis",
						tfjsonpath.New("id"),
						knownvalue.StringExact("clis:yq:jq:igc:kubeseal:oc"),
					),
					statecheck.ExpectKnownValue(
						"data.clis_check.clis",
						tfjsonpath.New("clis"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("jq"),
						}),
					),
				},
			},
		},
	})
}

const testAccExampleDataSourceConfig = `
data "clis_check" "clis" {
  clis = ["jq"]
  bin_dir = "/tmp/test_bin"
}
`
