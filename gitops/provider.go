package gitops

import (
	context "context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mutexkv "terraform-provider-gitops/mutex"
)

var gitopsMutexKV = mutexkv.NewMutexKV()

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_USERNAME", nil),
			},
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_TOKEN", nil),
			},
			"bin_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"lock": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_LOCK", "branch"),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitops_namespace": resourceGitopsNamespace(),
			"gitops_module":    resourceGitopsModule(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

type ProviderConfig struct {
	BinDir   string
	Username string
	Token    string
	Lock     string
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	binDir := d.Get("bin_dir").(string)
	username := d.Get("username").(string)
	token := d.Get("token").(string)
	lock := d.Get("lock").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	c := &ProviderConfig{
		BinDir:   binDir,
		Username: username,
		Token:    token,
		Lock:     lock,
	}

	return c, diags
}
