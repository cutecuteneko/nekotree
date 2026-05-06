package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	BinaryName    = "nekotree"
	BuildDir      = "build"
	SiteDir       = "site"
	ContainerName = "nekotree"

	ExpectedSize     = 6 * 1024 * 1024
	SizeTolerancePct = 10

	GoldenChecksum = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	VerifyChecksum = false
)

func main() {
	app := &cli.App{
		Name:  "nekotree-build",
		Usage: "Build and Documentation Pipeline for Nekotree",
		Commands: []*cli.Command{
			{
				Name:   "install-tools",
				Usage:  "Install required Go and Python tools",
				Action: installTools,
			},
			{
				Name:   "doctor",
				Usage:  "Fix toolchain version mismatches",
				Action: runDoctor,
			},
			{
				Name:   "build",
				Usage:  "Build the static binary",
				Action: buildBinary,
			},
			{
				Name:  "test",
				Usage: "Run test suite",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "int", Usage: "Run integration tests"},
					&cli.BoolFlag{Name: "all", Usage: "Run all tests"},
				},
				Action: runTests,
			},
			{
				Name:  "docs",
				Usage: "Documentation pipeline",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "build", Usage: "Build static site"},
					&cli.BoolFlag{Name: "serve", Usage: "Serve documentation locally"},
				},
				Action: runDocs,
			},
			{
				Name:   "release",
				Usage:  "Cross-compile for multiple platforms",
				Action: runRelease,
			},
			{
				Name:   "clean",
				Usage:  "Cleanup build artifacts",
				Action: runClean,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// --- Action Implementations ---

func runDoctor(c *cli.Context) error {
	fmt.Println("🩺 Running toolchain diagnostics...")
	if err := sh("go", "clean", "-cache", "-modcache"); err != nil {
		return err
	}
	if val, ok := os.LookupEnv("GOROOT"); ok {
		fmt.Printf("⚠️  Found GOROOT=%s. Unsetting for this session.\n", val)
		if err := os.Unsetenv("GOROOT"); err != nil {
			return err
		}
	}
	fmt.Println("✅ Diagnostics complete.")
	return nil
}

func installTools(c *cli.Context) error {
	tools := []string{
		"github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest",
		"github.com/jfeliu007/goplantuml/cmd/goplantuml@latest",
		"golang.org/x/vuln/cmd/govulncheck@latest",
		"github.com/securego/gosec/v2/cmd/gosec@latest",
		"github.com/blugnu/test-report@latest",
	}

	for _, tool := range tools {
		fmt.Printf("🛠️  Installing %s...\n", tool)
		if err := sh("go", "install", "-a", tool); err != nil {
			fmt.Println("❌ Installation failed. Attempting to fix toolchain...")
			_ = runDoctor(nil)
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}
	return setupVenv()
}

func buildBinary(c *cli.Context) error {
	fmt.Printf("🔨 Building %s...\n", BinaryName)
	if err := os.MkdirAll(BuildDir, 0750); err != nil {
		return err
	}
	targetPath := filepath.Join(BuildDir, BinaryName)
	err := shEnv(map[string]string{"CGO_ENABLED": "0"}, "go", "build", "-o", targetPath, "./cmd/nekotree")
	if err != nil {
		return err
	}
	return validateBinary(targetPath)
}

func validateBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}
	size := info.Size()
	minSize := int64(ExpectedSize * (100 - SizeTolerancePct) / 100)
	maxSize := int64(ExpectedSize * (100 + SizeTolerancePct) / 100)
	fmt.Printf("📏 Binary Size: %d bytes (Expected: %d - %d)\n", size, minSize, maxSize)
	if size < minSize || size > maxSize {
		return fmt.Errorf("🚨 VALIDATION FAILED: Binary size (%d) outside acceptable range", size)
	}
	fmt.Println("✅ Binary size integrity check passed.")
	return nil
}

func runTests(c *cli.Context) error {
	if c.Bool("all") || !c.Bool("int") {
		_ = os.MkdirAll(BuildDir, 0750)
		unitCov := filepath.Join(BuildDir, "unit.out")
		cmdCov := filepath.Join(BuildDir, "cmd.out")
		combinedCov := filepath.Join(BuildDir, "coverage.out")

		// Run internal package tests
		if err := sh("go", "test", "-v", "-coverprofile="+unitCov, "./internal/..."); err != nil {
			return err
		}

		// Run CLI package tests using internal as coverpkg
		if err := sh("go", "test", "-v", "-coverprofile="+cmdCov, "-coverpkg=./internal/...", "./cmd/nekotree"); err != nil {
			return err
		}

		mergeProfiles(combinedCov, unitCov, cmdCov)
	}
	if c.Bool("all") || c.Bool("int") {
		if err := sh("go", "test", "-v", "-tags=integration", "-coverprofile="+filepath.Join(BuildDir, "integration.out"), "./integration/..."); err != nil {
			return err
		}
	}
	return nil
}

func mergeProfiles(dest string, sources ...string) {
	var merged []string
	modeLine := "mode: set"

	for i, src := range sources {
		// Clean path to address G304
		content, err := os.ReadFile(filepath.Clean(src))
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 && i == 0 {
			modeLine = lines[0]
		}
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) != "" {
				merged = append(merged, lines[j])
			}
		}
	}
	final := modeLine + "\n" + strings.Join(merged, "\n")
	// #nosec G703 G306 - This is a local build script; paths are internal artifacts
	_ = os.WriteFile(filepath.Clean(dest), []byte(final), 0600)
}

func runDocs(c *cli.Context) error {
	docPath := filepath.Clean(filepath.Join(BuildDir, "docs"))
	imgBuildPath := filepath.Clean(filepath.Join(docPath, "img"))

	fmt.Println("🧹 Clearing and preparing doc directories...")
	_ = os.RemoveAll(docPath)
	if err := os.MkdirAll(filepath.Join(docPath, "api"), 0750); err != nil {
		return err
	}
	if err := os.MkdirAll(imgBuildPath, 0750); err != nil {
		return err
	}

	manualDocs := []string{"index.md", "architecture.md"}
	for _, doc := range manualDocs {
		src := filepath.Join("docs", doc)
		if _, err := os.Stat(src); err == nil {
			if err := sh("cp", src, filepath.Join(docPath, doc)); err != nil {
				return err
			}
		}
	}

	fmt.Println("🖼️  Syncing static assets...")
	srcImgDir := filepath.Join("docs", "img")
	if _, err := os.Stat(srcImgDir); err == nil {
		err = filepath.Walk(srcImgDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				rel, _ := filepath.Rel(srcImgDir, path)
				target := filepath.Join(imgBuildPath, rel)
				if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
					return err
				}
				return sh("cp", path, target)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	fmt.Println("📝 Generating Diagrams and API Markdown...")
	gopath := getGoPathBin()

	umlTool := filepath.Join(gopath, "goplantuml")
	pumlPath := filepath.Join(imgBuildPath, "api.puml")

	f, err := os.OpenFile(filepath.Clean(pumlPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	// #nosec G204
	cmd := exec.Command(umlTool, "-recursive", "./internal")
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Convert .puml to .png via plantum Docker container
	fmt.Println("🐳 Converting UML to PNG via PlantUML Container...")
	absImgPath, _ := filepath.Abs(imgBuildPath)
	if err := sh("docker", "run", "--rm", "-v", absImgPath+":/data", "plantuml/plantuml", "-o", "/data", "/data/api.puml"); err != nil {
		return err
	}
	_ = os.Remove(pumlPath)

	// API Markdown
	if err := sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/config/...", "-o", filepath.Join(docPath, "api/config.md")); err != nil {
		return err
	}
	if err := sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/docker/...", "-o", filepath.Join(docPath, "api/docker.md")); err != nil {
		return err
	}
	if err := sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/gitworktree/...", "-o", filepath.Join(docPath, "api/git.md")); err != nil {
		return err
	}

	fmt.Println("🛡️  Running Security Reports...")
	// #nosec G204
	vulnOut, _ := exec.Command(filepath.Join(gopath, "govulncheck"), "./...").Output()
	// #nosec G204
	secOut, _ := exec.Command(filepath.Join(gopath, "gosec"), "-quiet", "./...").Output()

	secReport := fmt.Sprintf("# 🛡️ Security Report\n*Generated: %s*\n\n## Vulnerability Scan\n```text\n%s\n```\n\n## Static Analysis\n```text\n%s\n```",
		time.Now().Format(time.RFC822), string(vulnOut), string(secOut))
	if err := os.WriteFile(filepath.Join(docPath, "security.md"), []byte(secReport), 0600); err != nil {
		return err
	}

	covPath := filepath.Join(BuildDir, "cover.out")
	if err := sh("go", "test", "-coverprofile="+covPath, "./..."); err != nil {
		return err
	}
	coverage := calculateCoverage(covPath)

	covReport := fmt.Sprintf("# 📊 Test Coverage\n\n**Total Project Coverage:** %s%%\n\n*Detailed reports are available in the CI build artifacts.*", coverage)
	if err := os.WriteFile(filepath.Join(docPath, "coverage.md"), []byte(covReport), 0600); err != nil {
		return err
	}

	updateBadges(coverage)

	if c.Bool("build") {
		return sh(".venv/bin/mkdocs", "build", "--config-file", "mkdocs.yaml", "--site-dir", SiteDir)
	}
	if c.Bool("serve") {
		return sh(".venv/bin/mkdocs", "serve", "--config-file", "mkdocs.yaml")
	}
	return nil
}

func runClean(c *cli.Context) error {
	_ = os.RemoveAll(BuildDir)
	_ = clearDir(SiteDir)
	_ = os.RemoveAll("venv")
	return nil
}

// clearDir removes all contents of a directory without deleting the directory itself.
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".gitkeep" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func runRelease(c *cli.Context) error {
	platforms := []string{"linux/amd64", "darwin/amd64", "darwin/arm64"}
	for _, p := range platforms {
		parts := strings.Split(p, "/")
		osName, arch := parts[0], parts[1]
		targetPath := filepath.Join(BuildDir, fmt.Sprintf("%s-%s-%s", BinaryName, osName, arch))
		if err := shEnv(map[string]string{"CGO_ENABLED": "0", "GOOS": osName, "GOARCH": arch}, "go", "build", "-o", targetPath, "./cmd/nekotree"); err != nil {
			return err
		}
		if err := writeChecksumFile(targetPath); err != nil {
			return err
		}
	}
	return nil
}

func writeChecksumFile(binaryPath string) error {
	hash, err := calculateHash(binaryPath)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("%s  %s\n", hash, filepath.Base(binaryPath))
	return os.WriteFile(binaryPath+".sha256", []byte(content), 0600)
}

func calculateHash(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// --- Helpers ---

func sh(name string, args ...string) error { return shEnv(nil, name, args...) }

func shEnv(env map[string]string, name string, args ...string) error {
	// #nosec G204
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cleanEnv := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOROOT=") {
			cleanEnv = append(cleanEnv, e)
		}
	}
	cmd.Env = cleanEnv
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd.Run()
}

func getGoPathBin() string {
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(string(out)), "bin")
}

func setupVenv() error {
	if _, err := os.Stat(".venv"); os.IsNotExist(err) {
		if err := sh("python3", "-m", "venv", ".venv"); err != nil {
			return err
		}
	}
	if err := sh(".venv/bin/pip", "install", "--upgrade", "pip"); err != nil {
		return err
	}
	return sh(".venv/bin/pip", "install", "-r", "requirements.txt")
}

func calculateCoverage(path string) string {
	// #nosec G204
	out, err := exec.Command("go", "tool", "cover", "-func", filepath.Clean(path)).Output()
	if err != nil {
		return "0.0"
	}
	re := regexp.MustCompile(`total:\s+\(statements\)\s+(\d+\.\d+)%`)
	match := re.FindStringSubmatch(string(out))
	if len(match) > 1 {
		return match[1]
	}
	return "0.0"
}

func updateBadges(coverage string) {
	paths := []string{"docs/index.md", filepath.Join(BuildDir, "docs/index.md")}
	for _, path := range paths {
		cleanedPath := filepath.Clean(path)
		content, err := os.ReadFile(cleanedPath)
		if err != nil {
			continue
		}
		reCov := regexp.MustCompile(`coverage-\d+(\.\d+)?%`)
		newContent := reCov.ReplaceAllString(string(content), "coverage-"+coverage+"%")
		// #nosec G703 -- path is validated via strict whitelist mapping above
		_ = os.WriteFile(cleanedPath, []byte(newContent), 0600)
	}
}
