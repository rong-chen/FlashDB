package services

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileService handles file operations
type FileService struct {
	ctx context.Context
}

// NewFileService creates a new FileService
func NewFileService() *FileService {
	return &FileService{}
}

// SetContext sets the Wails context
func (s *FileService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SaveFile opens a save dialog and writes content to the selected file
func (s *FileService) SaveFile(defaultFilename, content, filterPattern, filterName string) Response {
	// Open save dialog
	filename, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{
				DisplayName: filterName,
				Pattern:     filterPattern,
			},
		},
	})

	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if filename == "" {
		return Response{Success: false, Error: "用户取消"}
	}

	// Ensure proper extension
	ext := filepath.Ext(defaultFilename)
	if ext != "" && filepath.Ext(filename) != ext {
		filename += ext
	}

	// Write file
	err = os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: filename}
}
