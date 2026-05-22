package models

import "time"

// ─── Organization ───────────────────────────────────────────

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── User ────────────────────────────────────────────────────

type User struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	FullName       string     `json:"full_name"`
	Role           string     `json:"role"`
	IsActive       bool       `json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ─── Transaction ─────────────────────────────────────────────

type TransactionStatus string
type TransactionPriority string

const (
	StatusPending  TransactionStatus = "pending"
	StatusAccepted TransactionStatus = "accepted"
	StatusRejected TransactionStatus = "rejected"
)

const (
	PriorityNormal TransactionPriority = "normal"
	PriorityHigh   TransactionPriority = "high"
	PriorityUrgent TransactionPriority = "urgent"
)

type Transaction struct {
	ID              string              `json:"id"`
	ReferenceNumber string              `json:"reference_number"`
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	SenderOrgID     string              `json:"sender_org_id"`
	ReceiverOrgID   string              `json:"receiver_org_id"`
	CreatedByUserID string              `json:"created_by_user_id"`
	Status          TransactionStatus   `json:"status"`
	Priority        TransactionPriority `json:"priority"`
	RejectionReason *string             `json:"rejection_reason,omitempty"`
	SentAt          time.Time           `json:"sent_at"`
	RespondedAt     *time.Time          `json:"responded_at,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// TransactionFull - بيانات كاملة للـ dashboard
type TransactionFull struct {
	ID               string              `json:"id"`
	ReferenceNumber  string              `json:"reference_number"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Status           TransactionStatus   `json:"status"`
	Priority         TransactionPriority `json:"priority"`
	RejectionReason  *string             `json:"rejection_reason,omitempty"`
	SentAt           time.Time           `json:"sent_at"`
	RespondedAt      *time.Time          `json:"responded_at,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	SenderOrgID      string              `json:"sender_org_id"`
	SenderOrgName    string              `json:"sender_org_name"`
	SenderOrgCode    string              `json:"sender_org_code"`
	SenderUserID     string              `json:"sender_user_id"`
	SenderUserName   string              `json:"sender_user_name"`
	ReceiverOrgID    string              `json:"receiver_org_id"`
	ReceiverOrgName  string              `json:"receiver_org_name"`
	ReceiverOrgCode  string              `json:"receiver_org_code"`
	AttachmentsCount int                 `json:"attachments_count"`
	AttachmentTypes  []string            `json:"attachment_types"`
}

// ─── Attachment ───────────────────────────────────────────────

type DataType string

const (
	DataTypeImage       DataType = "image"
	DataTypePDF         DataType = "pdf"
	DataTypeDocument    DataType = "document"
	DataTypeSpreadsheet DataType = "spreadsheet"
	DataTypeGeodata     DataType = "geodata"
	DataTypeDatabase    DataType = "database"
	DataTypeVideo       DataType = "video"
	DataTypeAudio       DataType = "audio"
	DataTypeArchive     DataType = "archive"
	DataTypeText        DataType = "text"
	DataTypeUnknown     DataType = "unknown"
)

type Attachment struct {
	ID               string    `json:"id"`
	TransactionID    string    `json:"transaction_id"`
	OriginalFilename string    `json:"original_filename"`
	StoredFilename   string    `json:"stored_filename"`
	StoredPath       string    `json:"stored_path"`
	FileSize         int64     `json:"file_size"`
	MimeType         string    `json:"mime_type"`
	DataType         DataType  `json:"data_type"`
	Checksum         string    `json:"checksum"`
	CreatedAt        time.Time `json:"created_at"`
}

// ─── Notification ─────────────────────────────────────────────

type NotificationType string

const (
	NotifNewTransaction NotificationType = "new_transaction"
	NotifAccepted       NotificationType = "accepted"
	NotifRejected       NotificationType = "rejected"
)

type Notification struct {
	ID            string           `json:"id"`
	UserID        string           `json:"user_id"`
	TransactionID string           `json:"transaction_id"`
	Type          NotificationType `json:"type"`
	Title         string           `json:"title"`
	Body          string           `json:"body"`
	IsRead        bool             `json:"is_read"`
	ReadAt        *time.Time       `json:"read_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}

// ─── Status History ───────────────────────────────────────────

type StatusHistory struct {
	ID              string             `json:"id"`
	TransactionID   string             `json:"transaction_id"`
	ChangedByUserID string             `json:"changed_by_user_id"`
	ChangedByName   string             `json:"changed_by_name,omitempty"`
	OldStatus       *TransactionStatus `json:"old_status,omitempty"`
	NewStatus       TransactionStatus  `json:"new_status"`
	Note            *string            `json:"note,omitempty"`
	ChangedAt       time.Time          `json:"changed_at"`
}

// ─── WebSocket Message ────────────────────────────────────────

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
