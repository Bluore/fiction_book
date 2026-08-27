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

// UserFeedRepositoryTestSuite 用户 Feed 仓储测试套件
type UserFeedRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo UserFeedRepository
}

func (s *UserFeedRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(s.T(), err)

	err = db.AutoMigrate(&model.UserFeed{})
	assert.NoError(s.T(), err)

	s.db = db
	s.repo = NewUserFeedRepository(db)
}

func (s *UserFeedRepositoryTestSuite) TearDownTest() {
	s.db.Exec("DELETE FROM `user_feed`")
}

// createFeed 创建一条测试 Feed 记录
func (s *UserFeedRepositoryTestSuite) createFeed(feedID, userID string, createdAt time.Time) {
	err := s.db.Create(&model.UserFeed{
		FeedID:    feedID,
		UserID:    userID,
		CreatedAt: createdAt,
	}).Error
	assert.NoError(s.T(), err)
}

func (s *UserFeedRepositoryTestSuite) TestGetFeedByFeedID() {
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	s.createFeed("feed-1", "user-1", createdAt)

	feed, err := s.repo.GetFeedByFeedID(ctx, s.db, "feed-1")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), feed)
	assert.Equal(s.T(), "feed-1", feed.FeedID)
	assert.Equal(s.T(), "user-1", feed.UserID)
	assert.Equal(s.T(), createdAt, feed.CreatedAt)
}

func (s *UserFeedRepositoryTestSuite) TestGetFeedByFeedIDNotFound() {
	ctx := context.Background()

	feed, err := s.repo.GetFeedByFeedID(ctx, s.db, "non-existent-feed")
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), feed)
}

func (s *UserFeedRepositoryTestSuite) TestGetFeedListByUserIDWithStartCursorAndLimit() {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	// user-1 有 5 条 Feed，时间依次递增（t5 最新）
	for i := 1; i <= 5; i++ {
		s.createFeed("feed-"+string(rune('0'+i)), "user-1", base.Add(time.Duration(i)*time.Minute))
	}
	// 其他用户的 Feed，不应出现在 user-1 的查询结果中
	s.createFeed("feed-other", "user-2", base.Add(time.Hour))

	// 第一页：无游标，取最新的 2 条
	page1, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", time.Time{}, "", 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page1, 2)
	assert.Equal(s.T(), "feed-5", page1[0].FeedID)
	assert.Equal(s.T(), "feed-4", page1[1].FeedID)

	// 第二页：以上一页最后一条为游标
	page2, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", page1[1].CreatedAt, page1[1].FeedID, 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page2, 2)
	assert.Equal(s.T(), "feed-3", page2[0].FeedID)
	assert.Equal(s.T(), "feed-2", page2[1].FeedID)

	// 第三页
	page3, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", page2[1].CreatedAt, page2[1].FeedID, 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page3, 1)
	assert.Equal(s.T(), "feed-1", page3[0].FeedID)

	// 游标越过最旧记录后返回空
	page4, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", page3[0].CreatedAt, page3[0].FeedID, 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page4, 0)
}

func (s *UserFeedRepositoryTestSuite) TestGetFeedListByUserIDWithStartCursorAndLimitSameTimestamp() {
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	// 相同创建时间的 Feed，按 feed_id DESC 排序，用 feed_id 做游标打破平局
	s.createFeed("feed-b", "user-1", createdAt)
	s.createFeed("feed-a", "user-1", createdAt)

	page1, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(ctx, s.db, "user-1", time.Time{}, "", 1)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page1, 1)
	assert.Equal(s.T(), "feed-b", page1[0].FeedID)

	page2, err := s.repo.GetFeedListByUserIDWithStartCursorAndLimit(
		ctx, s.db, "user-1", page1[0].CreatedAt, page1[0].FeedID, 1)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), page2, 1)
	assert.Equal(s.T(), "feed-a", page2[0].FeedID)
}

func (s *UserFeedRepositoryTestSuite) TestGetFeedListByUserIDWithStartCursorAndEndCursor() {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	for i := 1; i <= 5; i++ {
		s.createFeed("feed-"+string(rune('0'+i)), "user-1", base.Add(time.Duration(i)*time.Minute))
	}

	// 窗口查询：起始游标 (t5, feed-5) 为最新边界（不含），结束游标 t3 为最旧边界（含）
	feeds, err := s.repo.GetFeedListByUserIDWithStartCursorAndEndCursor(
		ctx, s.db, "user-1", base.Add(5*time.Minute), "feed-5", base.Add(3*time.Minute))
	assert.NoError(s.T(), err)
	assert.Len(s.T(), feeds, 2)
	assert.Equal(s.T(), "feed-4", feeds[0].FeedID)
	assert.Equal(s.T(), "feed-3", feeds[1].FeedID)

	// 结束游标为零值时不做最旧边界限制，只按起始游标续读
	feeds, err = s.repo.GetFeedListByUserIDWithStartCursorAndEndCursor(
		ctx, s.db, "user-1", base.Add(4*time.Minute), "feed-4", time.Time{})
	assert.NoError(s.T(), err)
	assert.Len(s.T(), feeds, 3)
	assert.Equal(s.T(), "feed-3", feeds[0].FeedID)
	assert.Equal(s.T(), "feed-1", feeds[2].FeedID)
}

func (s *UserFeedRepositoryTestSuite) TestUpdateFeedByFeedID() {
	ctx := context.Background()
	s.createFeed("feed-1", "user-1", time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC))

	// 更新 UserID（UserFeed 内嵌的 feedContent 字段未导出，此处仅验证按 FeedID 定位更新）
	err := s.repo.UpdateFeedByFeedID(ctx, s.db, "feed-1", &model.UserFeed{UserID: "user-2"})
	assert.NoError(s.T(), err)

	feed, err := s.repo.GetFeedByFeedID(ctx, s.db, "feed-1")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), feed)
	assert.Equal(s.T(), "user-2", feed.UserID)
}

func (s *UserFeedRepositoryTestSuite) TestUpdateFeedByFeedIDNilDetail() {
	ctx := context.Background()
	s.createFeed("feed-1", "user-1", time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC))

	err := s.repo.UpdateFeedByFeedID(ctx, s.db, "feed-1", nil)
	assert.Error(s.T(), err)
}

func (s *UserFeedRepositoryTestSuite) TestDeleteFeedByFeedID() {
	ctx := context.Background()
	s.createFeed("feed-1", "user-1", time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC))

	err := s.repo.DeleteFeedByFeedID(ctx, s.db, "feed-1")
	assert.NoError(s.T(), err)

	// 软删除后普通查询不可见
	feed, err := s.repo.GetFeedByFeedID(ctx, s.db, "feed-1")
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), feed)

	// 物理行仍在，且 deleted_at 已写入
	var count int64
	s.db.Unscoped().Model(&model.UserFeed{}).Where("feed_id = ?", "feed-1").Count(&count)
	assert.Equal(s.T(), int64(1), count)

	var deleted model.UserFeed
	s.db.Unscoped().Model(&model.UserFeed{}).Where("feed_id = ?", "feed-1").First(&deleted)
	assert.False(s.T(), deleted.DeletedAt.Time.IsZero())
}

func TestUserFeedRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserFeedRepositoryTestSuite))
}
