package services

import (
	"context"
	"errors"
	"fmt"

	"archive-system/internal/models"
	"archive-system/internal/repository"
)

var (
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrNotReceiver           = errors.New("not the receiver")
	ErrAlreadyResponded      = errors.New("already responded")
	ErrRejectionReasonNeeded = errors.New("rejection reason needed")
)

type TransactionService struct {
	txRepo      *repository.TransactionRepository
	userRepo    *repository.UserRepository
	orgRepo     *repository.OrgRepository
	notifService *NotificationService
}

func NewTransactionService(
	txRepo *repository.TransactionRepository,
	userRepo *repository.UserRepository,
	orgRepo *repository.OrgRepository,
	notifService *NotificationService,
) *TransactionService {
	return &TransactionService{
		txRepo:      txRepo,
		userRepo:    userRepo,
		orgRepo:     orgRepo,
		notifService: notifService,
	}
}

type CreateTransactionRequest struct {
	Title           string `json:"title"            binding:"required"`
	Description     string `json:"description"`
	ReceiverOrgCode string `json:"receiver_org_code" binding:"required"`
	Priority        string `json:"priority"`
}

func (s *TransactionService) Create(ctx context.Context, req *CreateTransactionRequest, senderUserID, senderOrgID string) (*models.Transaction, error) {
	receiverOrg, err := s.orgRepo.GetByCode(ctx, req.ReceiverOrgCode)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	if receiverOrg == nil {
		return nil, ErrOrgNotFound
	}

	priority := models.PriorityNormal
	switch req.Priority {
	case "high":
		priority = models.PriorityHigh
	case "urgent":
		priority = models.PriorityUrgent
	}

	tx := &models.Transaction{
		Title:           req.Title,
		Description:     req.Description,
		SenderOrgID:     senderOrgID,
		ReceiverOrgID:   receiverOrg.ID,
		CreatedByUserID: senderUserID,
		Priority:        priority,
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	// جيب اسم الجهة المرسلة عشان الـ notification
	senderOrg, _ := s.orgRepo.GetByID(ctx, senderOrgID)
	senderOrgName := ""
	if senderOrg != nil {
		senderOrgName = senderOrg.Name
	}

	// ابعت notification للجهة المستقبلة
	go s.notifService.NotifyNewTransaction(context.Background(), tx, senderOrgName)

	return tx, nil
}

func (s *TransactionService) GetAll(ctx context.Context, orgID string) ([]models.TransactionFull, error) {
	txs, err := s.txRepo.GetAllForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []models.TransactionFull{}
	}
	return txs, nil
}

func (s *TransactionService) GetByID(ctx context.Context, id, orgID string) (*models.TransactionFull, []models.Attachment, []models.StatusHistory, error) {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if tx == nil {
		return nil, nil, nil, ErrTransactionNotFound
	}
	if tx.SenderOrgID != orgID && tx.ReceiverOrgID != orgID {
		return nil, nil, nil, ErrTransactionNotFound
	}

	attachments, err := s.txRepo.GetAttachments(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	history, err := s.txRepo.GetStatusHistory(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}

	if attachments == nil {
		attachments = []models.Attachment{}
	}
	if history == nil {
		history = []models.StatusHistory{}
	}

	return tx, attachments, history, nil
}

func (s *TransactionService) Accept(ctx context.Context, id, userOrgID string) error {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tx == nil {
		return ErrTransactionNotFound
	}
	if tx.ReceiverOrgID != userOrgID {
		return ErrNotReceiver
	}
	if tx.Status != models.StatusPending {
		return ErrAlreadyResponded
	}

	if err := s.txRepo.UpdateStatus(ctx, id, models.StatusAccepted, nil); err != nil {
		return err
	}

	// notification للجهة المرسلة
	go s.notifService.NotifyAccepted(context.Background(), tx)

	return nil
}

func (s *TransactionService) Reject(ctx context.Context, id, userOrgID, reason string) error {
	if reason == "" {
		return ErrRejectionReasonNeeded
	}

	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tx == nil {
		return ErrTransactionNotFound
	}
	if tx.ReceiverOrgID != userOrgID {
		return ErrNotReceiver
	}
	if tx.Status != models.StatusPending {
		return ErrAlreadyResponded
	}

	if err := s.txRepo.UpdateStatus(ctx, id, models.StatusRejected, &reason); err != nil {
		return err
	}

	// notification للجهة المرسلة مع السبب
	go s.notifService.NotifyRejected(context.Background(), tx, reason)

	return nil
}