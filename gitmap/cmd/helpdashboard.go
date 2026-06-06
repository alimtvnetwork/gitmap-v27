package cmd

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
)

// maxDocsSiteSize is the maximum total extraction size for docs-site.zip (100 MB).
const maxDocsSiteSize = 100 * 1024 * 1024

// runHelpDashboard serves the docs site locally.
func runHelpDashboard(args []string) {
	checkHelp("help-dashboard", args)

	port := parseHelpDashboardFlags(args)
	binaryDir := resolveBinaryDir()
	docsDir := filepath.Join(binaryDir, constants.HDDocsDir)

	// Auto-extract docs-site.zip if docs-site/ directory doesn't exist.
	// If the zip is also missing (older installer, or `gitmap update` not yet
	// run after a docs-site release), try to download it from GitHub first.
	// If that also fails (release didn't bundle docs-site.zip), gracefully
	// fall back to opening the hosted docs URL instead of hard-exiting.
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		zipPath := filepath.Join(binaryDir, constants.DocsSiteArchive)
		if _, zipErr := os.Stat(zipPath); os.IsNotExist(zipErr) {
			if _, n, dlErr := downloadDocsSiteArchive(zipPath); dlErr != nil {
				fmt.Fprintf(os.Stderr, constants.ErrDocsSiteDownload, 2, dlErr, zipPath)
				openHostedDocsFallback()
				return
			} else {
				fmt.Printf(constants.MsgDocsSiteDownloaded, n)
			}
		}
		fmt.Printf("  Extracting %s...\n", constants.DocsSiteArchive)
		if extractErr := extractDocsSiteZip(zipPath, binaryDir); extractErr != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Failed to extract docs-site.zip: %v\n", extractErr)
			openHostedDocsFallback()
			return
		}
		fmt.Printf("  ✓ Docs site extracted to %s\n", docsDir)
	}

	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, constants.ErrHDNoDocsDir, docsDir)
		openHostedDocsFallback()
		return
	}

	distDir := filepath.Join(docsDir, constants.HDDistDir)

	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		serveStatic(distDir, port)
	} else {
		fmt.Print(constants.MsgHDNoDistFallback)
		serveDev(docsDir, port)
	}
}

// parseHelpDashboardFlags parses the --port flag.
func parseHelpDashboardFlags(args []string) int {
	fs := flag.NewFlagSet(constants.CmdHelpDashboard, flag.ExitOnError)
	port := fs.Int("port", constants.HDDefaultPort, constants.FlagDescHDPort)
	fs.Parse(args)

	return *port
}

// resolveBinaryDir returns the directory containing the gitmap binary.
func resolveBinaryDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return filepath.Dir(exe)
	}

	return filepath.Dir(resolved)
}

// serveStatic serves pre-built dist/ files over HTTP.
func serveStatic(distDir string, port int) {
	fmt.Printf(constants.MsgHDServingStatic, distDir, port)
	openBrowser(port)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           http.FileServer(http.Dir(distDir)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go handleShutdown(server)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, constants.ErrHDServe, err)
		os.Exit(1)
	}

	fmt.Print(constants.MsgHDStopped)
}

// serveDev runs npm install + npm run dev as a fallback.
func serveDev(docsDir string, port int) {
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		fmt.Fprint(os.Stderr, constants.ErrHDNPMNotFound)
		os.Exit(1)
	}

	fmt.Printf(constants.MsgHDRunningNPM)

	install := exec.Command(npmPath, "install")
	install.Dir = docsDir
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr

	if err := install.Run(); err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrHDNPMInstall, err)
		os.Exit(1)
	}

	fmt.Printf(constants.MsgHDStartingDev, docsDir)

	dev := exec.Command(npmPath, "run", "dev", "--", "--port", fmt.Sprintf("%d", port))
	dev.Dir = docsDir
	dev.Stdout = os.Stdout
	dev.Stderr = os.Stderr

	if err := dev.Start(); err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrHDDevServer, err)
		os.Exit(1)
	}

	openBrowser(port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	_ = dev.Process.Kill()
	fmt.Print(constants.MsgHDStopped)
}

// openBrowser opens the local dev/static URL in the default browser.
func openBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf(constants.MsgHDOpening, port)
	openURL(url)
}

// handleShutdown gracefully stops the static server on Ctrl+C.
func handleShutdown(server *http.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	server.Close()
}
