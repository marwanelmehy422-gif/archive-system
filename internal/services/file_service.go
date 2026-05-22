package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"

	"archive-system/internal/database"
	"archive-system/internal/models"
	"archive-system/internal/repository"
)

type FileService struct {
	txRepo    *repository.TransactionRepository
	uploadDir string
	db        *database.DB
}

func NewFileService(txRepo *repository.TransactionRepository, uploadDir string, db *database.DB) *FileService {
	return &FileService{txRepo: txRepo, uploadDir: uploadDir, db: db}
}

// DetectDataType - يتعرف على نوع الداتا من الـ mime type والامتداد
func DetectDataType(mimeStr, filename string) models.DataType {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case strings.HasPrefix(mimeStr, "image/"):
		return models.DataTypeImage

	case mimeStr == "application/pdf":
		return models.DataTypePDF

	case ext == ".geojson" || ext == ".kml" || ext == ".kmz" ||
		ext == ".shp" || ext == ".gpkg" || ext == ".gml":
		return models.DataTypeGeodata

	case mimeStr == "application/json" && (strings.Contains(filename, "geo") || ext == ".geojson"):
		return models.DataTypeGeodata

	case ext == ".db" || ext == ".sqlite" || ext == ".sqlite3":
		return models.DataTypeDatabase

	case ext == ".sql":
		return models.DataTypeDatabase

	case ext == ".csv" || ext == ".xlsx" || ext == ".xls" ||
		mimeStr == "text/csv" ||
		mimeStr == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return models.DataTypeSpreadsheet

	case strings.HasPrefix(mimeStr, "video/"):
		return models.DataTypeVideo

	case strings.HasPrefix(mimeStr, "audio/"):
		return models.DataTypeAudio

	case ext == ".zip" || ext == ".rar" || ext == ".tar" ||
		ext == ".gz" || ext == ".7z":
		return models.DataTypeArchive

	case ext == ".txt" || ext == ".md" || ext == ".log":
		return models.DataTypeText

	case mimeStr == "application/msword" ||
		mimeStr == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return models.DataTypeDocument

	default:
		return models.DataTypeUnknown
	}
}

// UploadFile - رفع ملف وربطه بالـ transaction
func (s *FileService) UploadFile(ctx context.Context, transactionID, orgID string, fileHeader *multipart.FileHeader) (*models.Attachment, error) {
	// جيب الـ transaction عشان تتأكد إنها موجودة والجهة ليها حق
	tx, err := s.txRepo.GetByID(ctx, transactionID)
	if err != nil || tx == nil {
		return nil, ErrTransactionNotFound
	}
	if tx.SenderOrgID != orgID && tx.ReceiverOrgID != orgID {
		return nil, ErrTransactionNotFound
	}

	// افتح الملف
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// اقرأ محتوى الملف
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// تعرف على الـ mime type
	mtype := mimetype.Detect(fileBytes)
	mimeStr := mtype.String()

	// تعرف على نوع الداتا
	dataType := DetectDataType(mimeStr, fileHeader.Filename)

	// احسب الـ checksum
	hash := sha256.Sum256(fileBytes)
	checksum := hex.EncodeToString(hash[:])

	// عمل اسم فريد للملف
	storedFilename := uuid.New().String() + filepath.Ext(fileHeader.Filename)

	// عمل فولدر للـ transaction
	txDir := filepath.Join(s.uploadDir, "transactions", tx.Title)
	if err := os.MkdirAll(txDir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	// احفظ الملف
	storedPath := filepath.Join(txDir, storedFilename)
	if err := os.WriteFile(storedPath, fileBytes, 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	// خزن في الـ DB
	attachment := &models.Attachment{
		TransactionID:    transactionID,
		OriginalFilename: fileHeader.Filename,
		StoredFilename:   storedFilename,
		StoredPath:       storedPath,
		FileSize:         int64(len(fileBytes)),
		MimeType:         mimeStr,
		DataType:         dataType,
		Checksum:         checksum,
	}

	if err := s.saveAttachment(ctx, attachment, orgID); err != nil {
		os.Remove(storedPath)
		return nil, err
	}

	return attachment, nil
}

func (s *FileService) saveAttachment(ctx context.Context, a *models.Attachment, orgID string) error {
	// خزن في transaction_attachments
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO transaction_attachments
			(transaction_id, original_filename, stored_filename, stored_path, file_size, mime_type, data_type, checksum)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at`,
		a.TransactionID, a.OriginalFilename, a.StoredFilename,
		a.StoredPath, a.FileSize, a.MimeType, a.DataType, a.Checksum,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("save attachment: %w", err)
	}

	// خزن في data_inventory
	metadata := buildMetadata(a)
	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO data_inventory
			(attachment_id, transaction_id, org_id, detected_type, mime_type, file_size, original_filename, metadata, detected_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.TransactionID, orgID,
		a.DataType, a.MimeType, a.FileSize, a.OriginalFilename,
		metadata, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save data_inventory: %w", err)
	}

	return nil
}

func buildMetadata(a *models.Attachment) string {
	ext := strings.ToLower(filepath.Ext(a.OriginalFilename))
	sizeMB := float64(a.FileSize) / 1024 / 1024
	return fmt.Sprintf(
		`{"extension":"%s","size_mb":%.2f,"mime_type":"%s","checksum":"%s"}`,
		ext, sizeMB, a.MimeType, a.Checksum,
	)
}

// GetFilePath - جيب مسار الملف للتحميل
func (s *FileService) GetFilePath(ctx context.Context, attachmentID, orgID string) (string, string, error) {
	var storedPath, originalFilename, txSenderOrg, txReceiverOrg string

	err := s.db.Pool.QueryRow(ctx, `
		SELECT ta.stored_path, ta.original_filename,
		       t.sender_org_id, t.receiver_org_id
		FROM transaction_attachments ta
		JOIN transactions t ON ta.transaction_id = t.id
		WHERE ta.id = $1`,
		attachmentID,
	).Scan(&storedPath, &originalFilename, &txSenderOrg, &txReceiverOrg)

	if err != nil {
		return "", "", ErrTransactionNotFound
	}

	if txSenderOrg != orgID && txReceiverOrg != orgID {
		return "", "", ErrNotReceiver
	}

	return storedPath, originalFilename, nil
}
