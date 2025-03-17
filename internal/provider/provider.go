// Copyright (c) 2025 Cloud-Native Toolkit
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	mutexkv "github.com/cloud-native-toolkit/terraform-provider-clis/internal/mutex"
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var cliMutexKV = mutexkv.NewMutexKV()

// Ensure CliProvider satisfies various provider interfaces.
var _ provider.Provider = &CliProvider{}
var _ provider.ProviderWithFunctions = &CliProvider{}

// CliProvider defines the provider implementation.
type CliProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version    string
	EnvContext EnvContext
}

// CliProviderModel describes the provider data model.
type CliProviderModel struct {
	BinDir types.String `tfsdk:"bin_dir"`
}

type CliProviderDataSourceModel struct {
	BinDir     types.String
	EnvContext EnvContext
}

func (p *CliProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clis"
	resp.Version = p.version
}

func (p *CliProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bin_dir": schema.StringAttribute{
				MarkdownDescription: "The directory where the clis should be installed.",
				Optional:            true,
			},
		},
	}
}

func (p *CliProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CliProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = &CliProviderDataSourceModel{
		BinDir:     data.BinDir,
		EnvContext: p.EnvContext,
	}
	resp.ResourceData = &CliProviderDataSourceModel{
		BinDir:     data.BinDir,
		EnvContext: p.EnvContext,
	}
}

func (p *CliProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *CliProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCliCheckDataSource,
	}
}

func (p *CliProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CliProvider{
			version: version,
			EnvContext: EnvContext{
				Arch:   runtime.GOARCH,
				Os:     runtime.GOOS,
				Alpine: checkForAlpine(),
			},
		}
	}
}
