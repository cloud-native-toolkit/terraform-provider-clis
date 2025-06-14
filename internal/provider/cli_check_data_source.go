// Copyright (c) 2025 Cloud-Native Toolkit
// SPDX-License-Identifier: MIT

package provider

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CliCheckDataSource{}

func NewCliCheckDataSource() datasource.DataSource {
	return &CliCheckDataSource{}
}

var versionedInstallRe = regexp.MustCompile("([a-z-]+)-([0-9]+[.]?[0-9]*[.]?[0-9]*)")
var fullVersionRe = regexp.MustCompile("[0-9][.][0-9]+[.][0-9]+")

// CliCheckDataSource defines the data source implementation.
type CliCheckDataSource struct {
	BinDir     types.String
	EnvContext EnvContext
}

// CliCheckDataSourceModel describes the data source data model.
type CliCheckDataSourceModel struct {
	Id     types.String `tfsdk:"id"`
	Clis   types.List   `tfsdk:"clis"`
	BinDir types.String `tfsdk:"bin_dir"`
}

func (d *CliCheckDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (d *CliCheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data source to ensure required clis are available",

		Attributes: map[string]schema.Attribute{
			"clis": schema.ListAttribute{
				MarkdownDescription: "The list of clis that should be installed. Should be any of: jq, igc, yq, helm, argocd, rosa, kubeseal, oc, kustomize, ibmcloud, ibmcloud-is, ibmcloud-ob, ibmcloud-ks, ibmcloud-cr, gitu, gh, glab, openshift-install",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"bin_dir": schema.StringAttribute{
				MarkdownDescription: "The directory where the clis have been installed from the provider bin_dir config.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier",
				Computed:            true,
			},
		},
	}
}

func (d *CliCheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*CliProviderDataSourceModel)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CliProviderDataSourceModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.BinDir = provider.BinDir
	d.EnvContext = provider.EnvContext
}

func (d *CliCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CliCheckDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := d.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	binDir := first(typeStringsToStrings(data.BinDir, d.BinDir)...)

	defaultClis := []string{"yq", "jq", "igc", "kubeseal", "oc"}
	clis := unique(append(defaultClis, listTypeToStrings(data.Clis)...))

	tflog.Info(ctx, "Processing clis("+binDir+"): "+strings.Join(clis, ","))

	// Generate the id from the configured clis to save in state
	data.Id = types.StringValue("clis:" + strings.Join(clis[:], ":"))
	// Save the binDir into state
	data.BinDir = types.StringValue(binDir)

	err := addBinDirToPath(binDir)
	if err != nil {
		resp.Diagnostics.AddError("Error adding bin directory to path", fmt.Sprintf("Unable to add path, got error: %s", err))
		return
	}

	for _, cliName := range clis {
		if _, err := setupNamedCli(cliName, ctx, binDir, d.EnvContext); err != nil {
			resp.Diagnostics.AddError("Error setting up cli", fmt.Sprintf("Unable to setup cli, got error: %s", err))
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var installers map[string]func(ctx2 context.Context, binDir string, envContext EnvContext, version string) (bool, error)
var defaultVersions map[string]string

func getInstallers() map[string]func(ctx2 context.Context, binDir string, envContext EnvContext, version string) (bool, error) {
	if installers != nil {
		return installers
	}

	installers = make(map[string]func(ctx2 context.Context, binDir string, envContext EnvContext, version string) (bool, error))

	installers["jq"] = setupJq
	installers["igc"] = setupIgc
	installers["yq"] = setupYq
	installers["helm"] = setupHelm
	installers["argocd"] = setupArgoCD
	installers["rosa"] = setupRosa
	installers["kubeseal"] = setupKubeseal
	installers["oc"] = setupKube
	installers["kustomize"] = setupKustomize
	installers["ibmcloud"] = setupIBMCloud
	installers["ibmcloud-is"] = setupIBMCloudISPlugin
	installers["ibmcloud-ob"] = setupIBMCloudOBPlugin
	installers["ibmcloud-ks"] = setupIBMCloudKSPlugin
	installers["ibmcloud-cr"] = setupIBMCloudCRPlugin
	installers["gitu"] = setupGitu
	installers["gh"] = setupGh
	installers["glab"] = setupGlab
	installers["openshift-install"] = setupOpenShiftInstall
	installers["operator-sdk"] = setupOperatorSdk

	return installers
}

func getDefaultVersions() map[string]string {
	if defaultVersions != nil {
		return defaultVersions
	}

	defaultVersions = make(map[string]string)

	installersMap := getInstallers()
	// initialize with empty string
	for k := range installersMap {
		defaultVersions[k] = ""
	}

	defaultVersions["jq"] = "1.7.1"
	defaultVersions["igc"] = "1.50.2"
	defaultVersions["gitu"] = "1.15.0"

	return defaultVersions
}

func addBinDirToPath(binDir string) error {
	if len(binDir) == 0 {
		return nil
	}

	if !strings.HasPrefix(binDir, "/") {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}

		binDir = path.Join(cwd, binDir)
	}

	cliPath := os.Getenv("PATH")
	err := os.Setenv("PATH", fmt.Sprintf("%s:%s", binDir, cliPath))

	return err
}

func setupNamedCli(cliName string, ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	if cliName == "kubectl" {
		return false, nil
	}

	installers := getInstallers()

	version := ""
	if versionedInstallRe.MatchString(cliName) {
		nameParts := versionedInstallRe.FindStringSubmatch(cliName)

		if len(nameParts) < 3 {
			return false, fmt.Errorf("unable to parse versioned cli string: %s", cliName)
		}

		cliName = nameParts[1]
		version = nameParts[2]
	}

	if len(version) == 0 {
		version = getDefaultVersions()[cliName]
	}

	cliMutexKV.Lock(ctx, cliName)
	defer cliMutexKV.Unlock(ctx, cliName)

	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return false, fmt.Errorf("unable to create directory: %s, %w", destDir, err)
	}

	setupCli := installers[cliName]
	if setupCli == nil {
		return false, fmt.Errorf("unable to find installer for cli: %s", cliName)
	}

	return setupCli(ctx, destDir, envContext, version)
}

func setupJq(ctx context.Context, destDir string, envContext EnvContext, version string) (bool, error) {
	cliName := "jq"
	if cliAlreadyPresent(ctx, destDir, cliName, version) {
		return false, nil
	}

	filename := "jq-linux"

	if envContext.isMacOs() {
		filename = "jq-macos"
	}

	if envContext.isArmArch() {
		filename = filename + "-arm64"
	} else {
		filename = filename + "-amd64"
	}

	url := fmt.Sprintf("https://github.com/jqlang/jq/releases/download/jq-%s/%s", version, filename)

	return setupBinary(ctx, destDir, cliName, url, []string{"--version"}, version)
}

func setupIgc(ctx context.Context, destDir string, envContext EnvContext, version string) (bool, error) {
	cliName := "igc"
	if cliAlreadyPresent(ctx, destDir, cliName, version) {
		return false, nil
	}

	gitOrg := "cloud-native-toolkit"
	gitRepo := "ibm-garage-cloud-cli"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "x64"
	}

	var osName string
	if envContext.isMacOs() {
		osName = "macos"
	} else if envContext.isAlpine() {
		osName = "alpine"
	} else {
		osName = "linux"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/igc-%s-%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"--version"}, version)
}

func setupYq(ctx context.Context, destDir string, envContext EnvContext, _ string) (bool, error) {
	yq3Result, err := setupYq3(ctx, destDir, envContext, "")
	if err != nil {
		return false, err
	}

	yq4Result, err := setupYq4(ctx, destDir, envContext, "")
	if err != nil {
		return false, err
	}

	return yq3Result || yq4Result, nil
}

func setupYq3(ctx context.Context, destDir string, envContext EnvContext, _ string) (bool, error) {
	cliName := "yq3"
	if checkCurrentVersion(ctx, "yq", []string{"--version"}, "^3[.][0-9]*") {
		return createSymLink("yq", path.Join(destDir, cliName))
	}
	if checkCurrentVersion(ctx, "yq3", []string{"--version"}, "^3[.][0-9]*") {
		return createSymLink("yq3", path.Join(destDir, cliName))
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://github.com/mikefarah/yq/releases/download/3.4.1/yq_%s_%s", osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"--version"}, "")
}

func setupYq4(ctx context.Context, destDir string, envContext EnvContext, _ string) (bool, error) {
	cliName := "yq4"
	if checkCurrentVersion(ctx, "yq", []string{"--version"}, "^4[.][0-9]*") {
		return createSymLink("yq", path.Join(destDir, cliName))
	}
	if checkCurrentVersion(ctx, "yq4", []string{"--version"}, "^4[.][0-9]*") {
		return createSymLink("yq4", path.Join(destDir, cliName))
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://github.com/mikefarah/yq/releases/download/v4.25.2/yq_%s_%s", osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"--version"}, "")
}

func setupHelm(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "helm"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	filename := fmt.Sprintf("helm-v3.8.2-%s-%s.tar.gz", osName, arch)
	tgzPath := fmt.Sprintf("%s-%s/helm", osName, arch)

	url := fmt.Sprintf("https://get.helm.sh/%s", filename)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, tgzPath, []string{"version"}, minVersion)
}

func setupArgoCD(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "argocd"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "argoproj"
	gitRepo := "argo-cd"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/argocd-%s-%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"version", "--client"}, minVersion)
}

func setupRosa(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "rosa"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "macosx"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/rosa/latest/rosa-%s.tar.gz", arch, osName)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, []string{"version"}, minVersion)
}

func setupKubeseal(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "kubeseal"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "bitnami-labs"
	gitRepo := "sealed-secrets"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	shortRelease := strings.ReplaceAll(releaseInfo.TagName, "v", "")

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/kubeseal-%s-%s-%s.tar.gz", gitOrg, gitRepo, releaseInfo.TagName, shortRelease, osName, arch)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, []string{"--version"}, minVersion)
}

func setupKube(ctx context.Context, destDir string, envContext EnvContext, _ string) (bool, error) {
	ocResult, err := setupOc(ctx, destDir, envContext)
	if err != nil {
		return false, err
	}

	kubectlResult, err := setupKubectl(ctx, destDir, envContext)
	if err != nil {
		return false, err
	}

	return ocResult || kubectlResult, nil
}

func setupOc(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "oc"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "mac"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/ocp/stable/openshift-client-%s.tar.gz", arch, osName)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, []string{"version", "--client"}, "")
}

func setupKubectl(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "kubectl"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	resp, err := http.Get("https://dl.k8s.io/release/stable.txt")
	if err != nil {
		return false, err
	}
	defer func() {
		if tmpError := resp.Body.Close(); tmpError != nil {
			err = tmpError
		}
	}()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return false, err
	}
	release := buf.String()

	url := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/kubectl", release, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"version", "--client"}, "")
}

func setupKustomize(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "kustomize"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	filename := fmt.Sprintf("kustomize_v4.5.4_%s_%s.tar.gz", osName, arch)

	url := "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.5.4/" + filename

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, []string{"version"}, minVersion)
}

func setupGitu(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "gitu"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "cloud-native-toolkit"
	gitRepo := "git-client"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	var osName string
	if envContext.isMacOs() {
		osName = "macos"
	} else if envContext.isAlpine() {
		osName = "alpine"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "x64"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/gitu-%s-%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"--version"}, minVersion)
}

func setupGh(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "gh"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "cli"
	gitRepo := "cli"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	shortRelease := strings.ReplaceAll(releaseInfo.TagName, "v", "")

	var osName string
	if envContext.isMacOs() {
		osName = "macOS"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	filename := fmt.Sprintf("gh_%s_%s_%s", shortRelease, osName, arch)

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s.tar.gz", gitOrg, gitRepo, releaseInfo.TagName, filename)
	tgzPath := fmt.Sprintf("%s/bin/gh", filename)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, tgzPath, []string{"--version"}, minVersion)
}

func setupGlab(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "glab"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "profclems"
	gitRepo := "glab"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	shortRelease := strings.ReplaceAll(releaseInfo.TagName, "v", "")

	var osName string
	if envContext.isMacOs() {
		osName = "macOS"
	} else {
		osName = "Linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "x86_64"
	}

	filename := fmt.Sprintf("glab_%s_%s_%s", shortRelease, osName, arch)

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s.tar.gz", gitOrg, gitRepo, releaseInfo.TagName, filename)
	tgzPath := "bin/glab"

	return setupBinaryFromTgz(ctx, destDir, cliName, url, tgzPath, []string{"--version"}, minVersion)
}

func setupOpenShiftInstall(ctx context.Context, destDir string, envContext EnvContext, version string) (bool, error) {
	cliName := "openshift-install"
	if cliAlreadyPresent(ctx, destDir, cliName, version) {
		return false, nil
	}

	var osName string
	if envContext.isMacOs() {
		osName = "mac"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	var url string
	if len(version) == 0 || version == "4" {
		url = fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/ocp/stable/openshift-install-%s.tar.gz", arch, osName)
	} else if fullVersionRe.MatchString(version) {
		url = fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/ocp/%s/openshift-install-%s.tar.gz", arch, version, osName)
	} else {
		url = fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/ocp/stable-%s/openshift-install-%s.tar.gz", arch, version, osName)
	}

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, []string{"version"}, "")
}

func setupIBMCloud(ctx context.Context, destDir string, envContext EnvContext, _ string) (bool, error) {
	cliName := "ibmcloud"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
		return false, nil
	}

	gitOrg := "IBM-Cloud"
	gitRepo := "ibm-cloud-cli-release"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	shortRelease := strings.ReplaceAll(releaseInfo.TagName, "v", "")

	var osName string
	if envContext.isMacOs() {
		if envContext.isArmArch() {
			osName = "macos_arm64"
		} else {
			osName = "macos"
		}
	} else {
		if envContext.isArmArch() {
			osName = "linux_arm64"
		} else {
			osName = "linux_amd64"
		}
	}

	url := fmt.Sprintf("https://download.clis.cloud.ibm.com/ibm-cloud-cli/%s/binaries/IBM_Cloud_CLI_%s_%s.tgz", shortRelease, shortRelease, osName)

	result, err := setupBinaryFromTgz(ctx, destDir, cliName, url, "IBM_Cloud_CLI/ibmcloud", []string{"version"}, "")
	if err != nil {
		return false, err
	}

	cmd := exec.Command(filepath.Join(destDir, cliName), []string{"config", "--check-version=false"}...)
	err = cmd.Run()
	if err != nil {
		return false, err
	}

	return result, err
}

func setupIBMCloudISPlugin(ctx context.Context, destDir string, _ EnvContext, _ string) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "infrastructure-service")
}

func setupIBMCloudCRPlugin(ctx context.Context, destDir string, _ EnvContext, _ string) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "container-registry")
}

func setupIBMCloudKSPlugin(ctx context.Context, destDir string, _ EnvContext, _ string) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "kubernetes-service")
}

func setupIBMCloudOBPlugin(ctx context.Context, destDir string, _ EnvContext, _ string) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "observe-service")
}

func setupIBMCloudPlugin(ctx context.Context, destDir string, pluginName string) (bool, error) {

	if ibmcloudPluginExists(ctx, destDir, pluginName) {
		tflog.Debug(ctx, fmt.Sprintf("Plugin already installed: %s", pluginName))
		return false, nil
	}

	tflog.Info(ctx, fmt.Sprintf("Installing plugin: %s", pluginName))

	cmd := exec.Command(filepath.Join(destDir, "ibmcloud"), []string{"plugin", "install", pluginName}...)
	err := cmd.Run()
	if err != nil {
		return false, err
	}

	return true, nil
}

func setupOperatorSdk(ctx context.Context, destDir string, envContext EnvContext, minVersion string) (bool, error) {
	cliName := "operator-sdk"
	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	gitOrg := "operator-framework"
	gitRepo := "operator-sdk"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	var osName string
	if envContext.isMacOs() {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "amd64"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/operator-sdk_%s_%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, []string{"version"}, minVersion)
}

func ibmcloudPluginExists(ctx context.Context, destDir string, pluginName string) bool {

	cmd := exec.Command(filepath.Join(destDir, "ibmcloud"), []string{"plugin", "show", pluginName}...)
	err := cmd.Run()
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("IBM Cloud plugin not already installed: %s", pluginName))
		return false
	}

	return true
}

func getLatestGitHubRelease(org string, repo string) (*GitHubRelease, error) {

	url := fmt.Sprintf("https://github.com/%s/%s/releases/latest", org, repo)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tmpError := resp.Body.Close(); tmpError != nil {
			err = tmpError
		}
	}()

	latestUrl := resp.Header.Get("Location")
	if len(latestUrl) == 0 {
		return nil, fmt.Errorf("unable to retrieve location header from url: %s", url)
	}

	latestTagMatch := regexp.MustCompile(".*/tag/(.+)").FindStringSubmatch(latestUrl)
	if len(latestTagMatch) < 2 {
		return nil, fmt.Errorf("unable to parse latest tag from url: %s", latestUrl)
	}

	releaseInfo := &GitHubRelease{}
	releaseInfo.TagName = latestTagMatch[1]

	return releaseInfo, err
}

func cliAlreadyPresent(ctx context.Context, destDir string, cliName string, minVersion string) bool {
	cliPath, err := exec.LookPath(cliName)
	if err != nil || len(cliPath) == 0 {
		tflog.Debug(ctx, fmt.Sprintf("CLI not found in path: %s (%s)", cliName, err))
		return false
	}

	if strings.HasPrefix(cliPath, destDir) {
		tflog.Debug(ctx, fmt.Sprintf("CLI already provided in bin_dir: %s", cliName))
		return true
	}

	if len(minVersion) > 0 {
		out, err := exec.Command(cliName, "--version").Output()
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error getting cli version: %s", cliName))
		} else {
			versionString := cleanVersionString(string(out))
			if len(out) > 0 {
				tflog.Debug(ctx, fmt.Sprintf("Found version for cli: %s, %s", cliName, versionString))

				currentVersion, err1 := version.NewVersion(versionString)
				desiredVersion, err2 := version.NewVersion(minVersion)

				if err1 != nil {
					log.Fatal(err1)
				} else if err2 != nil {
					log.Fatal(err2)
				} else if currentVersion.LessThan(desiredVersion) {
					tflog.Debug(ctx, fmt.Sprintf("Current cli version is earlier than required version: %s < %s", versionString, minVersion))
					return false
				} else if currentVersion.GreaterThanOrEqual(desiredVersion) {
					tflog.Debug(ctx, fmt.Sprintf("Current cli version is same or newer than required version: %s >= %s", versionString, minVersion))
				}
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("CLI already available in PATH: %s. Creating symlink in %s", cliPath, destDir))
	result, err := createSymLink(cliName, filepath.Join(destDir, cliName))
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error creating symlink: %s, %s", cliName, err.Error()))
	}

	return result
}

func cleanVersionString(value string) string {
	regEx := `[^\d]*(?P<Major>\d+).(?P<Minor>\d+)[.]?(?P<Patch>\d*).*`
	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(value)

	cleanValue := ""
	for i := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			matchValue := match[i]

			if len(matchValue) == 0 {
				matchValue = "0"
			}

			if i > 1 {
				cleanValue = cleanValue + "."
			}
			cleanValue = cleanValue + matchValue
		}
	}

	return cleanValue
}

func setupBinary(ctx context.Context, destDir string, cliName string, url string, testArgs []string, minVersion string) (bool, error) {

	if cliAlreadyPresent(ctx, destDir, cliName, minVersion) {
		return false, nil
	}

	exists, err := fileExists(filepath.Join(destDir, cliName))
	if exists || err != nil {
		return false, err
	}

	tflog.Debug(ctx, fmt.Sprintf("Downloading cli (%s) from url: %s", cliName, url))

	err = writeFileFromUrl(url, destDir, cliName)
	if err != nil {
		return false, err
	}

	tflog.Trace(ctx, fmt.Sprintf("Testing downloaded cli: %s", cliName))

	cmd := exec.Command(filepath.Join(destDir, cliName), testArgs...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		return false, fmt.Errorf("unable to validate downloaded cli: %s, %s", filepath.Join(destDir, cliName), errb.String())
	}

	tflog.Debug(ctx, fmt.Sprintf("Validation of cli successful: %s, %s", filepath.Join(destDir, cliName), outb.String()))

	return true, err
}

func writeFileFromUrl(url string, destDir string, destFile string) error {
	out, err := os.OpenFile(filepath.Join(destDir, destFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer func() {
		if tempErr := out.Close(); tempErr != nil {
			err = tempErr
		}
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if tempErr := resp.Body.Close(); tempErr != nil {
			err = tempErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status retrieving file %s from url: %s, %s", destFile, resp.Status, url)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return err
}

func setupBinaryFromTgz(ctx context.Context, destDir string, cliName string, url string, tgzPath string, testArgs []string, _ string) (bool, error) {

	cliPath, err := exec.LookPath(cliName)
	if err == nil && len(cliPath) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("CLI already available in PATH: %s", cliPath))
		return false, nil
	}

	tflog.Debug(ctx, fmt.Sprintf("Downloading cli (%s) from %s", cliName, url))

	err = extractTarGxFromUrl(ctx, url, tgzPath, destDir, cliName)
	if err != nil {
		err = fmt.Errorf("unable to extract tgz from url: %s", url)
		return false, err
	}

	tflog.Trace(ctx, fmt.Sprintf("Testing downloaded cli: %s", cliName))

	cmd := exec.Command(filepath.Join(destDir, cliName), testArgs...)
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("unable to validate downloaded cli: %s", cliName)
	}

	return true, err
}

func extractTarGxFromUrl(ctx context.Context, url string, tgzPath string, destDir string, cliName string) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if tempErr := resp.Body.Close(); tempErr != nil {
			err = tempErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status retrieving cli %s: %s", cliName, resp.Status)
	}

	err = extractTarGz(ctx, resp.Body, tgzPath, destDir, cliName)

	return err
}

func extractTarGz(ctx context.Context, gzipStream io.Reader, targetFile string, destDir string, destFile string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if header.Name != targetFile {
				tflog.Trace(ctx, fmt.Sprintf("Skipping file in tgz: %s", header.Name))
				continue
			}

			tflog.Debug(ctx, fmt.Sprintf("Extracting file from tgz to destination: %s -> %s", header.Name, filepath.Join(destDir, destFile)))
			err = extractFileFromTar(ctx, tarReader, destDir, destFile)

			if err != nil {
				return err
			}

		default:
			tflog.Error(ctx, fmt.Sprintf("unknown type: %b in %s", header.Typeflag, header.Name))
		}
	}

	return err
}

func extractFileFromTar(ctx context.Context, tarReader io.Reader, destDir string, destFile string) error {
	outFileName := filepath.Join(destDir, destFile)

	outFile, err := os.OpenFile(outFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to create file: %s", outFileName))
		return err
	}
	defer func() {
		if tmpError := outFile.Close(); tmpError != nil {
			err = tmpError
		}
	}()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to copy file: %s", outFileName))
		return err
	}

	return err
}

func checkCurrentVersion(ctx context.Context, cli string, versionArgs []string, versionRegEx string) bool {

	cliPath, _ := exec.LookPath(cli)
	if len(cliPath) == 0 {
		return false
	}

	// extract version string
	cmd := exec.Command(cliPath, versionArgs...)
	var outb bytes.Buffer
	cmd.Stdout = &outb

	err := cmd.Run()
	if err != nil {
		return false
	}

	stdout := outb.String()

	tflog.Debug(ctx, fmt.Sprintf("Version output for cli: %s, %s", cli, stdout))

	r := regexp.MustCompile(`.*([0-9]+[.][0-9]+[.][0-9]+).*`)
	matches := r.FindStringSubmatch(stdout)
	if len(matches) < 2 {
		return false
	}

	version := matches[1]

	tflog.Debug(ctx, fmt.Sprintf("Found version string: %s, %s", cli, version))

	versionRegex := regexp.MustCompile(versionRegEx)
	return versionRegex.MatchString(version)
}

func createSymLink(cli string, linkTo string) (bool, error) {

	exists, err := fileExists(linkTo)
	if exists || err != nil {
		return false, err
	}

	cliPath, err := exec.LookPath(cli)
	if err != nil {
		return false, err
	}

	if cliPath == linkTo {
		return false, nil
	}

	err = os.Symlink(cliPath, linkTo)

	return true, err
}
