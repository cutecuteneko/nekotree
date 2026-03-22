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
	sh("go", "clean", "-cache", "-modcache")
	if val, ok := os.LookupEnv("GOROOT"); ok {
		fmt.Printf("⚠️  Found GOROOT=%s. Unsetting for this session.\n", val)
		os.Unsetenv("GOROOT")
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
			runDoctor(nil)
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}
	return setupVenv()
}

func buildBinary(c *cli.Context) error {
	fmt.Printf("🔨 Building %s...\n", BinaryName)
	os.MkdirAll(BuildDir, 0755)
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
		sh("go", "test", "-v", "./internal/...")
	}
	if c.Bool("all") || c.Bool("int") {
		sh("go", "test", "-v", "-tags=integration", "./integration/...")
	}
	return nil
}

func runDocs(c *cli.Context) error {
	docPath := filepath.Join(BuildDir, "docs")
	imgBuildPath := filepath.Join(docPath, "img")

	fmt.Println("🧹 Clearing and preparing doc directories...")
	os.RemoveAll(docPath)
	os.MkdirAll(filepath.Join(docPath, "api"), 0755)
	os.MkdirAll(imgBuildPath, 0755)

	// 1. Sync Manual Documentation
	manualDocs := []string{"index.md", "architecture.md"}
	for _, doc := range manualDocs {
		src := filepath.Join("docs", doc)
		if _, err := os.Stat(src); err == nil {
			sh("cp", src, filepath.Join(docPath, doc))
		}
	}

	// 2. Sync Existing Images
	fmt.Println("🖼️  Syncing static assets...")
	srcImgDir := filepath.Join("docs", "img")
	if _, err := os.Stat(srcImgDir); err == nil {
		filepath.Walk(srcImgDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				rel, _ := filepath.Rel(srcImgDir, path)
				target := filepath.Join(imgBuildPath, rel)
				os.MkdirAll(filepath.Dir(target), 0755)
				sh("cp", path, target)
			}
			return nil
		})
	}

	// 3. Generate UML & API Docs
	fmt.Println("📝 Generating Diagrams and API Markdown...")
	gopath := getGoPathBin()

	// Handle goplantuml redirection via Go file handling
	umlTool := filepath.Join(gopath, "goplantuml")
	pumlPath := filepath.Join(imgBuildPath, "api.puml")

	f, _ := os.Create(pumlPath)
	cmd := exec.Command(umlTool, "-recursive", "./internal")
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	cmd.Run()
	f.Close()

	// Convert .puml to .png via Docker
	fmt.Println("🐳 Converting UML to PNG via PlantUML Container...")
	absImgPath, _ := filepath.Abs(imgBuildPath)
	sh("docker", "run", "--rm", "-v", absImgPath+":/data", "plantuml/plantuml", "-o", "/data", "/data/api.puml")

	os.Remove(pumlPath)

	// API Markdown
	sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/config/...", "-o", filepath.Join(docPath, "api/config.md"))
	sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/docker/...", "-o", filepath.Join(docPath, "api/docker.md"))
	sh(filepath.Join(gopath, "gomarkdoc"), "--format", "github", "./internal/gitworktree/...", "-o", filepath.Join(docPath, "api/git.md"))

	// 4. Security Reports
	fmt.Println("🛡️  Running Security Reports...")
	vulnOut, _ := exec.Command(filepath.Join(gopath, "govulncheck"), "./...").Output()
	secOut, _ := exec.Command(filepath.Join(gopath, "gosec"), "-quiet", "./...").Output()

	secReport := fmt.Sprintf("# 🛡️ Security Report\n*Generated: %s*\n\n## Vulnerability Scan\n```text\n%s\n```\n\n## Static Analysis\n```text\n%s\n```",
		time.Now().Format(time.RFC822), string(vulnOut), string(secOut))
	os.WriteFile(filepath.Join(docPath, "security.md"), []byte(secReport), 0644)

	// 5. Coverage Analysis
	sh("go", "test", "-coverprofile="+filepath.Join(BuildDir, "cover.out"), "./...")
	coverage := calculateCoverage(filepath.Join(BuildDir, "cover.out"))

	covReport := fmt.Sprintf("# 📊 Test Coverage\n\n**Total Project Coverage:** %s%%\n\n*Detailed reports are available in the CI build artifacts.*", coverage)
	os.WriteFile(filepath.Join(docPath, "coverage.md"), []byte(covReport), 0644)

	updateBadges(coverage)

	if c.Bool("build") {
		return sh("./venv/bin/mkdocs", "build", "--config-file", "mkdocs.yml", "--site-dir", SiteDir)
	}
	if c.Bool("serve") {
		return sh("./venv/bin/mkdocs", "serve", "--config-file", "mkdocs.yml")
	}
	return nil
}

func runClean(c *cli.Context) error {
	os.RemoveAll(BuildDir)
	os.RemoveAll(SiteDir)
	os.RemoveAll("venv")
	return nil
}

func runRelease(c *cli.Context) error {
	platforms := []string{"linux/amd64", "darwin/amd64", "darwin/arm64"}
	for _, p := range platforms {
		parts := strings.Split(p, "/")
		osName, arch := parts[0], parts[1]
		targetPath := filepath.Join(BuildDir, fmt.Sprintf("%s-%s-%s", BinaryName, osName, arch))
		shEnv(map[string]string{"CGO_ENABLED": "0", "GOOS": osName, "GOARCH": arch}, "go", "build", "-o", targetPath, "./cmd/nekotree")
		writeChecksumFile(targetPath)
	}
	return nil
}

func writeChecksumFile(binaryPath string) error {
	hash, _ := calculateHash(binaryPath)
	content := fmt.Sprintf("%s  %s\n", hash, filepath.Base(binaryPath))
	return os.WriteFile(binaryPath+".sha256", []byte(content), 0644)
}

func calculateHash(path string) (string, error) {
	f, _ := os.Open(path)
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// --- Helpers ---

func sh(name string, args ...string) error { return shEnv(nil, name, args...) }

func shEnv(env map[string]string, name string, args ...string) error {
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
	if env != nil {
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd.Run()
}

func getGoPathBin() string {
	out, _ := exec.Command("go", "env", "GOPATH").Output()
	return filepath.Join(strings.TrimSpace(string(out)), "bin")
}

func setupVenv() error {
	if _, err := os.Stat("venv"); os.IsNotExist(err) {
		sh("python3", "-m", "venv", "venv")
	}
	sh("./venv/bin/pip", "install", "--upgrade", "pip")
	return sh("./venv/bin/pip", "install", "-r", "requirements.txt")
}

func calculateCoverage(path string) string {
	out, _ := exec.Command("go", "tool", "cover", "-func", path).Output()
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
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		reCov := regexp.MustCompile(`coverage-\d+(\.\d+)?%`)
		newContent := reCov.ReplaceAllString(string(content), "coverage-"+coverage+"%")
		os.WriteFile(path, []byte(newContent), 0644)
	}
}
