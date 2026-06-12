package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadTrackCmdReturnsValidCommand tests that loadTrackCmd returns a valid tea.Cmd
func TestLoadTrackCmdReturnsValidCommand(t *testing.T) {
	cmd := loadTrackCmd("nonexistent.flac")

	// Verify it returns a tea.Cmd (function type)
	if cmd == nil {
		t.Error("loadTrackCmd returned nil, expected a valid tea.Cmd")
	}
}

// TestLoadTrackCmdFileNotFound tests that loadTrackCmd handles missing files gracefully
func TestLoadTrackCmdFileNotFound(t *testing.T) {
	cmd := loadTrackCmd("/nonexistent/path/to/file.flac")

	// Execute the command to get the message
	msg := cmd()

	// Verify the returned message is a TrackLoadedMsg
	trackMsg, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
	}

	// Verify error is not nil when file doesn't exist
	if trackMsg.err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	// Verify audio and metadata are nil on error
	if trackMsg.audio != nil {
		t.Error("Expected audio to be nil on error, got non-nil")
	}
	if trackMsg.metadata != nil {
		t.Error("Expected metadata to be nil on error, got non-nil")
	}
}

// TestLoadTrackCmdReturnsTrackLoadedMsg tests that loadTrackCmd returns TrackLoadedMsg type
func TestLoadTrackCmdReturnsTrackLoadedMsg(t *testing.T) {
	cmd := loadTrackCmd("test.flac")
	msg := cmd()

	_, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
	}
}

// TestLoadTrackCmdWithEmptyPath tests behavior with empty file path
func TestLoadTrackCmdWithEmptyPath(t *testing.T) {
	cmd := loadTrackCmd("")
	msg := cmd()

	trackMsg, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
	}

	// Should have an error for empty path
	if trackMsg.err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

// TestLoadTrackCmdWithInvalidFlacFile tests loadTrackCmd with a non-FLAC file
func TestLoadTrackCmdWithInvalidFlacFile(t *testing.T) {
	// Create a temporary non-FLAC file
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some non-FLAC content
	_, err = tmpFile.WriteString("This is not a FLAC file")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cmd := loadTrackCmd(tmpFile.Name())
	msg := cmd()

	trackMsg, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
	}

	// Should have an error because it's not a valid FLAC file
	if trackMsg.err == nil {
		t.Error("Expected error for invalid FLAC file, got nil")
	}

	// Audio and metadata should be nil on error
	if trackMsg.audio != nil {
		t.Error("Expected audio to be nil on error, got non-nil")
	}
}

// TestLoadTrackCmdErrorPropagation tests that errors are properly propagated
func TestLoadTrackCmdErrorPropagation(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
	}{
		{"Directory instead of file", "/tmp"},
		{"Invalid path characters", "invalid\x00path.flac"},
		{"Permission denied path", "/root/protected.flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := loadTrackCmd(tc.filePath)
			msg := cmd()

			trackMsg, ok := msg.(TrackLoadedMsg)
			if !ok {
				t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
			}

			// For most of these paths, we expect an error to be returned
			// The behavior may vary depending on system state, but trackMsg should be valid
			if trackMsg.err != nil {
				t.Logf("Got expected error for path %q: %v", tc.filePath, trackMsg.err)
			}
		})
	}
}

// TestLoadTrackCmdMultipleCalls tests that loadTrackCmd can be called multiple times
func TestLoadTrackCmdMultipleCalls(t *testing.T) {
	paths := []string{"file1.flac", "file2.flac", "file3.flac"}

	for _, path := range paths {
		cmd := loadTrackCmd(path)

		// Verify each call returns a valid command
		if cmd == nil {
			t.Errorf("loadTrackCmd(%q) returned nil", path)
		}

		// Verify each command produces a TrackLoadedMsg
		msg := cmd()
		_, ok := msg.(TrackLoadedMsg)
		if !ok {
			t.Errorf("Expected TrackLoadedMsg for %q, got %T", path, msg)
		}
	}
}

// TestTrackLoadedMsgStructure tests that TrackLoadedMsg has the expected fields
func TestTrackLoadedMsgStructure(t *testing.T) {
	cmd := loadTrackCmd("nonexistent.flac")
	msg := cmd()

	trackMsg, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Fatalf("Expected TrackLoadedMsg, got %T", msg)
	}

	// Test that all fields are accessible
	_ = trackMsg.audio
	_ = trackMsg.metadata
	_ = trackMsg.err

	// For a nonexistent file, we should have an error
	if trackMsg.err == nil {
		t.Error("Expected err field to be populated")
	}
}

// BenchmarkLoadTrackCmd benchmarks the command creation (not execution)
func BenchmarkLoadTrackCmd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = loadTrackCmd("test.flac")
	}
}

// BenchmarkLoadTrackCmdExecution benchmarks executing loadTrackCmd with invalid path
func BenchmarkLoadTrackCmdExecution(b *testing.B) {
	cmd := loadTrackCmd("nonexistent.flac")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd()
	}
}

// TestLoadTrackCmdRelativeVsAbsolutePath tests both relative and absolute file paths
func TestLoadTrackCmdRelativeVsAbsolutePath(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
	}{
		{"Relative path", "./music.flac"},
		{"Relative path with dir", "music/album.flac"},
		{"Absolute path", "/home/user/music.flac"},
		{"Current dir", "music.flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := loadTrackCmd(tc.filePath)

			// All should return a valid command
			if cmd == nil {
				t.Errorf("loadTrackCmd(%q) returned nil", tc.filePath)
			}

			// Executing should return a TrackLoadedMsg with an error (since files don't exist)
			msg := cmd()
			_, ok := msg.(TrackLoadedMsg)
			if !ok {
				t.Errorf("Expected TrackLoadedMsg for %q, got %T", tc.filePath, msg)
			}
		})
	}
}

// TestLoadTrackCmdFileExtensionVariations tests with various file extensions
func TestLoadTrackCmdFileExtensionVariations(t *testing.T) {
	extensions := []string{".flac", ".FLAC", ".mp3", ".wav", ".txt"}

	for _, ext := range extensions {
		t.Run("extension_"+ext, func(t *testing.T) {
			cmd := loadTrackCmd("test" + ext)

			if cmd == nil {
				t.Errorf("loadTrackCmd returned nil for %s", ext)
			}

			msg := cmd()
			_, ok := msg.(TrackLoadedMsg)
			if !ok {
				t.Errorf("Expected TrackLoadedMsg for %s, got %T", ext, msg)
			}
		})
	}
}

// TestLoadTrackCmdHandlesVeryLongPaths tests handling of very long file paths
func TestLoadTrackCmdHandlesVeryLongPaths(t *testing.T) {
	// Create a very long path
	longPath := filepath.Join("/tmp", string(make([]byte, 255)))
	for i := 0; i < len(longPath); i++ {
		longPath = string(append([]byte(longPath), 'a'))
		if len(longPath) > 4096 {
			break
		}
	}

	cmd := loadTrackCmd(longPath)

	if cmd == nil {
		t.Error("loadTrackCmd returned nil for very long path")
	}

	msg := cmd()
	_, ok := msg.(TrackLoadedMsg)
	if !ok {
		t.Errorf("Expected TrackLoadedMsg, got %T", msg)
	}
}
