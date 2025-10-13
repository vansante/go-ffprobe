package ffprobe

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Test constants for side data types
const (
	testStereo3DTypeSideBySide      = "side_by_side"
	testSphericalProjectionEquirect = "equirectangular"
)

// Test asset paths with detailed characteristics
const (

	// testAVIPath - Empty file (0 bytes)
	// Used for: Error testing - invalid/corrupt file handling
	testAVIPath = "assets/test.avi"

	// testMOVPath - H.264 video with display matrix rotation
	// Codec: H.264 (Main), 1280x720, 5.28s duration, 1.0MB
	// Side data: Display Matrix with -180 degree rotation
	// Used for: Testing SideDataDisplayMatrix parsing
	testMOVPath = "assets/test.mov"

	// testMP4Path - Standard H.264 video with chapters
	// Codec: H.264 (Main), 1280x720, 5.31s duration, 1.0MB
	// Contains: Video stream, audio stream (AAC), data stream (chapters)
	// Chapters: 3 chapters ("Beginning", "Middle", "End")
	// Tags: major_brand=isom, language=und
	// Used for: Main test file - streams, format, chapters, tags
	testMP4Path = "assets/test.mp4"

	// testHDRPath - HEVC HDR10 video with mastering display metadata
	// Codec: HEVC Main 10, 3840x2160 (4K), 10.0s duration, 62MB
	// Color: bt2020nc space, smpte2084 transfer (PQ/HDR10), bt2020 primaries
	// Side data: Mastering display metadata with fractional string values
	//   - red_x: "34000/50000" (0.68)
	//   - red_y: "16000/50000" (0.32)
	//   - green_x: "13250/50000" (0.265)
	//   - green_y: "34500/50000" (0.69)
	//   - blue_x: "7500/50000" (0.15)
	//   - blue_y: "3000/50000" (0.06)
	//   - white_point_x: "15635/50000" (0.3127)
	//   - white_point_y: "16450/50000" (0.329)
	//   - min_luminance: "500/10000" (0.05 nits)
	//   - max_luminance: "12000000/10000" (1200 nits)
	// Used for: Testing FlexInt precision with HDR metadata fractions
	// testHDRPath = "assets/test.ts"
)

func Test_FlexInt_UnmarshalJSON_Integer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FlexFloat
	}{
		{
			name:     "positive integer",
			input:    `1000`,
			expected: FlexFloat(1000),
		},
		{
			name:     "zero",
			input:    `0`,
			expected: FlexFloat(0),
		},
		{
			name:     "negative integer",
			input:    `-100`,
			expected: FlexFloat(-100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result FlexFloat
			err := json.Unmarshal([]byte(tt.input), &result)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func Test_FlexInt_UnmarshalJSON_String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FlexFloat
	}{
		{
			name:     "string integer",
			input:    `"1000"`,
			expected: FlexFloat(1000),
		},
		{
			name:     "string zero",
			input:    `"0"`,
			expected: FlexFloat(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result FlexFloat
			err := json.Unmarshal([]byte(tt.input), &result)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func Test_FlexInt_UnmarshalJSON_Fraction(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  FlexFloat
		wantErr   bool
		tolerance float64 // For comparing floating point values
	}{
		{
			name:     "simple fraction - whole number result",
			input:    `"1000/1"`,
			expected: FlexFloat(1000),
		},
		{
			name:      "fraction preserves decimal precision",
			input:     `"34000/50000"`,
			expected:  FlexFloat(0.68),
			tolerance: 0.0001,
		},
		{
			name:      "HDR red_x coordinate",
			input:     `"11408507/16777216"`,
			expected:  FlexFloat(0.68),
			tolerance: 0.01, // More tolerance for complex fractions
		},
		{
			name:      "HDR min_luminance",
			input:     `"500/10000"`,
			expected:  FlexFloat(0.05),
			tolerance: 0.0001,
		},
		{
			name:     "HDR max_luminance - whole number",
			input:    `"12000000/10000"`,
			expected: FlexFloat(1200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result FlexFloat
			err := json.Unmarshal([]byte(tt.input), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}

			// Use tolerance for float comparison if specified
			if tt.tolerance > 0 {
				diff := float64(result) - float64(tt.expected)
				if diff < 0 {
					diff = -diff
				}
				if diff > tt.tolerance {
					t.Errorf("Expected %f (±%f), got %f (diff: %f)", tt.expected, tt.tolerance, result, diff)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			}
		})
	}
}

func Test_SideDataMasteringDisplayMetadata_UnmarshalJSON(t *testing.T) {
	// Test with the exact JSON structure from GitHub issue #55
	jsonData := `{
		"side_data_type": "Mastering display metadata",
		"red_x": "11408507/16777216",
		"red_y": "5368709/16777216",
		"green_x": "2222981/8388608",
		"green_y": "11576279/16777216",
		"blue_x": "5033165/33554432",
		"blue_y": "16106127/268435456",
		"white_point_x": "10492471/33554432",
		"white_point_y": "689963/2097152",
		"min_luminance": "5368709/536870912",
		"max_luminance": "1000/1"
	}`

	var metadata SideDataMasteringDisplayMetadata
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify it unmarshals without error
	if metadata.Type != SideDataTypeMasteringDisplayMetadata {
		t.Errorf("Expected type %s, got %s", SideDataTypeMasteringDisplayMetadata, metadata.Type)
	}

	// Verify decimal precision is now preserved (fix for GitHub issue #55)
	tolerance := 0.01
	expectedRedX := 0.68
	diff := float64(metadata.RedX) - expectedRedX
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("RedX: Expected ~%.2f, got %.4f", expectedRedX, metadata.RedX)
	}

	if metadata.MaxLuminance != 1000 {
		t.Errorf("MaxLuminance: Expected 1000, got %f", metadata.MaxLuminance)
	}
}

func Test_SideDataMasteringDisplayMetadata_LG_HDR_Demo(t *testing.T) {
	// Test with actual LG HDR demo file data
	jsonData := `{
		"side_data_type": "Mastering display metadata",
		"red_x": "34000/50000",
		"red_y": "16000/50000",
		"green_x": "13250/50000",
		"green_y": "34500/50000",
		"blue_x": "7500/50000",
		"blue_y": "3000/50000",
		"white_point_x": "15635/50000",
		"white_point_y": "16450/50000",
		"min_luminance": "500/10000",
		"max_luminance": "12000000/10000"
	}`

	var metadata SideDataMasteringDisplayMetadata
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify decimal precision is preserved (fixed!)
	if metadata.RedX != 0.68 {
		t.Errorf("RedX: Expected 0.68 (34000/50000), got %f", metadata.RedX)
	}
	if metadata.RedY != 0.32 {
		t.Errorf("RedY: Expected 0.32 (16000/50000), got %f", metadata.RedY)
	}
	if metadata.MinLuminance != 0.05 {
		t.Errorf("MinLuminance: Expected 0.05 (500/10000), got %f", metadata.MinLuminance)
	}
	if metadata.MaxLuminance != 1200 {
		t.Errorf("MaxLuminance: Expected 1200 (12000000/10000), got %f", metadata.MaxLuminance)
	}

	// Log all values for verification
	t.Logf("RedX: %f (34000/50000 = 0.68)", metadata.RedX)
	t.Logf("RedY: %f (16000/50000 = 0.32)", metadata.RedY)
	t.Logf("GreenX: %f (13250/50000 = 0.265)", metadata.GreenX)
	t.Logf("GreenY: %f (34500/50000 = 0.69)", metadata.GreenY)
	t.Logf("BlueX: %f (7500/50000 = 0.15)", metadata.BlueX)
	t.Logf("BlueY: %f (3000/50000 = 0.06)", metadata.BlueY)
	t.Logf("WhitePointX: %f (15635/50000 = 0.3127)", metadata.WhitePointX)
	t.Logf("WhitePointY: %f (16450/50000 = 0.329)", metadata.WhitePointY)
	t.Logf("MinLuminance: %f nits (500/10000 = 0.05)", metadata.MinLuminance)
	t.Logf("MaxLuminance: %f nits (12000000/10000 = 1200)", metadata.MaxLuminance)
}

func Test_SideDataMasteringDisplayMetadata_MixedTypes(t *testing.T) {
	// Test with mixed integer and string types
	jsonData := `{
		"side_data_type": "Mastering display metadata",
		"red_x": 1000,
		"red_y": "16000/50000",
		"green_x": "500",
		"max_luminance": "1000/1"
	}`

	var metadata SideDataMasteringDisplayMetadata
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if metadata.RedX != 1000 {
		t.Errorf("RedX: Expected 1000, got %f", metadata.RedX)
	}
	if metadata.RedY != 0.32 {
		t.Errorf("RedY: Expected 0.32 (16000/50000), got %f", metadata.RedY)
	}
	if metadata.GreenX != 500 {
		t.Errorf("GreenX: Expected 500, got %f", metadata.GreenX)
	}
	if metadata.MaxLuminance != 1000 {
		t.Errorf("MaxLuminance: Expected 1000, got %f", metadata.MaxLuminance)
	}
}

func Test_SideDataContentLightLevel(t *testing.T) {
	jsonData := `{
		"side_data_type": "Content light level metadata",
		"max_content": 1000,
		"max_average": 300
	}`

	var metadata SideDataContentLightLevel
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if metadata.MaxContent != 1000 {
		t.Errorf("MaxContent: Expected 1000, got %d", metadata.MaxContent)
	}
	if metadata.MaxAverage != 300 {
		t.Errorf("MaxAverage: Expected 300, got %d", metadata.MaxAverage)
	}
}

func Test_SideDataList_UnmarshalJSON(t *testing.T) {
	jsonData := `[
		{
			"side_data_type": "Content light level metadata",
			"max_content": 1000,
			"max_average": 300
		},
		{
			"side_data_type": "Mastering display metadata",
			"red_x": "34000/50000",
			"max_luminance": "1000/1"
		}
	]`

	var sideDataList SideDataList
	err := json.Unmarshal([]byte(jsonData), &sideDataList)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(sideDataList) != 2 {
		t.Errorf("Expected 2 side data items, got %d", len(sideDataList))
	}

	// Test GetContentLightLevel
	cll, err := sideDataList.GetContentLightLevel()
	if err != nil {
		t.Errorf("Failed to get ContentLightLevel: %v", err)
	}
	if cll.MaxContent != 1000 {
		t.Errorf("MaxContent: Expected 1000, got %d", cll.MaxContent)
	}

	// Test GetMasteringDisplayMetadata
	mdm, err := sideDataList.GetMasteringDisplayMetadata()
	if err != nil {
		t.Errorf("Failed to get MasteringDisplayMetadata: %v", err)
	}
	if mdm.MaxLuminance != 1000 {
		t.Errorf("MaxLuminance: Expected 1000, got %f", mdm.MaxLuminance)
	}
}

func Test_SideDataList_FindSideData(t *testing.T) {
	jsonData := `[
		{
			"side_data_type": "Display Matrix",
			"displaymatrix": "some data",
			"rotation": -180
		}
	]`

	var sideDataList SideDataList
	err := json.Unmarshal([]byte(jsonData), &sideDataList)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Test FindSideData
	data, err := sideDataList.FindSideData(SideDataTypeDisplayMatrix)
	if err != nil {
		t.Errorf("Failed to find DisplayMatrix: %v", err)
	}
	if data == nil {
		t.Error("DisplayMatrix data was nil")
	}

	// Test not found case
	_, err = sideDataList.FindSideData(SideDataTypeStereo3D)
	if !errors.Is(err, ErrSideDataNotFound) {
		t.Errorf("Expected ErrSideDataNotFound, got %v", err)
	}
}

// validateHDRMasteringDisplayMetadata validates the mastering display metadata FlexFloat values
func validateHDRMasteringDisplayMetadata(t *testing.T, mdm *SideDataMasteringDisplayMetadata) {
	// Verify FlexFloat fractional values are correctly parsed
	if mdm.RedX != 0.68 {
		t.Errorf("RedX: Expected 0.68 (34000/50000), got %f", mdm.RedX)
	}
	if mdm.RedY != 0.32 {
		t.Errorf("RedY: Expected 0.32 (16000/50000), got %f", mdm.RedY)
	}
	if mdm.GreenX != 0.265 {
		t.Errorf("GreenX: Expected 0.265 (13250/50000), got %f", mdm.GreenX)
	}
	if mdm.GreenY != 0.69 {
		t.Errorf("GreenY: Expected 0.69 (34500/50000), got %f", mdm.GreenY)
	}
	if mdm.BlueX != 0.15 {
		t.Errorf("BlueX: Expected 0.15 (7500/50000), got %f", mdm.BlueX)
	}
	if mdm.BlueY != 0.06 {
		t.Errorf("BlueY: Expected 0.06 (3000/50000), got %f", mdm.BlueY)
	}
	if mdm.WhitePointX != 0.3127 {
		t.Errorf("WhitePointX: Expected 0.3127 (15635/50000), got %f", mdm.WhitePointX)
	}
	if mdm.WhitePointY != 0.329 {
		t.Errorf("WhitePointY: Expected 0.329 (16450/50000), got %f", mdm.WhitePointY)
	}
	if mdm.MinLuminance != 0.05 {
		t.Errorf("MinLuminance: Expected 0.05 (500/10000), got %f", mdm.MinLuminance)
	}
	if mdm.MaxLuminance != 1200 {
		t.Errorf("MaxLuminance: Expected 1200 (12000000/10000), got %f", mdm.MaxLuminance)
	}
}

// validateHDRContentLightLevel validates the content light level metadata
func validateHDRContentLightLevel(t *testing.T, cll *SideDataContentLightLevel) {
	if cll.MaxContent != 1000 {
		t.Errorf("MaxContent: Expected 1000, got %d", cll.MaxContent)
	}
	if cll.MaxAverage != 300 {
		t.Errorf("MaxAverage: Expected 300, got %d", cll.MaxAverage)
	}
}

// validateHDRVideoProperties validates basic HDR video stream properties
func validateHDRVideoProperties(t *testing.T, videoStream *Stream) {
	// Verify HDR properties
	if videoStream.ColorTransfer != "smpte2084" {
		t.Errorf("Expected smpte2084 (HDR10), got %s", videoStream.ColorTransfer)
	}
	if videoStream.ColorPrimaries != "bt2020" {
		t.Errorf("Expected bt2020 color primaries, got %s", videoStream.ColorPrimaries)
	}
	if videoStream.ColorSpace != "bt2020nc" {
		t.Errorf("Expected bt2020nc color space, got %s", videoStream.ColorSpace)
	}
}

// Test with HDR JSON data (avoids segfault issues with physical file in CI)
func Test_ProbeHDRFile(t *testing.T) {
	// Read the JSON data from test.ts.json
	jsonData, err := os.ReadFile("assets/test.ts.json")
	if err != nil {
		t.Skipf("HDR test JSON not available: %v", err)
	}

	// Parse the JSON into ProbeData structure
	var data ProbeData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatalf("Failed to parse HDR test JSON: %v", err)
	}

	videoStream := data.FirstVideoStream()
	if videoStream == nil {
		t.Fatal("No video stream found in HDR test data")
	}

	validateHDRVideoProperties(t, videoStream)

	// Verify codec and format
	if videoStream.CodecName != "hevc" {
		t.Errorf("Expected hevc codec, got %s", videoStream.CodecName)
	}
	if data.Format.FormatName != "mpegts" {
		t.Errorf("Expected mpegts format, got %s", data.Format.FormatName)
	}

	// *** CRITICAL: Test the HDR metadata parsing with FlexFloat values ***
	// This is the main point of the PR - testing fractional string parsing

	// Test Mastering Display Metadata with FlexFloat parsing
	mdm, err := videoStream.SideDataList.GetMasteringDisplayMetadata()
	if err != nil {
		t.Fatalf("Failed to get MasteringDisplayMetadata: %v", err)
	}
	validateHDRMasteringDisplayMetadata(t, mdm)

	// Test Content Light Level
	cll, err := videoStream.SideDataList.GetContentLightLevel()
	if err != nil {
		t.Fatalf("Failed to get ContentLightLevel: %v", err)
	}
	validateHDRContentLightLevel(t, cll)

	// Log the verified properties
	t.Logf("Video codec: %s", videoStream.CodecName)
	t.Logf("Color transfer: %s", videoStream.ColorTransfer)
	t.Logf("Color primaries: %s", videoStream.ColorPrimaries)
	t.Logf("Color space: %s", videoStream.ColorSpace)
	t.Logf("HDR RedX: %f (parsed from \"34000/50000\")", mdm.RedX)
	t.Logf("HDR MaxLuminance: %f nits (parsed from \"12000000/10000\")", mdm.MaxLuminance)
}

// assetTestCase defines a test case for asset probing
type assetTestCase struct {
	path        string
	shouldError bool
	description string
	useJSON     bool // Indicates if this should be parsed as JSON instead of probed
}

// getAssetTestCases returns the test cases for asset probing
func getAssetTestCases() map[string]assetTestCase {
	return map[string]assetTestCase{
		"test.avi": {
			path:        testAVIPath,
			shouldError: true, // Empty file, should error
			description: "Empty AVI file (error test)",
		},
		"test.mov": {
			path:        testMOVPath,
			shouldError: false,
			description: "MOV with display matrix side data",
		},
		"test.mp4": {
			path:        testMP4Path,
			shouldError: false,
			description: "Standard MP4 with chapters",
		},
		"test.ts": {
			path:        "assets/test.ts.json", // Use JSON instead of physical file
			shouldError: false,
			description: "HDR HEVC transport stream (JSON)",
			useJSON:     true, // Flag to indicate JSON parsing
		},
	}
}

// probeAsset handles probing a single asset file or JSON
func probeAsset(ctx context.Context, asset assetTestCase) (*ProbeData, error) {
	if asset.useJSON {
		// Read and parse JSON data
		jsonData, err := os.ReadFile(asset.path)
		if err != nil {
			return nil, err
		}

		var probeData ProbeData
		if err = json.Unmarshal(jsonData, &probeData); err != nil {
			return nil, err
		}
		return &probeData, nil
	}

	// Use normal ffprobe for other files
	return ProbeURL(ctx, asset.path)
}

// logAssetInfo logs detailed information about a probed asset
func logAssetInfo(t *testing.T, name string, asset assetTestCase, data *ProbeData) {
	t.Logf("%s - %s", name, asset.description)
	t.Logf("  Format: %s", data.Format.FormatName)
	t.Logf("  Duration: %v", data.Format.Duration())
	t.Logf("  Streams: %d", len(data.Streams))

	if len(data.Streams) > 0 {
		videoStreams := data.StreamType(StreamVideo)
		audioStreams := data.StreamType(StreamAudio)
		t.Logf("  Video streams: %d", len(videoStreams))
		t.Logf("  Audio streams: %d", len(audioStreams))

		if len(videoStreams) > 0 {
			vs := videoStreams[0]
			t.Logf("  Video codec: %s", vs.CodecName)
			if vs.ColorSpace != "" {
				t.Logf("  Color space: %s", vs.ColorSpace)
			}
			if vs.ColorTransfer != "" {
				t.Logf("  Color transfer: %s", vs.ColorTransfer)
			}
			if vs.ColorPrimaries != "" {
				t.Logf("  Color primaries: %s", vs.ColorPrimaries)
			}
		}
	}
}

// Test_ProbeAllAssets tests that all asset files can be probed without error
func Test_ProbeAllAssets(t *testing.T) {
	// Skip if ffprobe is not available
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found in PATH")
	}

	assets := getAssetTestCases()
	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	for name, asset := range assets {
		t.Run(name, func(t *testing.T) {
			// Check file exists
			if _, err := os.Stat(asset.path); os.IsNotExist(err) {
				t.Skipf("Asset file not found: %s", asset.path)
			}

			data, err := probeAsset(ctx, asset)

			if asset.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s (%s), but got none", name, asset.description)
				} else {
					t.Logf("Got expected error for %s: %v", name, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s (%s): %v", name, asset.description, err)
				return
			}

			if data == nil {
				t.Errorf("No data returned for %s (%s)", name, asset.description)
				return
			}

			logAssetInfo(t, name, asset, data)
		})
	}
}

func Test_SideDataList_Errors(t *testing.T) {
	var emptyList SideDataList

	// Test all error cases
	tests := []struct {
		name         string
		getterFunc   func() (interface{}, error)
		expectedErr  error
		expectedType string
	}{
		{
			name: "GetDisplayMatrix not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetDisplayMatrix()
			},
			expectedErr: ErrSideDataNotFound,
		},
		{
			name: "GetStereo3D not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetStereo3D()
			},
			expectedErr: ErrSideDataNotFound,
		},
		{
			name: "GetSphericalMapping not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetSphericalMapping()
			},
			expectedErr: ErrSideDataNotFound,
		},
		{
			name: "GetSkipSamples not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetSkipSamples()
			},
			expectedErr: ErrSideDataNotFound,
		},
		{
			name: "GetMasteringDisplayMetadata not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetMasteringDisplayMetadata()
			},
			expectedErr: ErrSideDataNotFound,
		},
		{
			name: "GetContentLightLevel not found",
			getterFunc: func() (interface{}, error) {
				return emptyList.GetContentLightLevel()
			},
			expectedErr: ErrSideDataNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt // Fix gosec G601: avoid implicit memory aliasing in for loop
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.getterFunc()
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// getMarshalTestCases returns test cases for SideData marshaling
func getMarshalTestCases() []struct {
	name     string
	sideData SideData
} {
	return []struct {
		name     string
		sideData SideData
	}{
		{
			name: "DisplayMatrix",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeDisplayMatrix},
				Data: &SideDataDisplayMatrix{
					SideDataBase: SideDataBase{Type: SideDataTypeDisplayMatrix},
					Data:         "test data",
					Rotation:     -180,
				},
			},
		},
		{
			name: "MasteringDisplayMetadata",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeMasteringDisplayMetadata},
				Data: &SideDataMasteringDisplayMetadata{
					SideDataBase: SideDataBase{Type: SideDataTypeMasteringDisplayMetadata},
					RedX:         FlexFloat(0.68),
					RedY:         FlexFloat(0.32),
					MaxLuminance: FlexFloat(1000),
				},
			},
		},
		{
			name: "ContentLightLevel",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeContentLightLevel},
				Data: &SideDataContentLightLevel{
					SideDataBase: SideDataBase{Type: SideDataTypeContentLightLevel},
					MaxContent:   1000,
					MaxAverage:   300,
				},
			},
		},
	}
}

// validateMarshalResult validates the result of marshaling a SideData
func validateMarshalResult(t *testing.T, sideData SideData, data []byte) {
	if len(data) == 0 {
		t.Error("MarshalJSON returned empty data")
		return
	}

	// Verify it can be unmarshaled back
	var result SideData
	err := json.Unmarshal(data, &result)
	if err != nil {
		t.Errorf("Unmarshal of marshaled data failed: %v", err)
		return
	}
	if result.Type != sideData.Type {
		t.Errorf("Type mismatch after marshal/unmarshal: expected %s, got %s", sideData.Type, result.Type)
	}
}

// Test_SideDataMarshalJSON tests the MarshalJSON method for SideData
func Test_SideDataMarshalJSON(t *testing.T) {
	tests := getMarshalTestCases()

	for _, tt := range tests {
		tt := tt // Fix gosec G601: avoid implicit memory aliasing in for loop
		t.Run(tt.name, func(t *testing.T) {
			// Call MarshalJSON directly on the SideData pointer
			data, err := tt.sideData.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON failed: %v", err)
				return
			}
			validateMarshalResult(t, tt.sideData, data)

			// Also test via json.Marshal which should call our custom MarshalJSON
			data2, err := json.Marshal(&tt.sideData)
			if err != nil {
				t.Errorf("json.Marshal failed: %v", err)
				return
			}
			validateMarshalResult(t, tt.sideData, data2)
		})
	}
}

// Test_FindUnknownSideData tests the FindUnknownSideData method
func Test_FindUnknownSideData(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		jsonData := `[
			{
				"side_data_type": "Custom Unknown Type",
				"custom_field": "custom_value"
			}
		]`

		var sideDataList SideDataList
		err := json.Unmarshal([]byte(jsonData), &sideDataList)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		unknown, err := sideDataList.FindUnknownSideData("Custom Unknown Type")
		if err != nil {
			t.Errorf("Expected to find unknown side data, got error: %v", err)
		}
		if unknown == nil {
			t.Error("Expected unknown side data, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		var emptyList SideDataList
		_, err := emptyList.FindUnknownSideData("NonExistent")
		if !errors.Is(err, ErrSideDataNotFound) {
			t.Errorf("Expected ErrSideDataNotFound, got %v", err)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		jsonData := `[
			{
				"side_data_type": "Display Matrix",
				"displaymatrix": "test",
				"rotation": -180
			}
		]`

		var sideDataList SideDataList
		err := json.Unmarshal([]byte(jsonData), &sideDataList)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Try to get DisplayMatrix as Unknown - should fail type assertion
		_, err = sideDataList.FindUnknownSideData(SideDataTypeDisplayMatrix)
		if !errors.Is(err, ErrSideDataUnexpectedType) {
			t.Errorf("Expected ErrSideDataUnexpectedType, got %v", err)
		}
	})
}

// sideDataTypeTestCase defines a test case for side data type unmarshaling
type sideDataTypeTestCase struct {
	name     string
	jsonData string
	typeName string
	validate func(*testing.T, interface{})
}

// getStereo3DTestCase returns the Stereo3D test case
func getStereo3DTestCase() sideDataTypeTestCase {
	return sideDataTypeTestCase{
		name: "Stereo3D",
		jsonData: `{
			"side_data_type": "Stereo 3D",
			"type": "side_by_side",
			"inverted": true
		}`,
		typeName: SideDataTypeStereo3D,
		validate: func(t *testing.T, data interface{}) {
			stereo3D, ok := data.(*SideDataStereo3D)
			if !ok {
				t.Error("Failed to cast to SideDataStereo3D")
				return
			}
			if stereo3D.Type != testStereo3DTypeSideBySide {
				t.Errorf("Expected type %q, got %s", testStereo3DTypeSideBySide, stereo3D.Type)
			}
			if !stereo3D.Inverted {
				t.Error("Expected inverted to be true")
			}
		},
	}
}

// getSphericalMappingTestCase returns the SphericalMapping test case
func getSphericalMappingTestCase() sideDataTypeTestCase {
	return sideDataTypeTestCase{
		name: "SphericalMapping",
		jsonData: `{
			"side_data_type": "Spherical Mapping",
			"projection": "equirectangular",
			"yaw": 90,
			"pitch": 45,
			"roll": 30
		}`,
		typeName: SideDataTypeSphericalMapping,
		validate: func(t *testing.T, data interface{}) {
			spherical, ok := data.(*SideDataSphericalMapping)
			if !ok {
				t.Error("Failed to cast to SideDataSphericalMapping")
				return
			}
			if spherical.Projection != testSphericalProjectionEquirect {
				t.Errorf("Expected projection %q, got %s", testSphericalProjectionEquirect, spherical.Projection)
			}
			if spherical.Yaw != 90 {
				t.Errorf("Expected yaw 90, got %d", spherical.Yaw)
			}
		},
	}
}

// getSkipSamplesTestCase returns the SkipSamples test case
func getSkipSamplesTestCase() sideDataTypeTestCase {
	return sideDataTypeTestCase{
		name: "SkipSamples",
		jsonData: `{
			"side_data_type": "Skip Samples",
			"skip_samples": 100,
			"discard_padding": 50,
			"skip_reason": 1,
			"discard_reason": 2
		}`,
		typeName: SideDataTypeSkipSamples,
		validate: func(t *testing.T, data interface{}) {
			skipSamples, ok := data.(*SideDataSkipSamples)
			if !ok {
				t.Error("Failed to cast to SideDataSkipSamples")
				return
			}
			if skipSamples.SkipSamples != 100 {
				t.Errorf("Expected skip_samples 100, got %d", skipSamples.SkipSamples)
			}
			if skipSamples.DiscardPadding != 50 {
				t.Errorf("Expected discard_padding 50, got %d", skipSamples.DiscardPadding)
			}
		},
	}
}

// getUnknownTypeTestCase returns the Unknown Type test case
func getUnknownTypeTestCase() sideDataTypeTestCase {
	return sideDataTypeTestCase{
		name: "Unknown Type - Default Case",
		jsonData: `{
			"side_data_type": "Custom Unknown Type",
			"custom_field": "custom_value",
			"another_field": 123
		}`,
		typeName: "Custom Unknown Type",
		validate: func(t *testing.T, data interface{}) {
			unknown, ok := data.(*SideDataUnknown)
			if !ok {
				t.Error("Failed to cast to SideDataUnknown")
				return
			}
			// SideDataUnknown is just a Tags type, which is a map
			if unknown == nil {
				t.Error("Unknown side data is nil")
			}
		},
	}
}

// getSideDataTypeTestCases returns test cases for all side data types
func getSideDataTypeTestCases() []sideDataTypeTestCase {
	return []sideDataTypeTestCase{
		getStereo3DTestCase(),
		getSphericalMappingTestCase(),
		getSkipSamplesTestCase(),
		getUnknownTypeTestCase(),
	}
}

// testSideDataTypeUnmarshaling tests unmarshaling of a single side data type
func testSideDataTypeUnmarshaling(t *testing.T, testCase sideDataTypeTestCase) {
	var sd SideData
	err := json.Unmarshal([]byte(testCase.jsonData), &sd)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if sd.Type != testCase.typeName {
		t.Errorf("Expected type %s, got %s", testCase.typeName, sd.Type)
	}

	if testCase.validate != nil {
		testCase.validate(t, sd.Data)
	}
}

// Test_SideData_AllTypes tests unmarshaling all side data types
func Test_SideData_AllTypes(t *testing.T) {
	tests := getSideDataTypeTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSideDataTypeUnmarshaling(t, tt)
		})
	}
}

// Test_SideData_UnmarshalJSON_Error tests error handling in SideData unmarshaling
func Test_SideData_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "invalid JSON - malformed",
			jsonData: `{invalid json`,
			wantErr:  true,
		},
		{
			name:     "invalid JSON - not an object",
			jsonData: `["array"]`,
			wantErr:  true,
		},
		{
			name:     "valid JSON but invalid data unmarshal",
			jsonData: `{"side_data_type": "Display Matrix", "rotation": "not a number"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sd SideData
			err := json.Unmarshal([]byte(tt.jsonData), &sd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// Test_FlexFloat_InvalidJSON tests error handling in FlexFloat
func Test_FlexFloat_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FlexFloat
		wantErr  bool
	}{
		{
			name:    "invalid fraction - division by zero",
			input:   `"100/0"`,
			wantErr: true, // Should error: cannot divide by zero
		},
		{
			name:    "invalid fraction - non-numeric numerator",
			input:   `"abc/100"`,
			wantErr: true, // Should error: cannot parse numerator
		},
		{
			name:    "invalid fraction - non-numeric denominator",
			input:   `"100/def"`,
			wantErr: true, // Should error: cannot parse denominator
		},
		{
			name:    "invalid fraction - malformed",
			input:   `"100/200/300"`,
			wantErr: true, // Should error: malformed fraction
		},
		{
			name:    "invalid string - not a number",
			input:   `"not-a-number"`,
			wantErr: true, // Should error: not a valid number
		},
		{
			name:    "malformed JSON - should error",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "malformed JSON - array",
			input:   `["array"]`,
			wantErr: true,
		},
		{
			name:    "malformed JSON - object",
			input:   `{"key": "value"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result FlexFloat
			err := json.Unmarshal([]byte(tt.input), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got error: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("Expected %f for invalid input, got %f", tt.expected, result)
			}
		})
	}
}

// Test_SideDataList_UnmarshalJSON_Error tests error handling in SideDataList unmarshaling
func Test_SideDataList_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "invalid JSON - malformed",
			jsonData: `[{invalid json}]`,
			wantErr:  true,
		},
		{
			name:     "not an array",
			jsonData: `{"key": "value"}`,
			wantErr:  true,
		},
		{
			name:     "array with invalid side data object",
			jsonData: `[{"side_data_type": "Display Matrix", "rotation": "not-a-number"}]`,
			wantErr:  true,
		},
		{
			name:     "array with malformed JSON element",
			jsonData: `[{broken]`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var list SideDataList
			err := json.Unmarshal([]byte(tt.jsonData), &list)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// getAllGettersTestData returns comprehensive test data for all getter methods
func getAllGettersTestData() string {
	return `[
		{
			"side_data_type": "Display Matrix",
			"displaymatrix": "test matrix data",
			"rotation": -90
		},
		{
			"side_data_type": "Stereo 3D",
			"type": "` + testStereo3DTypeSideBySide + `",
			"inverted": false
		},
		{
			"side_data_type": "Spherical Mapping",
			"projection": "` + testSphericalProjectionEquirect + `",
			"yaw": 180,
			"pitch": 90,
			"roll": 45
		},
		{
			"side_data_type": "Skip Samples",
			"skip_samples": 256,
			"discard_padding": 128,
			"skip_reason": 1,
			"discard_reason": 2
		},
		{
			"side_data_type": "Mastering display metadata",
			"red_x": "34000/50000",
			"max_luminance": "1000/1"
		},
		{
			"side_data_type": "Content light level metadata",
			"max_content": 2000,
			"max_average": 500
		}
	]`
}

// testAllGetterMethods tests all getter methods with provided side data list
func testAllGetterMethods(t *testing.T, sideDataList SideDataList) {
	// Test GetDisplayMatrix
	t.Run("GetDisplayMatrix", func(t *testing.T) {
		dm, err := sideDataList.GetDisplayMatrix()
		if err != nil {
			t.Errorf("Failed to get DisplayMatrix: %v", err)
		}
		if dm == nil {
			t.Fatal("DisplayMatrix is nil")
		}
		if dm.Rotation != -90 {
			t.Errorf("Expected rotation -90, got %d", dm.Rotation)
		}
		if dm.Data != "test matrix data" {
			t.Errorf("Expected data 'test matrix data', got %s", dm.Data)
		}
	})

	// Test GetStereo3D
	t.Run("GetStereo3D", func(t *testing.T) {
		s3d, err := sideDataList.GetStereo3D()
		if err != nil {
			t.Errorf("Failed to get Stereo3D: %v", err)
		}
		if s3d == nil {
			t.Fatal("Stereo3D is nil")
		}
		if s3d.Type != testStereo3DTypeSideBySide {
			t.Errorf("Expected type %q, got %s", testStereo3DTypeSideBySide, s3d.Type)
		}
		if s3d.Inverted {
			t.Error("Expected inverted to be false")
		}
	})

	// Test GetSphericalMapping
	t.Run("GetSphericalMapping", func(t *testing.T) {
		sm, err := sideDataList.GetSphericalMapping()
		if err != nil {
			t.Errorf("Failed to get SphericalMapping: %v", err)
		}
		if sm == nil {
			t.Fatal("SphericalMapping is nil")
		}
		if sm.Projection != testSphericalProjectionEquirect {
			t.Errorf("Expected projection %q, got %s", testSphericalProjectionEquirect, sm.Projection)
		}
		if sm.Yaw != 180 {
			t.Errorf("Expected yaw 180, got %d", sm.Yaw)
		}
		if sm.Pitch != 90 {
			t.Errorf("Expected pitch 90, got %d", sm.Pitch)
		}
		if sm.Roll != 45 {
			t.Errorf("Expected roll 45, got %d", sm.Roll)
		}
	})
}

// testRemainingGetterMethods tests the remaining getter methods
func testRemainingGetterMethods(t *testing.T, sideDataList SideDataList) {
	// Test GetSkipSamples
	t.Run("GetSkipSamples", func(t *testing.T) {
		ss, err := sideDataList.GetSkipSamples()
		if err != nil {
			t.Errorf("Failed to get SkipSamples: %v", err)
		}
		if ss == nil {
			t.Fatal("SkipSamples is nil")
		}
		if ss.SkipSamples != 256 {
			t.Errorf("Expected skip_samples 256, got %d", ss.SkipSamples)
		}
		if ss.DiscardPadding != 128 {
			t.Errorf("Expected discard_padding 128, got %d", ss.DiscardPadding)
		}
		if ss.SkipReason != 1 {
			t.Errorf("Expected skip_reason 1, got %d", ss.SkipReason)
		}
		if ss.DiscardReason != 2 {
			t.Errorf("Expected discard_reason 2, got %d", ss.DiscardReason)
		}
	})

	// Test GetMasteringDisplayMetadata
	t.Run("GetMasteringDisplayMetadata", func(t *testing.T) {
		mdm, err := sideDataList.GetMasteringDisplayMetadata()
		if err != nil {
			t.Errorf("Failed to get MasteringDisplayMetadata: %v", err)
		}
		if mdm == nil {
			t.Fatal("MasteringDisplayMetadata is nil")
		}
		if mdm.RedX != 0.68 {
			t.Errorf("Expected red_x 0.68, got %f", mdm.RedX)
		}
		if mdm.MaxLuminance != 1000 {
			t.Errorf("Expected max_luminance 1000, got %f", mdm.MaxLuminance)
		}
	})

	// Test GetContentLightLevel
	t.Run("GetContentLightLevel", func(t *testing.T) {
		cll, err := sideDataList.GetContentLightLevel()
		if err != nil {
			t.Errorf("Failed to get ContentLightLevel: %v", err)
		}
		if cll == nil {
			t.Fatal("ContentLightLevel is nil")
		}
		if cll.MaxContent != 2000 {
			t.Errorf("Expected max_content 2000, got %d", cll.MaxContent)
		}
		if cll.MaxAverage != 500 {
			t.Errorf("Expected max_average 500, got %d", cll.MaxAverage)
		}
	})
}

// Test_SideDataList_AllGetters tests all getter methods with actual data
//
//nolint:gocyclo // Test function with multiple subtests
func Test_SideDataList_AllGetters(t *testing.T) {
	jsonData := getAllGettersTestData()

	var sideDataList SideDataList
	err := json.Unmarshal([]byte(jsonData), &sideDataList)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	testAllGetterMethods(t, sideDataList)
	testRemainingGetterMethods(t, sideDataList)
}

// getBasicMismatchCases returns the first set of type mismatch test cases
func getBasicMismatchCases() []struct {
	name        string
	sideData    SideData
	getterFunc  func(SideDataList) (interface{}, error)
	expectedErr error
} {
	return []struct {
		name        string
		sideData    SideData
		getterFunc  func(SideDataList) (interface{}, error)
		expectedErr error
	}{
		{
			name: "GetDisplayMatrix with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeDisplayMatrix},
				Data:         &SideDataStereo3D{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetDisplayMatrix()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
		{
			name: "GetStereo3D with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeStereo3D},
				Data:         &SideDataDisplayMatrix{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetStereo3D()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
		{
			name: "GetSphericalMapping with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeSphericalMapping},
				Data:         &SideDataDisplayMatrix{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetSphericalMapping()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
	}
}

// getAdvancedMismatchCases returns the remaining type mismatch test cases
func getAdvancedMismatchCases() []struct {
	name        string
	sideData    SideData
	getterFunc  func(SideDataList) (interface{}, error)
	expectedErr error
} {
	return []struct {
		name        string
		sideData    SideData
		getterFunc  func(SideDataList) (interface{}, error)
		expectedErr error
	}{
		{
			name: "GetSkipSamples with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeSkipSamples},
				Data:         &SideDataDisplayMatrix{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetSkipSamples()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
		{
			name: "GetMasteringDisplayMetadata with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeMasteringDisplayMetadata},
				Data:         &SideDataDisplayMatrix{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetMasteringDisplayMetadata()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
		{
			name: "GetContentLightLevel with wrong data type",
			sideData: SideData{
				SideDataBase: SideDataBase{Type: SideDataTypeContentLightLevel},
				Data:         &SideDataDisplayMatrix{}, // Wrong type!
			},
			getterFunc: func(list SideDataList) (interface{}, error) {
				return list.GetContentLightLevel()
			},
			expectedErr: ErrSideDataUnexpectedType,
		},
	}
}

// getTypeMismatchTestCases returns test cases for type mismatch errors
func getTypeMismatchTestCases() []struct {
	name        string
	sideData    SideData
	getterFunc  func(SideDataList) (interface{}, error)
	expectedErr error
} {
	basicCases := getBasicMismatchCases()
	advancedCases := getAdvancedMismatchCases()
	return append(basicCases, advancedCases...)
}

// Test_SideDataList_TypeMismatch tests type assertion errors by manually constructing invalid data
func Test_SideDataList_TypeMismatch(t *testing.T) {
	// Manually construct a SideDataList with wrong data types
	// This is the only way to trigger the type assertion errors in the getters

	tests := getTypeMismatchTestCases()

	for _, tt := range tests {
		tt := tt // Fix gosec G601: avoid implicit memory aliasing in for loop
		t.Run(tt.name, func(t *testing.T) {
			list := SideDataList{tt.sideData}
			_, err := tt.getterFunc(list)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}
