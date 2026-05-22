package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"archive-system/internal/database"
	"archive-system/internal/models"
)

type TransactionRepository struct {
	db *database.DB
}

func NewTransactionRepository(db *database.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (title, description, sender_org_id, receiver_org_id, created_by_user_id, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, reference_number, status, sent_at, created_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query,
		tx.Title, tx.Description, tx.SenderOrgID,
		tx.ReceiverOrgID, tx.CreatedByUserID, tx.Priority,
	).Scan(&tx.ID, &tx.ReferenceNumber, &tx.Status, &tx.SentAt, &tx.CreatedAt, &tx.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*models.TransactionFull, error) {
	query := `
		SELECT
			t.id, t.reference_number, t.title, t.description,
			t.status, t.priority, t.rejection_reason,
			t.sent_at, t.responded_at, t.created_at,
			s_org.id, s_org.name, s_org.code,
			s_usr.id, s_usr.full_name,
			r_org.id, r_org.name, r_org.code,
			COUNT(DISTINCT ta.id),
			COALESCE(ARRAY_AGG(DISTINCT ta.data_type::text) FILTER (WHERE ta.id IS NOT NULL), '{}')
		FROM transactions t
		JOIN organizations s_org ON t.sender_org_id   = s_org.id
		JOIN organizations r_org ON t.receiver_org_id  = r_org.id
		JOIN users s_usr         ON t.created_by_user_id = s_usr.id
		LEFT JOIN transaction_attachments ta ON ta.transaction_id = t.id
		WHERE t.id = $1
		GROUP BY t.id, s_org.id, s_usr.id, r_org.id`

	tx := &models.TransactionFull{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&tx.ID, &tx.ReferenceNumber, &tx.Title, &tx.Description,
		&tx.Status, &tx.Priority, &tx.RejectionReason,
		&tx.SentAt, &tx.RespondedAt, &tx.CreatedAt,
		&tx.SenderOrgID, &tx.SenderOrgName, &tx.SenderOrgCode,
		&tx.SenderUserID, &tx.SenderUserName,
		&tx.ReceiverOrgID, &tx.ReceiverOrgName, &tx.ReceiverOrgCode,
		&tx.AttachmentsCount, &tx.AttachmentTypes,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepository) GetAllForOrg(ctx context.Context, orgID string) ([]models.TransactionFull, error) {
	query := `
		SELECT
			t.id, t.reference_number, t.title, t.description,
			t.status, t.priority, t.rejection_reason,
			t.sent_at, t.responded_at, t.created_at,
			s_org.id, s_org.name, s_org.code,
			s_usr.id, s_usr.full_name,
			r_org.id, r_org.name, r_org.code,
			COUNT(DISTINCT ta.id),
			COALESCE(ARRAY_AGG(DISTINCT ta.data_type::text) FILTER (WHERE ta.id IS NOT NULL), '{}')
		FROM transactions t
		JOIN organizations s_org ON t.sender_org_id   = s_org.id
		JOIN organizations r_org ON t.receiver_org_id  = r_org.id
		JOIN users s_usr         ON t.created_by_user_id = s_usr.id
		LEFT JOIN transaction_attachments ta ON ta.transaction_id = t.id
		WHERE t.sender_org_id = $1 OR t.receiver_org_id = $1
		GROUP BY t.id, s_org.id, s_usr.id, r_org.id
		ORDER BY t.sent_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("get all transactions: %w", err)
	}
	defer rows.Close()

	var txs []models.TransactionFull
	for rows.Next() {
		var tx models.TransactionFull
		if err := rows.Scan(
			&tx.ID, &tx.ReferenceNumber, &tx.Title, &tx.Description,
			&tx.Status, &tx.Priority, &tx.RejectionReason,
			&tx.SentAt, &tx.RespondedAt, &tx.CreatedAt,
			&tx.SenderOrgID, &tx.SenderOrgName, &tx.SenderOrgCode,
			&tx.SenderUserID, &tx.SenderUserName,
			&tx.ReceiverOrgID, &tx.ReceiverOrgName, &tx.ReceiverOrgCode,
			&tx.AttachmentsCount, &tx.AttachmentTypes,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, id string, status models.TransactionStatus, rejectionReason *string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE transactions SET status = $1, rejection_reason = $2 WHERE id = $3`,
		status, rejectionReason, id,
	)
	return err
}

func (r *TransactionRepository) GetAttachments(ctx context.Context, transactionID string) ([]models.Attachment, error) {
	query := `
		SELECT id, transaction_id, original_filename, stored_filename,
		       stored_path, file_size, mime_type, data_type, checksum, created_at
		FROM transaction_attachments
		WHERE transaction_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Pool.Query(ctx, query, transactionID)
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.Attachment
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(
			&a.ID, &a.TransactionID, &a.OriginalFilename, &a.StoredFilename,
			&a.StoredPath, &a.FileSize, &a.MimeType, &a.DataType, &a.Checksum, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (r *TransactionRepository) GetStatusHistory(ctx context.Context, transactionID string) ([]models.StatusHistory, error) {
	query := `
		SELECT h.id, h.transaction_id, h.changed_by_user_id,
		       u.full_name, h.old_status, h.new_status, h.note, h.changed_at
		FROM transaction_status_history h
		JOIN users u ON h.changed_by_user_id = u.id
		WHERE h.transaction_id = $1
		ORDER BY h.changed_at ASC`

	rows, err := r.db.Pool.Query(ctx, query, transactionID)
	if err != nil {
		return nil, fmt.Errorf("get status history: %w", err)
	}
	defer rows.Close()

	var history []models.StatusHistory
	for rows.Next() {
		var h models.StatusHistory
		if err := rows.Scan(
			&h.ID, &h.TransactionID, &h.ChangedByUserID,
			&h.ChangedByName, &h.OldStatus, &h.NewStatus, &h.Note, &h.ChangedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}
