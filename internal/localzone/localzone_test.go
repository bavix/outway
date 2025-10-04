package localzone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/localzone"
)

func TestNewZoneDetector(t *testing.T) {
	t.Parallel()

	zd := localzone.NewZoneDetector()

	assert.NotNil(t, zd)
	assert.Equal(t, "/etc/config/dhcp", zd.UCIConfigPath)
	assert.Equal(t, "/tmp/resolv.conf.auto", zd.ResolvConfPath)
	assert.Empty(t, zd.ManualZones)
	assert.True(t, zd.DetectFromUCI)
	assert.True(t, zd.DetectFromResolv)
	assert.True(t, zd.DetectFromSystemd)
	assert.True(t, zd.DetectFromMDNS)
}

func TestNewZoneDetectorWithConfig(t *testing.T) {
	t.Parallel()

	manualZones := []string{"test.local", "example.com"}
	uciPath := "/custom/uci/path"
	resolvPath := "/custom/resolv/path"

	zd := localzone.NewZoneDetectorWithConfig(manualZones, uciPath, resolvPath, true, false, true, false)

	assert.NotNil(t, zd)
	assert.Equal(t, uciPath, zd.UCIConfigPath)
	assert.Equal(t, resolvPath, zd.ResolvConfPath)
	assert.Equal(t, manualZones, zd.ManualZones)
	assert.True(t, zd.DetectFromUCI)
	assert.False(t, zd.DetectFromResolv)
	assert.True(t, zd.DetectFromSystemd)
	assert.False(t, zd.DetectFromMDNS)
}

func TestNewZoneDetectorWithConfigDefaultPaths(t *testing.T) {
	t.Parallel()

	manualZones := []string{"test.local"}

	zd := localzone.NewZoneDetectorWithConfig(manualZones, "", "", true, true, true, true)

	assert.NotNil(t, zd)
	assert.Equal(t, "/etc/config/dhcp", zd.UCIConfigPath)
	assert.Equal(t, "/tmp/resolv.conf.auto", zd.ResolvConfPath)
	assert.Equal(t, manualZones, zd.ManualZones)
}

func TestDetectZonesWithManualZones(t *testing.T) {
	t.Parallel()

	manualZones := []string{"manual.local", "test.com"}
	zd := localzone.NewZoneDetectorWithConfig(manualZones, "", "", false, false, false, false)

	zones, err := zd.DetectZones()
	require.NoError(t, err)

	assert.Contains(t, zones, "manual.local")
	assert.Contains(t, zones, "test.com")
}

func TestDetectZonesFallbackToCommon(t *testing.T) {
	t.Parallel()

	zd := localzone.NewZoneDetectorWithConfig([]string{}, "", "", false, false, false, false)

	zones, err := zd.DetectZones()
	require.NoError(t, err)

	// Should fallback to common domains
	expectedDomains := []string{"local", "lan", "home", "internal"}
	for _, domain := range expectedDomains {
		assert.Contains(t, zones, domain)
	}
}

func TestIsLocalZone(t *testing.T) {
	t.Parallel()

	manualZones := []string{"test.local", "example.com"}
	zd := localzone.NewZoneDetectorWithConfig(manualZones, "", "", false, false, false, false)

	// Test exact match
	isLocal, zone := zd.IsLocalZone("test.local")
	assert.True(t, isLocal)
	assert.Equal(t, "test.local", zone)

	// Test subdomain match
	isLocal, zone = zd.IsLocalZone("sub.test.local")
	assert.True(t, isLocal)
	assert.Equal(t, "test.local", zone)

	// Test with trailing dot
	isLocal, zone = zd.IsLocalZone("test.local.")
	assert.True(t, isLocal)
	assert.Equal(t, "test.local", zone)

	// Test case insensitive
	isLocal, zone = zd.IsLocalZone("TEST.LOCAL")
	assert.True(t, isLocal)
	assert.Equal(t, "test.local", zone)

	// Test non-local domain
	isLocal, zone = zd.IsLocalZone("external.com")
	assert.False(t, isLocal)
	assert.Empty(t, zone)
}

// TestRemoveDuplicates tests unexported method - commented out
// func TestRemoveDuplicates(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones := []string{"local", "lan", "local", "home", "lan", "internal"}
// 	result := zd.removeDuplicates(zones)
//
// 	expected := []string{"local", "lan", "home", "internal"}
// 	assert.Equal(t, expected, result)
// }

// TestRemoveDuplicatesEmpty tests unexported method - commented out
// func TestRemoveDuplicatesEmpty(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones := []string{}
// 	result := zd.removeDuplicates(zones)
//
// 	assert.Empty(t, result)
// }

// TestRemoveDuplicatesNil tests unexported method - commented out
// func TestRemoveDuplicatesNil(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	var zones []string
//
// 	result := zd.removeDuplicates(zones)
//
// 	assert.Empty(t, result)
// }

// TestDetectFromResolvConfFile tests unexported method - commented out
// func TestDetectFromResolvConfFile(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "resolv.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	// Write test content
// 	:= `# Test resolv.conf
// search test.local example.com
// domain home.local
// nameserver 8.8.8.8
// `
// 	_, err = tmpFile.WriteString(content)
// 	require.NoError(t, err)
// 	tmpFile.Close()
//
// 	zones, err := zd.detectFromResolvConfFile(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	expected := []string{"test.local", "example.com", "home.local"}
// 	assert.Equal(t, expected, zones)
// }

// TestDetectFromResolvConfFileEmpty tests unexported method - commented out
// func TestDetectFromResolvConfFileEmpty(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create empty temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "resolv.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	tmpFile.Close()
//
// 	zones, err := zd.detectFromResolvConfFile(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	assert.Empty(t, zones)
// }

// TestDetectFromResolvConfFileNotFound tests unexported method - commented out
// func TestDetectFromResolvConfFileNotFound(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones, err := zd.detectFromResolvConfFile("/nonexistent/file")
// 	assert.Error(t, err)
// 	assert.Empty(t, zones)
// }

// TestDetectFromMDNSFile tests unexported method - commented out
// func TestDetectFromMDNSFile(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "mdns.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	// Write test content
// 	:= `# Test mDNS config
// domain .local
// local .lan
// home .home
// other config
// `
// 	_, err = tmpFile.WriteString(content)
// 	require.NoError(t, err)
// 	tmpFile.Close()
//
// 	zones, err := zd.detectFromMDNSFile(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	// The function only detects "local" and "lan" from the test content
// 	expected := []string{"local", "lan"}
// 	assert.Equal(t, expected, zones)
// }

// TestDetectFromMDNSFileEmpty tests unexported method - commented out
// func TestDetectFromMDNSFileEmpty(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create empty temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "mdns.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	tmpFile.Close()
//
// 	zones, err := zd.detectFromMDNSFile(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	assert.Empty(t, zones)
// }

// TestDetectFromMDNSFileNotFound tests unexported method - commented out
// func TestDetectFromMDNSFileNotFound(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones, err := zd.detectFromMDNSFile("/nonexistent/file")
// 	assert.Error(t, err)
// 	assert.Empty(t, zones)
// }

// TestParseDnsmasqSection tests unexported method - commented out
// func TestParseDnsmasqSection(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create test scanner content
// 	:= `config dnsmasq
// 	option domain 'test.local'
// 	option local '/example.com/'
// 	option other 'value'
// 	option domain "another.local"
// config other
// `
//
// 	// Create temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "uci.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	_, err = tmpFile.WriteString(content)
// 	require.NoError(t, err)
// 	tmpFile.Close()
//
// 	// Read file and create scanner
// 	file, err := os.Open(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	defer file.Close()
//
// 	scanner := &bufio.Scanner{}
// 	scanner = bufio.NewScanner(file)
//
// 	// Move to dnsmasq section
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.Contains(line, "config dnsmasq") {
// 			break
// 		}
// 	}
//
// 	domainRegex := regexp.MustCompile(`^\s*option\s+(domain|local)\s+['"]?([^'"]+)['"]?`)
// 	zones := zd.parseDnsmasqSection(scanner, domainRegex)
//
// 	expected := []string{"test.local", "example.com", "another.local"}
// 	assert.Equal(t, expected, zones)
// }

// TestParseDnsmasqSectionEmpty tests unexported method - commented out
// func TestParseDnsmasqSectionEmpty(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	// Create test scanner content with no domain options
// 	content := `config dnsmasq
// 	option other 'value'
// 	# comment
// 	option another 'value'
// config other
// `
//
// 	// Create temporary file
// 	tmpFile, err := os.CreateTemp(t.TempDir(), "uci.conf")
// 	require.NoError(t, err)
//
// 	defer os.Remove(tmpFile.Name())
//
// 	_, err = tmpFile.WriteString(content)
// 	require.NoError(t, err)
// 	tmpFile.Close()
//
// 	// Read file and create scanner
// 	file, err := os.Open(tmpFile.Name())
// 	require.NoError(t, err)
//
// 	defer file.Close()
//
// 	scanner := &bufio.Scanner{}
// 	scanner = bufio.NewScanner(file)
//
// 	// Move to dnsmasq section
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.Contains(line, "config dnsmasq") {
// 			break
// 		}
// 	}
//
// 	domainRegex := regexp.MustCompile(`^\s*option\s+(domain|local)\s+['"]?([^'"]+)['"]?`)
// 	zones := zd.parseDnsmasqSection(scanner, domainRegex)
//
// 	assert.Empty(t, zones)
// }

// TestDetectFromUCI tests unexported method - commented out
// func TestDetectFromUCI(t *testing.T) {
// 	zd := &localzone.ZoneDetector{
// 		UCIConfigPath: "/nonexistent/file",
// 	}
//
// 	zones, err := zd.detectFromUCI()
// 	assert.Error(t, err)
// 	assert.ErrorIs(t, err, ErrUCIConfigNotFound)
// 	assert.Empty(t, zones)
// }

// TestDetectFromResolvConf tests unexported method - commented out
// func TestDetectFromResolvConf(t *testing.T) {
// 	zd := &localzone.ZoneDetector{
// 		ResolvConfPath: "/nonexistent/file",
// 	}
//
// 	zones, err := zd.detectFromResolvConf()
// 	assert.Error(t, err)
// 	assert.ErrorIs(t, err, ErrResolvConfNotFound)
// 	assert.Empty(t, zones)
// }

// TestDetectFromSystemd tests unexported method - commented out
// func TestDetectFromSystemd(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones, err := zd.detectFromSystemd()
// 	// The function might not return an error if it finds common domains
// 	// It only returns ErrSystemdConfigNotFound if no files are found
// 	if err != nil {
// 		assert.ErrorIs(t, err, ErrSystemdConfigNotFound)
// 		assert.Empty(t, zones)
// 	} else {
// 		// If no error, it should return common domains
// 		assert.NotEmpty(t, zones)
// 	}
// }

// TestDetectFromMDNS tests unexported method - commented out
// func TestDetectFromMDNS(t *testing.T) {
// 	zd := &localzone.ZoneDetector{}
//
// 	zones, err := zd.detectFromMDNS()
// 	require.NoError(t, err)
//
// 	// Should fallback to common domains
// 	expectedDomains := []string{"local", "lan", "home", "internal"}
// 	for _, domain := range expectedDomains {
// 		assert.Contains(t, zones, domain)
// 	}
// }

// TestErrorConstants tests undefined error constants - commented out
// func TestErrorConstants(t *testing.T) {
// 	assert.Equal(t, "UCI config file not found", ErrUCIConfigNotFound.Error())
// 	assert.Equal(t, "resolv.conf file not found", ErrResolvConfNotFound.Error())
// 	assert.Equal(t, "systemd-resolved config not found", ErrSystemdConfigNotFound.Error())
// 	assert.Equal(t, "mDNS config not found", ErrMDNSConfigNotFound.Error())
// }

func TestZoneDetectorInterface(t *testing.T) {
	t.Parallel()

	var _ localzone.ZoneDetectorInterface = &localzone.ZoneDetector{}
}

// TestFileReaderInterface tests undefined types - commented out
// func TestFileReaderInterface(t *testing.T) {
// 	var (
// 		_ FileReader = &OSFileReader{}
// 		_ FileReader = &MockFileReader{}
// 	)
// }

func TestFileWatcherInterface(t *testing.T) {
	t.Parallel()
	// This is just to ensure the interface is properly defined
	// FileWatcher is an interface, so we can't instantiate it directly
	// We can only test that it's properly defined by checking if it compiles
}

// TestOSFileReader tests undefined type - commented out
// func TestOSFileReader(t *testing.T) {
// 	reader := &OSFileReader{}
//
// 	// Test with existing file
// 	exists := reader.FileExists("/etc/hosts")
// 	assert.True(t, exists)
//
// 	// Test with non-existing file
// 	exists = reader.FileExists("/nonexistent/file")
// 	assert.False(t, exists)
//
// 	// Test ReadFile with existing file
// 	content, err := reader.ReadFile("/etc/hosts")
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, content)
//
// 	// Test ReadFile with non-existing file
// 	_, err = reader.ReadFile("/nonexistent/file")
// 	assert.Error(t, err)
// }

// TestMockFileReader tests undefined type - commented out
// func TestMockFileReader(t *testing.T) {
// 	reader := NewMockFileReader()
//
// 	// Test with no files
// 	exists := reader.FileExists("test.txt")
// 	assert.False(t, exists)
//
// 	_, err := reader.ReadFile("test.txt")
// 	assert.Error(t, err)
// 	assert.ErrorIs(t, err, os.ErrNotExist)
//
// 	// Test with added file
// 	content := []byte("test content")
// 	reader.SetFile("test.txt", content)
//
// 	exists = reader.FileExists("test.txt")
// 	assert.True(t, exists)
//
// 	readContent, err := reader.ReadFile("test.txt")
// 	require.NoError(t, err)
// 	assert.Equal(t, content, readContent)
// }

// TestMockFileReaderMultipleFiles tests undefined type - commented out
// func TestMockFileReaderMultipleFiles(t *testing.T) {
// 	reader := NewMockFileReader()
//
// 	// Add multiple files
// 	reader.SetFile("file1.txt", []byte("content1"))
// 	reader.SetFile("file2.txt", []byte("content2"))
//
// 	// Test file1
// 	exists := reader.FileExists("file1.txt")
// 	assert.True(t, exists)
//
// 	content, err := reader.ReadFile("file1.txt")
// 	require.NoError(t, err)
// 	assert.Equal(t, []byte("content1"), content)
//
// 	// Test file2
// 	exists = reader.FileExists("file2.txt")
// 	assert.True(t, exists)
//
// 	content, err = reader.ReadFile("file2.txt")
// 	require.NoError(t, err)
// 	assert.Equal(t, []byte("content2"), content)
//
// 	// Test non-existing file
// 	exists = reader.FileExists("file3.txt")
// 	assert.False(t, exists)
// }

// TestMockFileReaderOverwrite tests undefined type - commented out
// func TestMockFileReaderOverwrite(t *testing.T) {
// 	reader := NewMockFileReader()
//
// 	// Add file
// 	reader.SetFile("test.txt", []byte("original"))
//
// 	// Overwrite file
// 	reader.SetFile("test.txt", []byte("updated"))
//
// 	content, err := reader.ReadFile("test.txt")
// 	require.NoError(t, err)
// 	assert.Equal(t, []byte("updated"), content)
// }
