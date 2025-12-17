package playlist

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQueueOperations(t *testing.T) {
	// Setup temp dir and override global paths
	tmpDir := t.TempDir()
	filePath = filepath.Join(tmpDir, "played.json")
	queuePath = filepath.Join(tmpDir, "queue.json")

	// 1. Test Empty Queue
	item, err := GetNextQueueItem()
	if err != nil {
		t.Fatalf("Failed to get from empty queue: %v", err)
	}
	if item != nil {
		t.Errorf("Expected nil item from empty queue, got %v", item)
	}

	// 2. Test AddToQueue
	testItem := QueueItem{Name: "test.mp3", Type: "song"}
	if err := AddToQueue(testItem); err != nil {
		t.Fatalf("Failed to add to queue: %v", err)
	}

	// 3. Test GetNextQueueItem
	item, err = GetNextQueueItem()
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}
	if item == nil || item.Name != "test.mp3" {
		t.Errorf("Expected test.mp3, got %v", item)
	}

	// 4. Queue should be empty again
	item, _ = GetNextQueueItem()
	if item != nil {
		t.Errorf("Queue should be empty after pop, got %v", item)
	}
}

func TestPlayedItemsRetention(t *testing.T) {
	// Setup temp dir
	tmpDir := t.TempDir()
	filePath = filepath.Join(tmpDir, "played.json")

	// 1. Add old item (should be cleaned up)
	oldItem := PlayedItem{Name: "old", Timestamp: time.Now().Add(-2 * time.Hour)}
	if err := AddPlayedItem(oldItem, 1*time.Hour); err != nil {
		t.Fatalf("Failed to add old item: %v", err)
	}

	// 2. Add new item (should be kept)
	newItem := PlayedItem{Name: "new", Timestamp: time.Now()}
	if err := AddPlayedItem(newItem, 1*time.Hour); err != nil {
		t.Fatalf("Failed to add new item: %v", err)
	}

	// 3. Verify only new item remains
	items, err := GetPlayedItems()
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item (retention logic), got %d", len(items))
	} else if items[0].Name != "new" {
		t.Errorf("Expected 'new' item, got %s", items[0].Name)
	}
}
