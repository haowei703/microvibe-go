package service

import (
	"context"
	"fmt"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/event"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// LiveEventHandler live stream event handler
// Automatically handles all live streaming events
type LiveEventHandler struct {
	liveRepo repository.LiveStreamRepository
}

// NewLiveEventHandler creates a new live event handler
func NewLiveEventHandler(liveRepo repository.LiveStreamRepository) *LiveEventHandler {
	return &LiveEventHandler{
		liveRepo: liveRepo,
	}
}

// RegisterHandlers registers all event handlers to the event bus
func (h *LiveEventHandler) RegisterHandlers(bus event.EventBus) error {
	// Register live stream created event
	if err := bus.Subscribe(event.EventLiveStreamCreated, &event.EventListener{
		ID:      "live_stream_created_handler",
		Handler: h.handleLiveStreamCreated,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register live stream created event: %w", err)
	}

	// Register live stream started event
	if err := bus.Subscribe(event.EventLiveStreamStarted, &event.EventListener{
		ID:      "live_stream_started_handler",
		Handler: h.handleLiveStreamStarted,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register live stream started event: %w", err)
	}

	// Register live stream ended event
	if err := bus.Subscribe(event.EventLiveStreamEnded, &event.EventListener{
		ID:      "live_stream_ended_handler",
		Handler: h.handleLiveStreamEnded,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register live stream ended event: %w", err)
	}

	// Register user joined event
	if err := bus.Subscribe(event.EventLiveUserJoined, &event.EventListener{
		ID:      "live_user_joined_handler",
		Handler: h.handleUserJoined,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register user joined event: %w", err)
	}

	// Register user left event
	if err := bus.Subscribe(event.EventLiveUserLeft, &event.EventListener{
		ID:      "live_user_left_handler",
		Handler: h.handleUserLeft,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register user left event: %w", err)
	}

	// Register like received event
	if err := bus.Subscribe(event.EventLiveLikeReceived, &event.EventListener{
		ID:      "live_like_received_handler",
		Handler: h.handleLikeReceived,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register like received event: %w", err)
	}

	// Register gift received event
	if err := bus.Subscribe(event.EventLiveGiftReceived, &event.EventListener{
		ID:      "live_gift_received_handler",
		Handler: h.handleGiftReceived,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register gift received event: %w", err)
	}

	// Register comment received event
	if err := bus.Subscribe(event.EventLiveCommentReceived, &event.EventListener{
		ID:      "live_comment_received_handler",
		Handler: h.handleCommentReceived,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register comment received event: %w", err)
	}

	// Register share received event
	if err := bus.Subscribe(event.EventLiveShareReceived, &event.EventListener{
		ID:      "live_share_received_handler",
		Handler: h.handleShareReceived,
		Async:   true,
	}); err != nil {
		return fmt.Errorf("failed to register share received event: %w", err)
	}

	logger.Info("Live event handlers registered successfully",
		zap.Int("handler_count", 9))

	return nil
}

// handleLiveStreamCreated handles live stream created event
func (h *LiveEventHandler) handleLiveStreamCreated(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveStreamCreatedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveStreamCreatedEvent")
	}

	logger.Info("Processing live stream created event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("owner_id", evt.OwnerID),
		zap.String("title", evt.Title))

	// Business logic:
	// 1. Send creation notification
	// 2. Initialize live stream configuration
	// 3. Log to data warehouse
	// 4. Trigger recommendation system update

	return nil
}

// handleLiveStreamStarted handles live stream started event
func (h *LiveEventHandler) handleLiveStreamStarted(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveStreamStartedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveStreamStartedEvent")
	}

	logger.Info("Processing live stream started event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("owner_id", evt.OwnerID))

	// Business logic:
	// 1. Update live stream status to "live"
	if err := h.liveRepo.UpdateStatus(ctx, evt.LiveID, "live"); err != nil {
		logger.Error("Failed to update live stream status", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Set start time (use event timestamp)
	if err := h.liveRepo.UpdateStartTime(ctx, evt.LiveID, evt.Timestamp()); err != nil {
		logger.Error("Failed to update start time", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 3. Notify followers (async)
	// notificationService.NotifyFollowers(evt.OwnerID, "started streaming", evt.Title)

	// 4. Add to recommendation pool
	// recommendationService.AddLiveStream(evt.LiveID)

	logger.Info("Live stream started successfully",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID))

	return nil
}

// handleLiveStreamEnded handles live stream ended event
func (h *LiveEventHandler) handleLiveStreamEnded(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveStreamEndedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveStreamEndedEvent")
	}

	logger.Info("Processing live stream ended event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Int64("duration", evt.Duration),
		zap.Int("view_count", evt.ViewCount),
		zap.Int("like_count", evt.LikeCount))

	// Business logic:
	// 1. Update live stream status to "ended"
	if err := h.liveRepo.UpdateStatus(ctx, evt.LiveID, "ended"); err != nil {
		logger.Error("Failed to update live stream status", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Set end time (use event timestamp)
	if err := h.liveRepo.UpdateEndTime(ctx, evt.LiveID, evt.Timestamp()); err != nil {
		logger.Error("Failed to update end time", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 3. Save duration
	if err := h.liveRepo.UpdateDuration(ctx, evt.LiveID, evt.Duration); err != nil {
		logger.Error("Failed to update duration", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 4. Generate live stream report
	// reportService.GenerateLiveReport(evt.LiveID, evt.Duration, evt.ViewCount, evt.LikeCount, evt.GiftValue)

	// 5. Update streamer level and revenue
	// userService.UpdateStreamerStats(evt.OwnerID, evt.Duration, evt.GiftValue)

	logger.Info("Live stream ended successfully",
		zap.Uint("live_id", evt.LiveID),
		zap.Int64("duration", evt.Duration),
		zap.Int("total_views", evt.ViewCount))

	return nil
}

// handleUserJoined handles user joined event
func (h *LiveEventHandler) handleUserJoined(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveUserJoinedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveUserJoinedEvent")
	}

	logger.Debug("Processing user joined event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID),
		zap.String("username", evt.Username))

	// Business logic:
	// 1. Increment online count
	if err := h.liveRepo.IncrementOnlineCount(ctx, evt.LiveID); err != nil {
		logger.Error("Failed to increment online count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Increment view count
	if err := h.liveRepo.IncrementViewCount(ctx, evt.LiveID); err != nil {
		logger.Error("Failed to increment view count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 3. Send join message to chat room
	// chatService.SendSystemMessage(evt.RoomID, fmt.Sprintf("%s joined the live stream", evt.Username))

	// 4. Record watch history
	// historyService.RecordWatchHistory(evt.UserID, evt.LiveID)

	return nil
}

// handleUserLeft handles user left event
func (h *LiveEventHandler) handleUserLeft(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveUserLeftEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveUserLeftEvent")
	}

	logger.Debug("Processing user left event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID))

	// Business logic:
	// 1. Decrement online count
	if err := h.liveRepo.DecrementOnlineCount(ctx, evt.LiveID); err != nil {
		logger.Error("Failed to decrement online count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Update watch duration
	// historyService.UpdateWatchDuration(evt.UserID, evt.LiveID, evt.Duration)

	return nil
}

// handleLikeReceived handles like received event
func (h *LiveEventHandler) handleLikeReceived(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveLikeReceivedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveLikeReceivedEvent")
	}

	logger.Debug("Processing like received event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID),
		zap.Int("count", evt.Count))

	// Business logic:
	// 1. Increment like count
	if err := h.liveRepo.IncrementLikeCount(ctx, evt.LiveID, int64(evt.Count)); err != nil {
		logger.Error("Failed to increment like count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Send like animation message
	// chatService.SendLikeAnimation(evt.RoomID, evt.UserID, evt.Count)

	// 3. Update streamer popularity
	// userService.IncrementPopularity(evt.OwnerID, evt.Count)

	return nil
}

// handleGiftReceived handles gift received event
func (h *LiveEventHandler) handleGiftReceived(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveGiftReceivedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveGiftReceivedEvent")
	}

	logger.Info("Processing gift received event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID),
		zap.Uint("gift_id", evt.GiftID),
		zap.Int("amount", evt.Amount),
		zap.Int64("value", evt.Value))

	// Business logic:
	// 1. Increment gift count (use Amount field)
	if err := h.liveRepo.IncrementGiftCount(ctx, evt.LiveID, evt.Amount); err != nil {
		logger.Error("Failed to increment gift count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Increment gift value
	if err := h.liveRepo.IncrementGiftValue(ctx, evt.LiveID, evt.Value); err != nil {
		logger.Error("Failed to increment gift value", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 3. Send gift effect message
	// chatService.SendGiftEffect(evt.RoomID, evt.UserID, evt.GiftID, evt.Count)

	// 4. Update streamer revenue
	// incomeService.AddIncome(evt.OwnerID, evt.Value)

	// 5. Trigger ranking update
	// rankingService.UpdateGiftRanking(evt.LiveID, evt.UserID, evt.Value)

	return nil
}

// handleCommentReceived handles comment received event
func (h *LiveEventHandler) handleCommentReceived(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveCommentReceivedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveCommentReceivedEvent")
	}

	logger.Debug("Processing comment received event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID),
		zap.String("content", evt.Content))

	// Business logic:
	// 1. Increment comment count
	if err := h.liveRepo.IncrementCommentCount(ctx, evt.LiveID, 1); err != nil {
		logger.Error("Failed to increment comment count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Content moderation
	// if moderationService.CheckContent(evt.Content) {
	//     chatService.BroadcastComment(evt.RoomID, evt.UserID, evt.Username, evt.Content)
	// }

	// 3. Filter sensitive words
	// filteredContent := filterService.FilterWords(evt.Content)

	return nil
}

// handleShareReceived handles share received event
func (h *LiveEventHandler) handleShareReceived(ctx context.Context, e event.Event) error {
	evt, ok := e.(*event.LiveShareReceivedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected LiveShareReceivedEvent")
	}

	logger.Info("Processing share received event",
		zap.Uint("live_id", evt.LiveID),
		zap.String("room_id", evt.RoomID),
		zap.Uint("user_id", evt.UserID),
		zap.String("platform", evt.Platform))

	// Business logic:
	// 1. Increment share count
	if err := h.liveRepo.IncrementShareCount(ctx, evt.LiveID, 1); err != nil {
		logger.Error("Failed to increment share count", zap.Error(err), zap.Uint("live_id", evt.LiveID))
		return err
	}

	// 2. Give share reward
	// rewardService.GiveShareReward(evt.UserID)

	// 3. Boost live stream exposure
	// recommendationService.BoostLiveStream(evt.LiveID)

	// 4. Record share analytics
	// analyticsService.RecordShare(evt.LiveID, evt.UserID, evt.Platform)

	return nil
}
