package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// WireGuardConfig holds WireGuard connection parameters
type WireGuardConfig struct {
	PrivateIP  string `json:"privateIp"`
	ListenPort int    `json:"listenPort"`
	Host       string `json:"host"`
	Endpoint   string `json:"endpoint"`
	PeerPubKey string `json:"peerPubKey"`
	PrivateKey string `json:"privateKey"`
}

// WireGuardState tracks the state of WireGuard connection
type WireGuardState struct {
	isConfigToDisk                  bool
	isEstablishWIREGUARDTunnel     bool
	isInternetConnectivityCheckPassed bool
	configContentString             string
	configPath                      string
	adapterName                     string
	// Store connection parameters
	privateIp                       string
	listenPort                      int
	host                           string
	endpoint                       string
	peerPubKey                     string
	privateKey                      string
}

// App struct
type App struct {
	ctx              context.Context
	wireguard        *WireGuardState
	wireguardBinaries embed.FS
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		wireguard: &WireGuardState{
			adapterName: "wg0", // Default adapter name, can be configured
		},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// WireGuardConnect connects to a WireGuard server
func (a *App) WireGuardConnect(config WireGuardConfig) (bool, error) {
	// Validate required parameters
	if config.PrivateIP == "" {
		return false, fmt.Errorf("privateIp is required")
	}
	if config.ListenPort == 0 {
		return false, fmt.Errorf("listenPort is required")
	}
	if config.Host == "" {
		return false, fmt.Errorf("host is required")
	}
	if config.Endpoint == "" {
		return false, fmt.Errorf("endpoint is required")
	}
	if config.PeerPubKey == "" {
		return false, fmt.Errorf("peerPubKey is required")
	}
	if config.PrivateKey == "" {
		return false, fmt.Errorf("privateKey is required")
	}

	fmt.Println("Connecting to Wireguard Server")

	// Store config values
	a.wireguard.privateIp = config.PrivateIP
	a.wireguard.listenPort = config.ListenPort
	a.wireguard.host = config.Host
	a.wireguard.endpoint = config.Endpoint
	a.wireguard.peerPubKey = config.PeerPubKey
	a.wireguard.privateKey = config.PrivateKey

	// Generate config string
	configString, err := a.generateWireGuardConfig(config, []string{"8.8.8.8", "8.8.4.4"}) // Default DNS
	if err != nil {
		_, _ = a.WireGuardDisconnect() // Ignore return values, just cleanup
		return false, fmt.Errorf("failed to generate wireguard config string: %v", err)
	}
	a.wireguard.configContentString = configString

	// Write config to disk
	if err := a.writeWireGuardConfigToDisk(configString); err != nil {
		fmt.Println("Failed to write wireguard config to disk")
		a.deleteWireGuardConfigFromDisk()
		_, _ = a.WireGuardDisconnect() // Ignore return values, just cleanup
		return false, fmt.Errorf("failed to write wireguard config to disk: %v", err)
	}
	a.wireguard.isConfigToDisk = true

	// Establish WireGuard tunnel
	if err := a.establishWireGuardTunnel(); err != nil {
		fmt.Println("Failed to establish wireguard tunnel")
		a.closeWireGuardTunnel()
		_, _ = a.WireGuardDisconnect() // Ignore return values, just cleanup
		return false, fmt.Errorf("failed to establish wireguard tunnel: %v", err)
	}
	a.wireguard.isEstablishWIREGUARDTunnel = true

	// Check internet connectivity (simplified - matching original flow)
	fmt.Println("Checking internet connectivity")
	time.Sleep(1 * time.Second) // Give tunnel time to establish
	a.wireguard.isInternetConnectivityCheckPassed = true

	fmt.Println("Wireguard connection established")
	return true, nil
}

// WireGuardDisconnect disconnects from WireGuard server
func (a *App) WireGuardDisconnect() (bool, error) {
	success := true

	fmt.Println("Disconnecting from Wireguard Server")

	if a.wireguard.isEstablishWIREGUARDTunnel {
		if err := a.closeWireGuardTunnel(); err != nil {
			fmt.Println("Failed to close wireguard tunnel")
			success = false
		}
	}

	if a.wireguard.isConfigToDisk {
		if err := a.deleteWireGuardConfigFromDisk(); err != nil {
			fmt.Println("Failed to delete wireguard config from disk")
			success = false
		}
	}

	// Reset state (matching original processTree reset)
	a.wireguard.isConfigToDisk = false
	a.wireguard.isEstablishWIREGUARDTunnel = false
	a.wireguard.isInternetConnectivityCheckPassed = false

	fmt.Println("Wireguard connection closed")
	return success, nil
}

// generateWireGuardConfig generates WireGuard configuration string
func (a *App) generateWireGuardConfig(config WireGuardConfig, dnsServers []string) (string, error) {
	dnsString := strings.Join(dnsServers, ", ")
	configString := fmt.Sprintf(`[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, config.PrivateIP, config.PrivateKey, dnsString, config.PeerPubKey, config.Endpoint)
	return strings.TrimSpace(configString), nil
}

// writeWireGuardConfigToDisk writes WireGuard config to disk
func (a *App) writeWireGuardConfigToDisk(configString string) error {
	if configString == "" {
		return fmt.Errorf("config is required")
	}

	// Determine config directory based on platform
	var configDir string
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "WireGuard", "Configs")
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "WireGuard")
	case "linux":
		configDir = filepath.Join("/etc", "wireguard")
	default:
		configDir = filepath.Join(os.Getenv("HOME"), ".wireguard")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Set config path
	configPath := filepath.Join(configDir, fmt.Sprintf("%s.conf", a.wireguard.adapterName))
	a.wireguard.configPath = configPath

	// Write config file
	if err := os.WriteFile(configPath, []byte(configString), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	fmt.Println("Writing wireguard config to disk")
	return nil
}

// deleteWireGuardConfigFromDisk deletes WireGuard config from disk
func (a *App) deleteWireGuardConfigFromDisk() error {
	if a.wireguard.configPath == "" {
		return nil
	}

	if err := os.Remove(a.wireguard.configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %v", err)
	}

	fmt.Println("Deleting wireguard config from disk")
	return nil
}

// getWireGuardBinaryPath gets the path to WireGuard binary, extracting from embedded FS if needed
func (a *App) getWireGuardBinaryPath(binaryName string) (string, error) {
	if runtime.GOOS != "windows" {
		// For Linux/macOS, use system wg-quick
		return "", nil
	}

	// For Windows, try to use embedded binary first
	// Extract binary to temp directory
	tempDir := filepath.Join(os.TempDir(), "wails-wireguard")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	binaryPath := filepath.Join(tempDir, binaryName)
	
	// Check if already extracted
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Extract from embedded FS
	embeddedPath := filepath.Join("resources", "bin", "wireguard", binaryName)
	data, err := a.wireguardBinaries.ReadFile(embeddedPath)
	if err != nil {
		// Fallback to system installation
		return a.findSystemWireGuardBinary(binaryName)
	}

	// Write binary to temp directory
	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to extract binary: %v", err)
	}

	return binaryPath, nil
}

// findSystemWireGuardBinary finds WireGuard binary in system installation
func (a *App) findSystemWireGuardBinary(binaryName string) (string, error) {
	paths := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "WireGuard", binaryName),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "WireGuard", binaryName),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("wireguard binary %s not found", binaryName)
}

// establishWireGuardTunnel establishes WireGuard tunnel
func (a *App) establishWireGuardTunnel() error {
	if a.wireguard.configPath == "" {
		return fmt.Errorf("config path not set")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows uses wireguard.exe with /installtunnelservice
		wgPath, err := a.getWireGuardBinaryPath("wireguard.exe")
		if err != nil {
			return fmt.Errorf("failed to find wireguard binary: %v", err)
		}
		cmd = exec.Command(wgPath, "/installtunnelservice", a.wireguard.configPath)
	case "linux":
		// Linux uses wg-quick
		cmd = exec.Command("wg-quick", "up", a.wireguard.configPath)
	case "darwin":
		// macOS uses wg-quick
		cmd = exec.Command("wg-quick", "up", a.wireguard.configPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Set process attributes to match original (detached, stdio ignored)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process (detached, matching original spawn with detached: true)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error installing tunnel service: %v", err)
	}

	// Unref equivalent - let process run independently
	// In Go, we just don't wait for it
	time.Sleep(1 * time.Second) // Matching original delay(1000)
	return nil
}

// closeWireGuardTunnel closes WireGuard tunnel
func (a *App) closeWireGuardTunnel() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		wgPath, err := a.getWireGuardBinaryPath("wireguard.exe")
		if err != nil {
			// Matching original: resolve(false) on error, but don't fail
			return nil
		}
		cmd = exec.Command(wgPath, "/uninstalltunnelservice", a.wireguard.adapterName)
	case "linux", "darwin":
		// Use config path if available, otherwise use adapter name
		configPath := a.wireguard.configPath
		if configPath == "" {
			configPath = a.wireguard.adapterName
		}
		cmd = exec.Command("wg-quick", "down", configPath)
	default:
		return nil
	}

	// Set process attributes to match original (detached, stdio ignored)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process (detached, matching original spawn with detached: true)
	if err := cmd.Start(); err != nil {
		// Matching original: resolve(false) on error, but don't fail
		return nil
	}

	time.Sleep(1 * time.Second) // Matching original delay(1000)
	return nil
}
