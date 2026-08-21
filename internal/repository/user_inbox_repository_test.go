package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"fiction-book/internal/model"
)

// UserInboxRepositoryTestSuite 用户收件箱仓储测试套件
type UserInboxRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo UserInboxRepository
}

func (s *UserInboxRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(s.T(), err)

	err = db.AutoMigrate(&model.UserInbox{})
	assert.NoError(s.T(), err)

	s.db = db
	s.repo = NewUserInboxRepository(db)
}

func (s *UserInboxRepositoryTestSuite) TearDownTest() {
	s.db.Exec("DELETE FROM `user_inbox`")
}

// createInbox 创建一条测试收件箱记录
func (s *UserInboxRepositoryTestSuite) createInbox(id, userID, feedID string, createdAt time.Time) {
	err := s.db.Create(&model.UserInbox{
		ID:        id,
		UserID:    userID,
		FeedID:    feedID,
		CreatedAt: createdAt,
	}).Error
	assert.NoError(s.T(), err)
}

func (s *UserInboxRepositoryTestSuite) TestGetUserInboxIDListByUserIDWithStartCursorAndLimit() {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	// user-1 有 5 条收件箱记录，时间依次递增（t5 最新）
	for i := 1; i <= 5; i++ {
		s.createInbox("inbox-"+string(rune('0'+i)), "user-1", "feed-"+string(rune('0'+i)), base.Add(time.Duration(i)*time.Minute))
	}
	// 其他用户的记录，不应出现在 user-1 的查询结果中
	s.createInbox("inbox-other", "user-2", "feed-other", base.Add(time.Hour))

	// 第一页：无游标，取最新的 2 条
	page1, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", "", "", 2)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"feed-5", "feed-4"}, page1)

	// 第二页：以上一页最后一条记录 (created_at, id) 为游标
	page2, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", base.Add(4*time.Minute).Format(time.RFC3339Nano), "inbox-4", 2)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"feed-3", "feed-2"}, page2)

	// 第三页
	page3, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", base.Add(2*time.Minute).Format(time.RFC3339Nano), "inbox-2", 2)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"feed-1"}, page3)

	// 游标越过最旧记录后返回空
	page4, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", base.Add(time.Minute).Format(time.RFC3339Nano), "inbox-1", 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page4, 0)
}

func (s *UserInboxRepositoryTestSuite) TestGetUserInboxIDListByUserIDWithStartCursorAndLimitInvalidCursor() {
	ctx := context.Background()
	s.createInbox("inbox-1", "user-1", "feed-1", time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC))

	_, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", "not-a-time", "inbox-1", 10)
	assert.Error(s.T(), err)
}

func (s *UserInboxRepositoryTestSuite) TestCreateUserInboxBatch() {
	ctx := context.Background()

	successList, err := s.repo.CreateUserInboxBatch(ctx, s.db, "feed-1", []string{"user-1", "user-2", "user-3"})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"user-1", "user-2", "user-3"}, successList)

	var count int64
	s.db.Model(&model.UserInbox{}).Where("feed_id = ?", "feed-1").Count(&count)
	assert.Equal(s.T(), int64(3), count)

	var inboxList []model.UserInbox
	s.db.Model(&model.UserInbox{}).Where("feed_id = ?", "feed-1").Find(&inboxList)
	for _, inbox := range inboxList {
		assert.Equal(s.T(), "feed-1", inbox.FeedID)
		assert.NotEmpty(s.T(), inbox.ID)
	}
}

func (s *UserInboxRepositoryTestSuite) TestCreateUserInboxBatchEmptyInput() {
	ctx := context.Background()

	// 空 FeedID 直接返回
	successList, err := s.repo.CreateUserInboxBatch(ctx, s.db, "", []string{"user-1"})
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), successList)

	// 空用户列表直接返回
	successList, err = s.repo.CreateUserInboxBatch(ctx, s.db, "feed-1", nil)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), successList)

	// 空字符串用户被过滤
	successList, err = s.repo.CreateUserInboxBatch(ctx, s.db, "feed-1", []string{"user-1", "", "user-2"})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"user-1", "user-2"}, successList)

	var count int64
	s.db.Model(&model.UserInbox{}).Where("feed_id = ?", "feed-1").Count(&count)
	assert.Equal(s.T(), int64(2), count)
}

func (s *UserInboxRepositoryTestSuite) TestDeleteUserInboxBatch() {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	expireTime := base.Add(5 * time.Minute)

	s.createInbox("inbox-old-1", "user-1", "feed-old-1", base.Add(time.Minute))
	s.createInbox("inbox-old-2", "user-1", "feed-old-2", base.Add(2*time.Minute))
	s.createInbox("inbox-fresh", "user-1", "feed-fresh", base.Add(10*time.Minute))

	deletedList, err := s.repo.DeleteUserInboxBatch(ctx, s.db, expireTime)
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), []string{"inbox-old-1", "inbox-old-2"}, deletedList)

	// 未过期的记录仍然可见
	feedList, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", "", "", 10)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"feed-fresh"}, feedList)

	// 过期的记录已被软删除：普通查询不可见，物理行仍在且 deleted_at 已写入
	var count int64
	s.db.Unscoped().Model(&model.UserInbox{}).Where("id IN ?", []string{"inbox-old-1", "inbox-old-2"}).Count(&count)
	assert.Equal(s.T(), int64(2), count)

	var deleted model.UserInbox
	s.db.Unscoped().Model(&model.UserInbox{}).Where("id = ?", "inbox-old-1").First(&deleted)
	assert.False(s.T(), deleted.DeletedAt.Time.IsZero())
}

func (s *UserInboxRepositoryTestSuite) TestDeleteUserInboxBatchInDisk() {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	expireTime := base.Add(5 * time.Minute)

	s.createInbox("inbox-stale-1", "user-1", "feed-stale-1", base.Add(time.Minute))
	s.createInbox("inbox-stale-2", "user-1", "feed-stale-2", base.Add(2*time.Minute))
	s.createInbox("inbox-fresh", "user-1", "feed-fresh", base.Add(10*time.Minute))

	// 先软删除一批过期记录，再硬删除时应一并清除
	deletedList, err := s.repo.DeleteUserInboxBatch(ctx, s.db, expireTime)
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), []string{"inbox-stale-1", "inbox-stale-2"}, deletedList)

	inDiskDeletedList, err := s.repo.DeleteUserInboxBatchInDisk(ctx, s.db, expireTime)
	assert.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), []string{"inbox-stale-1", "inbox-stale-2"}, inDiskDeletedList)

	// 过期记录已物理删除，未过期记录保留
	var count int64
	s.db.Unscoped().Model(&model.UserInbox{}).Count(&count)
	assert.Equal(s.T(), int64(1), count)

	feedList, err := s.repo.GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", "", "", 10)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"feed-fresh"}, feedList)
}

func TestUserInboxRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserInboxRepositoryTestSuite))
}
