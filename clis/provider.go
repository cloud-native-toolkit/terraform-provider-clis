package clis

import (
	context "context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mutexkv "terraform-provider-clis/mutex"
)

var cliMutexKV = mutexkv.NewMutexKV()

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"bin_dir": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{},
		DataSourcesMap: map[string]*schema.Resource{
			"clis_check": dataClisCheck(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

type ProviderConfig struct {
	BinDir string
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	binDir := d.Get("bin_dir").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	c := &ProviderConfig{
		BinDir: binDir,
	}

	return c, diags
}
