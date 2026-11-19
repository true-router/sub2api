package service

import (
	"context"

	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/vmusers"
)

var vmUsername = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{1,31}$`)

type VMUserPreview struct {
	Email    string            `json:"email"`
	UserID   int64             `json:"user_id"`
	Username string            `json:"username"`
	Remote   *vmusers.Identity `json:"remote"`
	Bound    bool              `json:"bound"`
}
type vmBinding struct {
	remote, origin     string
	groupID, accountID int64
}

func findVMBinding(ctx context.Context, c *dbent.Client, id int64) (b vmBinding, err error) {
	rows, err := c.QueryContext(ctx, "SELECT remote_user_id, origin, group_id, account_id FROM vm2api_subscription_bindings WHERE user_id=$1", id)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	if !rows.Next() {
		return b, rows.Err()
	}
	err = rows.Scan(&b.remote, &b.origin, &b.groupID, &b.accountID)
	return
}
func (s *SubscriptionService) vmPreview(ctx context.Context, c *dbent.Client, client *vmusers.Client, id int64, lock bool) (*VMUserPreview, vmBinding, error) {
	var b vmBinding
	q := c.User.Query().Where(user.IDEQ(id), user.DeletedAtIsNil(), user.RoleEQ(RoleUser), user.StatusEQ(StatusActive))
	if lock {
		q = q.ForUpdate()
	}
	u, err := q.Only(ctx)
	if err != nil {
		return nil, b, err
	}
	b, err = findVMBinding(ctx, c, id)
	if err != nil {
		return nil, b, err
	}
	name := strings.TrimSpace(u.Username)
	if !vmUsername.MatchString(name) {
		name = ""
	}
	if b.remote == "" && name != "" {
		rows, e := c.QueryContext(ctx, "SELECT 1 FROM users WHERE id<>$1 AND deleted_at IS NULL AND lower(trim(username))=lower($2) LIMIT 1", id, name)
		if e != nil {
			return nil, b, e
		}
		defer rows.Close()
		if rows.Next() {
			return nil, b, infraerrors.BadRequest("VM_SUBSCRIPTION_INVALID", "sub2api 用户名不唯一")
		}
		if e = rows.Err(); e != nil {
			return nil, b, e
		}
	} else if b.remote != "" && b.origin != client.Origin {
		return nil, b, infraerrors.BadRequest("VM_SUBSCRIPTION_INVALID", "vm2api 实例与已绑定实例不符")
	}
	remote, err := client.Lookup(ctx, b.remote, name, u.Email)
	if err != nil {
		return nil, b, err
	}
	if remote != nil && b.remote == "" {
		rows, e := c.QueryContext(ctx, "SELECT 1 FROM vm2api_subscription_bindings WHERE remote_user_id=$1", remote.ID)
		if e != nil {
			return nil, b, e
		}
		defer rows.Close()
		if rows.Next() {
			return nil, b, infraerrors.BadRequest("VM_SUBSCRIPTION_INVALID", "vm2api 用户已绑定其他用户")
		}
		if e = rows.Err(); e != nil {
			return nil, b, e
		}
	}
	return &VMUserPreview{u.Email, id, name, remote, b.remote != ""}, b, nil
}
func (s *SubscriptionService) PreviewVMUser(ctx context.Context, id int64) (*VMUserPreview, error) {
	client, err := vmusers.Configured()
	if err != nil {
		return nil, err
	}
	p, _, err := s.vmPreview(ctx, s.entClient, client, id, false)
	return p, err
}

func (s *SubscriptionService) AssignVMUser(ctx context.Context, id, admin int64, days int, name, email, confirmed string) (*UserSubscription, error) {
	if days < 1 || days > MaxValidityDays {
		return nil, infraerrors.BadRequest("VM_SUBSCRIPTION_INVALID", "订阅有效期无效")
	}
	client, err := vmusers.Configured()
	if err != nil {
		return nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	c := tx.Client()
	p, b, err := s.vmPreview(ctx, c, client, id, true)
	if err != nil {
		return nil, err
	}
	expected := ""
	if p.Remote != nil {
		expected = p.Remote.ID
	}
	if email != p.Email || name != p.Username || (confirmed != expected && !(confirmed == "" && p.Bound)) {
		return nil, infraerrors.Conflict("VM_IDENTITY_CHANGED", "用户身份已变更，请重新核对")
	}
	remoteName := name
	if p.Remote != nil {
		remoteName = p.Remote.Username
	}
	txctx := dbent.NewTxContext(ctx, tx)
	var existing *UserSubscription
	if b.groupID != 0 {
		existing, err = s.userSubRepo.GetByUserIDAndGroupID(txctx, id, b.groupID)
		if err != nil {
			return nil, err
		}
		if !existing.IsActive() {
			return nil, infraerrors.Conflict("VM_SUBSCRIPTION_INACTIVE", "原订阅已失效，请在订阅管理中恢复或续期")
		}
	}
	if p.Bound && p.Remote.Email != "" {
		email = p.Remote.Email
	}
	issued, err := client.Enable(ctx, p.Remote, remoteName, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		a, e := c.Account.Get(ctx, b.accountID)
		if e != nil {
			return nil, e
		}
		if a.Credentials["api_key"] != issued.Key {
			a.Credentials["api_key"] = issued.Key
			if _, e = c.Account.UpdateOneID(a.ID).SetCredentials(a.Credentials).Save(ctx); e != nil {
				return nil, e
			}
			if e = publishVMAccount(ctx, c, a.ID, b.groupID); e != nil {
				return nil, e
			}
		}
		return existing, tx.Commit()
	}
	label := fmt.Sprintf("Claude · %s · %d", issued.Item.Username, id)
	a, err := c.Account.Create().SetName(label).SetPlatform(PlatformAnthropic).SetType(AccountTypeAPIKey).SetCredentials(map[string]any{"base_url": client.Origin, "api_key": issued.Key}).Save(ctx)
	if err != nil {
		return nil, err
	}
	g, err := c.Group.Create().SetName(label).SetPlatform(PlatformAnthropic).SetSubscriptionType(SubscriptionTypeSubscription).SetSubscriptionUsageSource("vm2api").SetIsExclusive(true).SetDefaultValidityDays(days).Save(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = c.AccountGroup.Create().SetAccountID(a.ID).SetGroupID(g.ID).Save(ctx); err != nil {
		return nil, err
	}
	if _, err = c.ExecContext(ctx, "INSERT INTO vm2api_subscription_bindings(user_id,remote_user_id,remote_username,origin,account_id,group_id) VALUES($1,$2,$3,$4,$5,$6)", id, issued.Item.ID, issued.Item.Username, client.Origin, a.ID, g.ID); err != nil {
		return nil, err
	}
	sub, err := s.createSubscription(txctx, &AssignSubscriptionInput{UserID: id, GroupID: g.ID, AssignedBy: admin, ValidityDays: days})
	if err != nil {
		return nil, err
	}
	if err = publishVMAccount(ctx, c, a.ID, g.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.maybeInvalidateAssignmentCaches(id, g.ID, false)
	return sub, nil
}

func publishVMAccount(ctx context.Context, c *dbent.Client, accountID, groupID int64) error {
	_, err := c.ExecContext(ctx, "INSERT INTO scheduler_outbox(event_type,account_id,group_id,payload) VALUES($1,$2,$3,jsonb_build_object('group_ids',jsonb_build_array($3::bigint)))", SchedulerOutboxEventAccountChanged, accountID, groupID)
	return err
}

// LoadSlotUsage resolves the group type in one query per page. Only dedicated
// subscriptions need a vm2api request. Empty and unavailable are distinct.
func (s *SubscriptionService) LoadSlotUsage(ctx context.Context, subs []UserSubscription) error {
	if len(subs) == 0 || s.entClient == nil {
		return nil
	}
	byGroup := make(map[int64][]int)
	ids := make([]string, 0, len(subs))
	for i, sub := range subs {
		if sub.Group == nil || sub.Group.SubscriptionUsageSource != "vm2api" {
			continue
		}
		subs[i].SlotUsage = &vmusers.Usage{Status: "unavailable"}
		if _, ok := byGroup[sub.GroupID]; !ok {
			ids = append(ids, strconv.FormatInt(sub.GroupID, 10))
		}
		byGroup[sub.GroupID] = append(byGroup[sub.GroupID], i)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.entClient.QueryContext(ctx, "SELECT group_id,user_id,remote_user_id,origin FROM vm2api_subscription_bindings WHERE group_id IN ("+strings.Join(ids, ",")+")")
	if err != nil {
		return err
	}
	type job struct {
		index  int
		remote string
	}
	var jobs []job
	client, configErr := vmusers.Configured()
	for rows.Next() {
		var group, owner int64
		var remote, origin string
		if err = rows.Scan(&group, &owner, &remote, &origin); err != nil {
			rows.Close()
			return err
		}
		for _, i := range byGroup[group] {
			subs[i].SlotUsage = &vmusers.Usage{Status: "unavailable"}
			if owner == subs[i].UserID && configErr == nil && origin == client.Origin {
				jobs = append(jobs, job{i, remote})
			}
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	queue := make(chan job)
	var workers sync.WaitGroup
	for range min(6, len(jobs)) {
		workers.Go(func() {
			for j := range queue {
				if u, e := client.Slots(ctx, j.remote); e == nil {
					subs[j.index].SlotUsage = u
				}
			}
		})
	}
	for _, j := range jobs {
		queue <- j
	}
	close(queue)
	workers.Wait()
	return nil
}
