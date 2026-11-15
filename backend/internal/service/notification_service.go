package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"sort"
	"time"
)

const NotificationAnnouncement = "announcement"
const NotificationGift = "gift"

var ErrNotificationNotFound = infraerrors.NotFound("NOTIFICATION_NOT_FOUND", "notification not found")

type GiftNotification struct {
	ID        int64
	Amount    float64
	Source    string
	CreatedAt time.Time
	ReadAt    *time.Time
}
type GiftNotificationRepository interface {
	List(context.Context, int64) ([]GiftNotification, error)
	MarkRead(context.Context, int64, int64) error
}
type UserNotification struct {
	ID         int64      `json:"id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	NotifyMode string     `json:"notify_mode"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartsAt   *time.Time `json:"starts_at,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	ReadAt     *time.Time `json:"read_at"`
	GiftAmount *float64   `json:"gift_amount,omitempty"`
	GiftSource string     `json:"gift_source,omitempty"`
}
type NotificationService struct {
	announcements *AnnouncementService
	gifts         GiftNotificationRepository
}

func NewNotificationService(announcements *AnnouncementService, gifts GiftNotificationRepository) *NotificationService {
	return &NotificationService{announcements: announcements, gifts: gifts}
}
func (s *NotificationService) List(ctx context.Context, userID int64) ([]UserNotification, error) {
	anns, err := s.announcements.ListForUser(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	gifts, err := s.gifts.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]UserNotification, 0, len(anns)+len(gifts))
	for _, a := range anns {
		v := a.Announcement
		out = append(out, UserNotification{ID: v.ID, Type: NotificationAnnouncement, Title: v.Title, Content: v.Content, NotifyMode: v.NotifyMode, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, StartsAt: v.StartsAt, EndsAt: v.EndsAt, ReadAt: a.ReadAt})
	}
	for _, g := range gifts {
		amount := g.Amount
		out = append(out, UserNotification{ID: g.ID, Type: NotificationGift, Title: "Gift credit", Content: "Gift credit is already available in your wallet. Claim acknowledges this notification.", NotifyMode: AnnouncementNotifyModePopup, CreatedAt: g.CreatedAt, UpdatedAt: g.CreatedAt, ReadAt: g.ReadAt, GiftAmount: &amount, GiftSource: g.Source})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].ReadAt == nil) != (out[j].ReadAt == nil) {
			return out[i].ReadAt == nil
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
func (s *NotificationService) MarkRead(ctx context.Context, userID int64, kind string, id int64) error {
	if id <= 0 {
		return ErrNotificationNotFound
	}
	switch kind {
	case NotificationAnnouncement:
		return s.announcements.MarkRead(ctx, userID, id)
	case NotificationGift:
		return s.gifts.MarkRead(ctx, userID, id)
	default:
		return ErrNotificationNotFound
	}
}
