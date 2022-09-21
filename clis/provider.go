package clis

import (
	context "context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"regexp"
	"runtime"
	mutexkv "terraform-provider-clis/mutex"
)

var cliMutexKV = mutexkv.NewMutexKV()

var armArch = regexp.MustCompile(`^arm`)
var macos = regexp.MustCompile(`darwin`)

type EnvContext struct {
	Arch   string
	Os     string
	Alpine bool
}

func (c EnvContext) isArmArch() bool {
	return armArch.MatchString(c.Arch)
}

func (c EnvContext) isMacOs() bool {
	return macos.MatchString(c.Os)
}

func (c EnvContext) isAlpine() bool {
	return c.Alpine
}

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"bin_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory where the clis should be installed.",
				Default:     "bin",
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
	BinDir     string
	EnvContext EnvContext
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	binDir := d.Get("bin_dir").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	c := &ProviderConfig{
		BinDir: binDir,
		EnvContext: EnvContext{
			Arch:   runtime.GOARCH,
			Os:     runtime.GOOS,
			Alpine: checkForAlpine(),
		},
	}

	return c, diags
}
