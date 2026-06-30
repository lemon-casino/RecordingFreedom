package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/iconbuilder"
)

func main() {
	var (
		sourcePath       = flag.String("source", filepath.Join("build", "appicon.png"), "source PNG/JPEG image")
		sizesValue       = flag.String("sizes", iconbuilder.DefaultSizes, "comma-separated PNG sizes to generate")
		outputDir        = flag.String("out", filepath.Join("build", "icons"), "directory for generated PNG icon sizes")
		appIconPath      = flag.String("appicon", filepath.Join("build", "appicon.png"), "Wails source app icon path")
		appIconSize      = flag.Int("appicon-size", iconbuilder.DefaultAppIconSize, "pixel size for the Wails source app icon")
		frontendIcons    = flag.String("frontend", strings.Join([]string{filepath.Join("frontend", "public", "appicon.png"), filepath.Join("frontend", "public", "wails.png")}, ","), "comma-separated frontend icon paths")
		frontendIconSize = flag.Int("frontend-size", 256, "pixel size for frontend icon files")
		skipWails        = flag.Bool("skip-wails", false, "skip regenerating Wails .ico/.icns platform icons")
	)
	flag.Parse()

	sizes, err := iconbuilder.ParseSizes(*sizesValue)
	if err != nil {
		fatal(err)
	}

	report, err := iconbuilder.Run(context.Background(), iconbuilder.Options{
		SourcePath:        *sourcePath,
		Sizes:             sizes,
		OutputDir:         *outputDir,
		AppIconPath:       *appIconPath,
		AppIconSize:       *appIconSize,
		FrontendIconPaths: splitList(*frontendIcons),
		FrontendIconSize:  *frontendIconSize,
	})
	if err != nil {
		fatal(err)
	}

	for _, icon := range report.Generated {
		fmt.Printf("generated %4dpx  %s\n", icon.Size, icon.Path)
	}
	if report.AppIconPath != "" {
		fmt.Printf("updated app icon   %s\n", report.AppIconPath)
	}
	for _, path := range report.FrontendIconPaths {
		fmt.Printf("updated frontend   %s\n", path)
	}

	if *skipWails {
		return
	}
	if err := regenerateWailsIcons(*appIconPath); err != nil {
		fatal(err)
	}
	fmt.Println("regenerated Wails platform icons")
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func regenerateWailsIcons(appIconPath string) error {
	buildDir := filepath.Dir(appIconPath)
	input := filepath.Base(appIconPath)
	cmd := exec.Command(
		"wails3",
		"generate",
		"icons",
		"-input", input,
		"-macfilename", filepath.Join("darwin", "icons.icns"),
		"-windowsfilename", filepath.Join("windows", "icon.ico"),
	)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "icon-build:", err)
	os.Exit(1)
}
