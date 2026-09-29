package equipment_requests

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"warehouse/internal/models"
)

// TransferCreator abstracts transfer creation to avoid circular dependency with transfers package
type TransferCreator interface {
	InitTransfer(req models.TransferRequest, transitStatus string) (int, error)
}

type Service struct {
	questRepo       QuestRepositoryInterface
	transferCreator TransferCreator

	// SSE broadcaster
	sseMu      sync.RWMutex
	sseClients map[chan QuestEvent]struct{}

	// Dispatch SSE hooks — set via SetDispatchHooks in DI.
	// onTransferDispatched is called when a transfer is created from a quest (on_mission).
	// onTransferEnded is called when a transfer is completed or cancelled (available).
	onTransferDispatched func(transferID int)
	onTransferEnded      func(transferID int)
}

// SetDispatchHooks wires dispatch SSE broadcast callbacks (avoids circular DI).
func (s *Service) SetDispatchHooks(dispatched, ended func(transferID int)) {
	s.onTransferDispatched = dispatched
	s.onTransferEnded = ended
}

func NewService(questRepo QuestRepositoryInterface) *Service {
	return &Service{
		questRepo:  questRepo,
		sseClients: make(map[chan QuestEvent]struct{}),
	}
}

// ============================================================================
// SSE Broadcaster
// ============================================================================

// Subscribe registers a channel to receive quest events over SSE.
// The returned channel is buffered (capacity 10) to avoid blocking the sync.
func (s *Service) Subscribe() chan QuestEvent {
	ch := make(chan QuestEvent, 10)
	s.sseMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the broadcaster and closes it.
func (s *Service) Unsubscribe(ch chan QuestEvent) {
	s.sseMu.Lock()
	delete(s.sseClients, ch)
	close(ch)
	s.sseMu.Unlock()
}

// broadcastEvent sends an event to all connected SSE clients.
// Slow clients are skipped (non-blocking send).
func (s *Service) broadcastEvent(event QuestEvent) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	if len(s.sseClients) == 0 {
		return
	}

	for ch := range s.sseClients {
		select {
		case ch <- event:
		default: // skip slow client
		}
	}
}

// BroadcastStocksChanged notifies SSE clients that stock inventory has changed.
// Called by StockService via callback wired in the DI container.
func (s *Service) BroadcastStocksChanged(locationID int, action string) {
	s.broadcastEvent(QuestEvent{Type: "stocks_changed", LocationID: locationID, Action: action})
}

// SetTransferCreator sets the transfer creator (called after DI wiring to avoid circular deps)
func (s *Service) SetTransferCreator(tc TransferCreator) {
	s.transferCreator = tc
}

// ============================================================================
// Phase 4: Quest → Transfer Integration
// ============================================================================

// ResolveQuestStockItems maps quest items (by category_id) to actual stock items at a source location
func (s *Service) ResolveQuestStockItems(quest *Quest, fromLocationID int) ([]ResolvedStockItem, []UnresolvedItem) {
	var resolved []ResolvedStockItem
	var unresolved []UnresolvedItem

	for _, item := range quest.Items {
		if item.Quantity == nil {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName:   item.Name,
				Quantity:   nil,
				CategoryID: item.CategoryID,
				Reason:     "quantity not specified",
			})
			continue
		}

		if item.CategoryID == nil {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName: item.Name,
				Quantity: item.Quantity,
				Reason:   "no category match for this item",
			})
			continue
		}

		matches, err := s.questRepo.FindStockItemsByCategory(fromLocationID, *item.CategoryID)
		if err != nil || len(matches) == 0 {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName:   item.Name,
				Quantity:   item.Quantity,
				CategoryID: item.CategoryID,
				Reason:     "no stock found at source location for this category",
			})
			continue
		}

		// Use the first matching stock item (same category at that location)
		match := matches[0]
		resolved = append(resolved, ResolvedStockItem{
			StockID:      match.StockID,
			CategoryID:   match.CategoryID,
			CategoryName: match.CategoryName,
			ItemName:     item.Name,
			Quantity:     *item.Quantity,
			Available:    match.Quantity,
		})
	}

	return resolved, unresolved
}

// PreviewTransferFromQuest builds a preview of what a transfer from this quest would look like
func (s *Service) PreviewTransferFromQuest(ctx context.Context, questID string, fromLocationID int) (*TransferPreview, error) {
	quest, err := s.questRepo.GetQuestByID(ctx, questID)
	if err != nil {
		return nil, fmt.Errorf("quest not found: %w", err)
	}

	preview := &TransferPreview{
		FromLocationID: fromLocationID,
	}

	// Destination is the quest's stored location (set when the quest was created or fixed manually)
	if quest.LocationID != nil {
		preview.ToLocationID = quest.LocationID
	}

	// Resolve stock items
	preview.ResolvedItems, preview.UnresolvedItems = s.ResolveQuestStockItems(quest, fromLocationID)

	return preview, nil
}

// CreateTransferFromQuest creates an inventory transfer from a quest
func (s *Service) CreateTransferFromQuest(ctx context.Context, questID string, req CreateTransferFromQuestRequest) (int, error) {
	if s.transferCreator == nil {
		return 0, fmt.Errorf("transfer service not configured")
	}

	// 1. Fetch and validate quest
	quest, err := s.questRepo.GetQuestByID(ctx, questID)
	if err != nil {
		return 0, fmt.Errorf("quest not found: %w", err)
	}

	if quest.Status == "completed" || quest.Status == "cancelled" {
		return 0, fmt.Errorf("cannot create transfer for quest with status '%s'", quest.Status)
	}

	// 2. Resolve destination location
	toLocationID := 0
	if req.ToLocationID != nil {
		toLocationID = *req.ToLocationID
	} else {
		if quest.LocationID == nil {
			return 0, fmt.Errorf("could not resolve destination location from pavilion '%s' and location '%s' — provide to_location_id explicitly",
				quest.Destination.Pavilion, quest.Destination.Location)
		}
		toLocationID = *quest.LocationID
	}

	// 3. Build transfer request
	transferReq := models.TransferRequest{
		FromLocationID: req.FromLocationID,
		LocationID:     toLocationID,
	}

	// 3a. Stock items — only what the request explicitly lists. We do NOT auto-resolve
	// from the quest items, so callers can send assets-only, stock-only, or a combination.
	// (Use the transfer-preview endpoint to get suggested stock items to send back here.)
	for _, si := range req.StockItems {
		transferReq.StockItemCollection = append(transferReq.StockItemCollection, models.StockItemRequest{
			ID:       si.ID,
			Quantity: si.Quantity,
		})
	}

	// Must move at least something.
	if len(req.StockItems) == 0 && len(req.Assets) == 0 {
		return 0, fmt.Errorf("transfer must include at least one stock item or asset")
	}

	// 3b. Assets — optional
	for _, a := range req.Assets {
		transferReq.AssetItemCollection = append(transferReq.AssetItemCollection, models.AssetItemRequest{
			ID: a.ID,
		})
	}

	// 3c. Users — optional
	for _, u := range req.Users {
		transferReq.Users = append(transferReq.Users, models.TransferUser{
			UserID: u.ID,
		})
	}

	// 4. Create the transfer
	transferID, err := s.transferCreator.InitTransfer(transferReq, "in_transit")
	if err != nil {
		return 0, fmt.Errorf("failed to create transfer: %w", err)
	}

	// 5. Link quest to transfer
	if err := s.questRepo.AddTransferToQuest(ctx, questID, transferID); err != nil {
		log.Printf("[equipment-requests] ORPHANED TRANSFER %d: created but failed to link to quest %s — manual cleanup required: %v", transferID, questID, err)
		return transferID, fmt.Errorf("transfer created (ID: %d) but failed to link to quest: %w", transferID, err)
	}

	log.Printf("[equipment-requests] Created transfer %d from quest %s", transferID, questID)
	if s.onTransferDispatched != nil {
		go s.onTransferDispatched(transferID)
	}
	return transferID, nil
}

// OnTransferStatusChanged is called by the transfer service when a linked transfer changes status.
// Implements the TransferStatusCallback interface.
func (s *Service) OnTransferStatusChanged(transferID int, newStatus string) error {
	ctx := context.Background()

	quest, err := s.questRepo.GetQuestByTransferID(ctx, transferID)
	if err != nil {
		return fmt.Errorf("failed to find quest for transfer %d: %w", transferID, err)
	}
	if quest == nil {
		return nil // transfer not linked to any quest
	}

	switch newStatus {
	case "completed":
		// Check whether any other active transfers remain.
		active, err := s.questRepo.GetActiveTransfersForQuest(ctx, quest.ID)
		if err != nil {
			return fmt.Errorf("failed to check active transfers for quest %s: %w", quest.ID, err)
		}
		// The completed transfer still has status "completed" in the DB at this point,
		// so GetActiveTransfersForQuest won't include it — length 0 means this was the last one.
		if len(active) == 0 {
			// Last active transfer is done — but only auto-close the quest if the goods
			// actually delivered across all transfers cover what was requested. If anything
			// is short (or can't be verified), leave the quest in_progress so an operator
			// can dispatch the remainder rather than silently marking it fulfilled.
			fulfilled, err := s.questRepo.GetFulfilledQuantitiesByCategory(ctx, quest.ID)
			if err != nil {
				return fmt.Errorf("failed to check fulfillment for quest %s: %w", quest.ID, err)
			}
			if complete, shortfalls := questFulfillment(quest, fulfilled); complete {
				if err := s.questRepo.UpdateQuestStatus(ctx, quest.ID, "completed"); err != nil {
					return fmt.Errorf("failed to complete quest %s: %w", quest.ID, err)
				}
				log.Printf("[equipment-requests] Quest %s completed — all transfers done and requested items fulfilled", quest.ID)
			} else {
				log.Printf("[equipment-requests] Transfer %d completed and no active transfers remain, but quest %s is not fully fulfilled — kept in_progress (%s)",
					transferID, quest.ID, strings.Join(shortfalls, "; "))
			}
		} else {
			log.Printf("[equipment-requests] Transfer %d completed, quest %s still has %d active transfer(s)", transferID, quest.ID, len(active))
		}
		if s.onTransferEnded != nil {
			go s.onTransferEnded(transferID)
		}

	case "cancelled":
		// RemoveTransferFromQuest handles resetting quest to pending when all transfers are gone.
		if err := s.questRepo.RemoveTransferFromQuest(ctx, transferID); err != nil {
			return fmt.Errorf("failed to remove cancelled transfer %d from quest %s: %w", transferID, quest.ID, err)
		}
		log.Printf("[equipment-requests] Transfer %d cancelled, removed from quest %s", transferID, quest.ID)
		if s.onTransferEnded != nil {
			go s.onTransferEnded(transferID)
		}
	}

	return nil
}

// questFulfillment reports whether everything the quest requested has been delivered,
// comparing requested quantities (aggregated per category from quest.Items) against the
// quantities actually delivered per category (fulfilled, from completed transfers).
//
// It is deliberately conservative: an item that can't be verified — no category match or
// no quantity — counts as a shortfall, because we cannot prove it was sent.
// The second return value lists human-readable shortfalls for logging. A quest with no
// items is trivially complete.
func questFulfillment(quest *Quest, fulfilled map[int]int) (bool, []string) {
	requested := make(map[int]int) // category_id -> total requested quantity
	var shortfalls []string

	for _, item := range quest.Items {
		if item.CategoryID == nil {
			shortfalls = append(shortfalls, fmt.Sprintf("item %q has no category match", item.Name))
			continue
		}
		if item.Quantity == nil {
			shortfalls = append(shortfalls, fmt.Sprintf("item %q (category %d) has no quantity", item.Name, *item.CategoryID))
			continue
		}
		requested[*item.CategoryID] += *item.Quantity
	}

	for categoryID, want := range requested {
		if got := fulfilled[categoryID]; got < want {
			shortfalls = append(shortfalls, fmt.Sprintf("category %d: requested %d, delivered %d", categoryID, want, got))
		}
	}

	return len(shortfalls) == 0, shortfalls
}
