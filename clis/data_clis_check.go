package clis

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var armArch = regexp.MustCompile(`^arm`)
var macos = regexp.MustCompile(`darwin`)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func dataClisCheck() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataClisCheckRead,
		Schema: map[string]*schema.Schema{
			"clis": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"bin_dir": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

var installers map[string]func(ctx2 context.Context, binDir string, envContext EnvContext) (bool, error)

func getInstallers() map[string]func(ctx2 context.Context, binDir string, envContext EnvContext) (bool, error) {
	if installers != nil {
		return installers
	}

	installers = make(map[string]func(ctx2 context.Context, binDir string, envContext EnvContext) (bool, error))

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

	return installers
}

type EnvContext struct {
	Arch string
	Os   string
}

func (c EnvContext) isArmArch() bool {
	return armArch.MatchString(c.Arch)
}

func (c EnvContext) isMacOs() bool {
	return macos.MatchString(c.Os)
}

func dataClisCheckRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	clis := interfacesToString(d.Get("clis").([]interface{}))
	config := m.(*ProviderConfig)

	binDir := config.BinDir

	err := d.Set("bin_dir", binDir)
	if err != nil {
		return diag.FromErr(err)
	}

	envContext := EnvContext{
		Arch: runtime.GOARCH,
		Os:   runtime.GOOS,
	}

	defaultClis := []string{"yq", "jq", "igc", "kubeseal", "oc"}

	clis = unique(append(defaultClis, clis...))

	cliPath := os.Getenv("PATH")
	err = os.Setenv("PATH", fmt.Sprintf("%s:%s", binDir, cliPath))
	if err != nil {
		return diag.FromErr(err)
	}

	for _, cliName := range clis {
		_, err := setupNamedCli(cliName, ctx, binDir, envContext)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func setupNamedCli(cliName string, ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	if cliName == "kubectl" {
		return false, nil
	}

	installers := getInstallers()

	cliMutexKV.Lock(cliName)
	defer cliMutexKV.Unlock(cliName)

	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return false, err
	}

	setupCli := installers[cliName]
	if setupCli == nil {
		return false, fmt.Errorf("unable to find installer for cli: %s", cliName)
	}

	return setupCli(ctx, destDir, envContext)
}

func setupJq(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "jq"
	if cliAlreadyPresent(ctx, destDir, cliName, "1.6") {
		return false, nil
	}

	if envContext.isArmArch() {
		tflog.Debug(ctx, "ARM not currently supported for jq. Trying amd64")
	}

	filename := "jq-linux64"
	if envContext.isMacOs() {
		filename = "jq-osx-amd64"
	}

	url := fmt.Sprintf("https://github.com/stedolan/jq/releases/download/jq-1.6/%s", filename)

	return setupBinary(ctx, destDir, cliName, url, "--version", "1.6")
}

func setupIgc(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "igc"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
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
	} else {
		osName = "linux"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/igc-%s-%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, "--version", "")
}

func setupYq(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	yq3Result, err := setupYq3(ctx, destDir, envContext)
	if err != nil {
		return false, err
	}

	yq4Result, err := setupYq4(ctx, destDir, envContext)
	if err != nil {
		return false, err
	}

	return yq3Result || yq4Result, nil
}

func setupYq3(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "yq3"
	if checkCurrentVersion("yq", "--version", "3[.]*") {
		return createSoftLink("yq", path.Join(destDir, cliName))
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

	return setupBinary(ctx, destDir, cliName, url, "--version", "")
}

func setupYq4(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "yq4"
	if checkCurrentVersion("yq", "--version", "4[.]*") {
		return createSoftLink("yq", path.Join(destDir, cliName))
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

	return setupBinary(ctx, destDir, cliName, url, "--version", "")
}

func setupHelm(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "helm"
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

	filename := fmt.Sprintf("helm-v3.8.2-%s-%s.tar.gz", osName, arch)
	tgzPath := fmt.Sprintf("%s-%s/helm", osName, arch)

	url := fmt.Sprintf("https://get.helm.sh/%s", filename)

	return setupBinaryFromTgz(ctx, destDir, cliName, url, tgzPath, "version", "")
}

func setupArgoCD(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "argocd"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
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

	return setupBinary(ctx, destDir, cliName, url, "version --client", "")
}

func setupRosa(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "rosa"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
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

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, "version", "")
}

func setupKubeseal(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "kubeseal"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
		return false, nil
	}

	gitOrg := "bitnami-labs"
	gitRepo := "sealed-secrets"

	releaseInfo, err := getLatestGitHubRelease(gitOrg, gitRepo)
	if err != nil {
		return false, err
	}

	shortRelease := strings.Replace(releaseInfo.TagName, "v", "", -1)

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

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, "--version", "")
}

func setupKube(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
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

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, "version --client=true", "")
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

	return setupBinary(ctx, destDir, cliName, url, "version --client=true", "")
}

func setupKustomize(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "kustomize"
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

	filename := fmt.Sprintf("kustomize_v4.5.4_%s_%s.tar.gz", osName, arch)

	url := "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.5.4/" + filename

	return setupBinaryFromTgz(ctx, destDir, cliName, url, cliName, "version", "")
}

func setupGitu(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
	cliName := "gitu"
	if cliAlreadyPresent(ctx, destDir, cliName, "") {
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
		osName = "darwin"
	} else {
		osName = "linux"
	}

	var arch string
	if envContext.isArmArch() {
		arch = "arm64"
	} else {
		arch = "x86"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/gitu-%s-%s", gitOrg, gitRepo, releaseInfo.TagName, osName, arch)

	return setupBinary(ctx, destDir, cliName, url, "--version", "")
}

func setupIBMCloud(ctx context.Context, destDir string, envContext EnvContext) (bool, error) {
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

	shortRelease := strings.Replace(releaseInfo.TagName, "v", "", -1)

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

	result, err := setupBinaryFromTgz(ctx, destDir, cliName, url, "IBM_Cloud_CLI/ibmcloud", "version", "")
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

func setupIBMCloudISPlugin(ctx context.Context, destDir string, _ EnvContext) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "infrastructure-service")
}

func setupIBMCloudCRPlugin(ctx context.Context, destDir string, _ EnvContext) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "container-registry")
}

func setupIBMCloudKSPlugin(ctx context.Context, destDir string, _ EnvContext) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "kubernetes-service")
}

func setupIBMCloudOBPlugin(ctx context.Context, destDir string, _ EnvContext) (bool, error) {
	return setupIBMCloudPlugin(ctx, destDir, "observe-service")
}

func setupIBMCloudPlugin(ctx context.Context, destDir string, pluginName string) (bool, error) {

	cliMutexKV.Lock("ibmcloud-plugin")
	defer cliMutexKV.Unlock("ibmcloud-plugin")

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

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", org, repo)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tmpError := resp.Body.Close(); tmpError != nil {
			err = tmpError
		}
	}()

	releaseInfo := &GitHubRelease{}
	err = json.NewDecoder(resp.Body).Decode(releaseInfo)

	return releaseInfo, err
}

func cliAlreadyPresent(ctx context.Context, destDir string, cliName string, _ string) bool {
	cliPath, err := exec.LookPath(cliName)
	if err != nil || len(cliPath) == 0 {
		tflog.Debug(ctx, fmt.Sprintf("CLI not found in path: %s", cliName))
		return false
	}

	if strings.HasPrefix(cliPath, destDir) {
		tflog.Debug(ctx, fmt.Sprintf("CLI already provided in bin_dir: %s", cliName))
		return true
	}

	// TODO check for matching cli version

	tflog.Debug(ctx, fmt.Sprintf("CLI already available in PATH: %s. Creating symlink in %s", cliPath, destDir))
	err = os.Symlink(cliPath, filepath.Join(destDir, cliName))
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error creating symlink: %s", cliName))
	}

	return true
}

func setupBinary(ctx context.Context, destDir string, cliName string, url string, testArgs string, _ string) (bool, error) {

	cliPath, err := exec.LookPath(cliName)
	if err == nil && len(cliPath) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("CLI already available: %s", destDir))
		return false, nil
	}

	tflog.Debug(ctx, fmt.Sprintf("Downloading cli (%s) from %s", cliName, url))

	err = writeFileFromUrl(url, destDir, cliName)
	if err != nil {
		return false, err
	}

	tflog.Trace(ctx, fmt.Sprintf("Testing downloaded cli: %s", cliName))

	cmd := exec.Command(filepath.Join(destDir, cliName), []string{testArgs}...)
	var outb bytes.Buffer
	cmd.Stdout = &outb

	err = cmd.Run()
	if err != nil {
		return false, fmt.Errorf("unable to validate downloaded cli: %s", filepath.Join(destDir, cliName))
	}

	tflog.Debug(ctx, fmt.Sprintf("Validation of cli successful: %s, %s", filepath.Join(destDir, cliName), outb.String())

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
		return fmt.Errorf("bad status retrieving file %s: %s", destFile, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return err
}

func setupBinaryFromTgz(ctx context.Context, destDir string, cliName string, url string, tgzPath string, testArgs string, _ string) (bool, error) {

	cliPath, err := exec.LookPath(cliName)
	if err == nil && len(cliPath) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("CLI already available: %s", destDir))
		return false, nil
	}

	tflog.Debug(ctx, fmt.Sprintf("Downloading cli (%s) from %s", cliName, url))

	err = extractTarGxFromUrl(ctx, url, tgzPath, destDir, cliName)

	tflog.Trace(ctx, fmt.Sprintf("Testing downloaded cli: %s", cliName))

	cmd := exec.Command(filepath.Join(destDir, cliName), []string{testArgs}...)
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

	for true {
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

func checkCurrentVersion(cli string, versionArg string, versionRegEx string) bool {

	cliPath, _ := exec.LookPath(cli)
	if len(cliPath) == 0 {
		return false
	}

	// extract version string
	cmd := exec.Command(cliPath, []string{versionArg}...)
	var outb bytes.Buffer
	cmd.Stdout = &outb

	err := cmd.Run()
	if err != nil {
		return false
	}

	r := regexp.MustCompile(`.*([0-9]+[.][0-9]+[.][0-9]+).*`)
	matches := r.FindStringSubmatch(outb.String())
	if len(matches) == 0 {
		return false
	}

	version := matches[0]

	versionRegex := regexp.MustCompile(versionRegEx)
	return versionRegex.MatchString(version)
}

func createSoftLink(cli string, linkTo string) (bool, error) {

	cliPath, err := exec.LookPath(cli)
	if err != nil {
		return false, err
	}

	err = os.Symlink(cliPath, linkTo)

	return true, err
}

func contains(list []string, item string) bool {
	for _, val := range list {
		if item == val {
			return true
		}
	}

	return false
}
