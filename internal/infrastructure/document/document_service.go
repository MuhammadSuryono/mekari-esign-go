package document

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"mekari-esign/internal/config"
)

// DocumentService handles document file operations
type DocumentService interface {
	// FindDocumentByInvoiceNumber finds a document in the ready folder by invoice number
	// Returns the base64 encoded content and filename
	FindDocumentByInvoiceNumber(invoiceNumber string) (base64Content string, filename string, err error)

	// FindFilenameInProgress finds a document filename in the progress folder by invoice number
	FindFilenameInProgress(invoiceNumber string) (filename string, err error)

	// MoveToProgress moves a document from ready to progress folder
	MoveToProgress(filename string) error

	// MoveToFinish moves a document from progress to finish folder
	MoveToFinish(filename string) error

	// ReplaceFileInProgress replaces a file in progress folder with new content
	ReplaceFileInProgress(filename string, content []byte) error

	// SaveToFinishAndDeleteProgress saves content to finish folder and deletes from progress
	SaveToFinishAndDeleteProgress(filename string, content []byte) error

	// SaveToReadyAndDeleteProgress saves content to ready folder and deletes from progress
	SaveToReadyAndDeleteProgress(filename string, content []byte) error

	// GetReadyPath returns the full path to ready folder
	GetReadyPath() string

	// GetProgressPath returns the full path to progress folder
	GetProgressPath() string

	// GetFinishPath returns the full path to finish folder
	GetFinishPath() string
}

type documentService struct {
	config *config.DocumentConfig
	logger *zap.Logger
}

func NewDocumentService(cfg *config.Config, logger *zap.Logger) (DocumentService, error) {
	svc := &documentService{
		config: &cfg.Document,
		logger: logger,
	}

	// Ensure all directories exist
	if err := svc.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create document directories: %w", err)
	}

	logger.Info("Document service initialized",
		zap.String("base_path", cfg.Document.BasePath),
		zap.String("ready_folder", svc.GetReadyPath()),
		zap.String("progress_folder", svc.GetProgressPath()),
		zap.String("finish_folder", svc.GetFinishPath()),
	)

	return svc, nil
}

func (s *documentService) ensureDirectories() error {
	dirs := []string{
		s.GetReadyPath(),
		s.GetProgressPath(),
		s.GetFinishPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (s *documentService) GetReadyPath() string {
	return filepath.Join(s.config.BasePath, s.config.ReadyFolder)
}

func (s *documentService) GetProgressPath() string {
	return filepath.Join(s.config.BasePath, s.config.ProgressFolder)
}

func (s *documentService) GetFinishPath() string {
	return filepath.Join(s.config.BasePath, s.config.FinishFolder)
}

func (s *documentService) FindDocumentByInvoiceNumber(invoiceNumber string) (string, string, error) {
	readyPath := s.GetReadyPath()

	s.logger.Info("Searching for document",
		zap.String("invoice_number", invoiceNumber),
		zap.String("ready_path", readyPath),
	)

	// List files in ready folder
	files, err := os.ReadDir(readyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read ready folder: %w", err)
	}

	// Search for file matching invoice number pattern
	// Pattern: {prefix}{invoice_number}*.pdf or {invoice_number}*.pdf
	extension := s.config.FileExtension
	if extension == "" {
		extension = ".pdf"
	}

	var matchedFile string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Check if filename matches the pattern
		// Format: invoicenumber_xxxx.pdf or prefix_invoicenumber_xxxx.pdf
		if !strings.HasSuffix(strings.ToLower(filename), strings.ToLower(extension)) {
			continue
		}

		// Check if invoice number is in the filename
		if strings.Contains(filename, invoiceNumber) {
			matchedFile = filename
			s.logger.Info("Found matching document",
				zap.String("invoice_number", invoiceNumber),
				zap.String("filename", filename),
			)
			break
		}
	}

	if matchedFile == "" {
		return "", "", fmt.Errorf("document not found for invoice number: %s", invoiceNumber)
	}

	// Read file content
	filePath := filepath.Join(readyPath, matchedFile)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read document file: %w", err)
	}

	// Encode to base64
	base64Content := base64.StdEncoding.EncodeToString(content)

	s.logger.Info("Document loaded successfully",
		zap.String("filename", matchedFile),
		zap.Int("size_bytes", len(content)),
		zap.Int("base64_length", len(base64Content)),
	)

	return base64Content, matchedFile, nil
}

func (s *documentService) MoveToProgress(filename string) error {
	srcPath := filepath.Join(s.GetReadyPath(), filename)
	dstPath := filepath.Join(s.GetProgressPath(), filename)

	s.logger.Info("Moving document to progress",
		zap.String("filename", filename),
		zap.String("from", srcPath),
		zap.String("to", dstPath),
	)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move document to progress: %w", err)
	}

	s.logger.Info("Document moved to progress successfully",
		zap.String("filename", filename),
	)

	return nil
}

func (s *documentService) MoveToFinish(filename string) error {
	srcPath := filepath.Join(s.GetProgressPath(), filename)
	dstPath := filepath.Join(s.GetFinishPath(), filename)

	s.logger.Info("Moving document to finish",
		zap.String("filename", filename),
		zap.String("from", srcPath),
		zap.String("to", dstPath),
	)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move document to finish: %w", err)
	}

	s.logger.Info("Document moved to finish successfully",
		zap.String("filename", filename),
	)

	return nil
}

func (s *documentService) FindFilenameInProgress(invoiceNumber string) (string, error) {
	progressPath := s.GetProgressPath()

	s.logger.Info("Searching for document in progress",
		zap.String("invoice_number", invoiceNumber),
		zap.String("progress_path", progressPath),
	)

	// List files in progress folder
	files, err := os.ReadDir(progressPath)
	if err != nil {
		return "", fmt.Errorf("failed to read progress folder: %w", err)
	}

	extension := s.config.FileExtension
	if extension == "" {
		extension = ".pdf"
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		if !strings.HasSuffix(strings.ToLower(filename), strings.ToLower(extension)) {
			continue
		}

		if strings.Contains(filename, invoiceNumber) {
			s.logger.Info("Found matching document in progress",
				zap.String("invoice_number", invoiceNumber),
				zap.String("filename", filename),
			)
			return filename, nil
		}
	}

	return "", fmt.Errorf("document not found in progress for invoice number: %s", invoiceNumber)
}

func (s *documentService) ReplaceFileInProgress(filename string, content []byte) error {
	filePath := filepath.Join(s.GetProgressPath(), filename)

	s.logger.Info("Replacing file in progress",
		zap.String("filename", filename),
		zap.String("path", filePath),
		zap.Int("new_size_bytes", len(content)),
	)

	// Write new content to file (overwrites existing)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to replace file in progress: %w", err)
	}

	s.logger.Info("File replaced successfully in progress",
		zap.String("filename", filename),
		zap.Int("size_bytes", len(content)),
	)

	return nil
}

func (s *documentService) SaveToFinishAndDeleteProgress(filename string, content []byte) error {
	progressPath := filepath.Join(s.GetProgressPath(), filename)
	finishPath := filepath.Join(s.GetFinishPath(), filename)

	s.logger.Info("Saving file to finish and deleting from progress",
		zap.String("filename", filename),
		zap.String("progress_path", progressPath),
		zap.String("finish_path", finishPath),
		zap.Int("size_bytes", len(content)),
	)

	// Write content to finish folder
	if err := os.WriteFile(finishPath, content, 0644); err != nil {
		return fmt.Errorf("failed to save file to finish folder: %w", err)
	}

	s.logger.Info("File saved to finish folder",
		zap.String("filename", filename),
		zap.Int("size_bytes", len(content)),
	)

	// Delete file from progress folder
	if err := os.Remove(progressPath); err != nil {
		// Log warning but don't fail - file might not exist
		s.logger.Warn("Failed to delete file from progress folder",
			zap.String("filename", filename),
			zap.Error(err),
		)
	} else {
		s.logger.Info("File deleted from progress folder",
			zap.String("filename", filename),
		)
	}

	return nil
}

func (s *documentService) SaveToReadyAndDeleteProgress(filename string, content []byte) error {
	progressPath := filepath.Join(s.GetProgressPath(), filename)
	readyPath := filepath.Join(s.GetReadyPath(), filename)

	s.logger.Info("Saving file to ready and deleting from progress",
		zap.String("filename", filename),
		zap.String("progress_path", progressPath),
		zap.String("ready_path", readyPath),
		zap.Int("size_bytes", len(content)),
	)

	// Write content to ready folder
	if err := os.WriteFile(readyPath, content, 0644); err != nil {
		return fmt.Errorf("failed to save file to ready folder: %w", err)
	}

	s.logger.Info("File saved to ready folder",
		zap.String("filename", filename),
		zap.Int("size_bytes", len(content)),
	)

	// Delete file from progress folder
	if err := os.Remove(progressPath); err != nil {
		// Log warning but don't fail - file might not exist
		s.logger.Warn("Failed to delete file from progress folder",
			zap.String("filename", filename),
			zap.Error(err),
		)
	} else {
		s.logger.Info("File deleted from progress folder",
			zap.String("filename", filename),
		)
	}

	return nil
}
