package playlist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PlayedItem struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"` // "song" or "text"
	Timestamp time.Time `json:"timestamp"`
}

type QueueItem struct {
	Name string `json:"name"`
	Type string `json:"type"` // "song" or "text"
}

var (
	mu        sync.Mutex
	filePath  = "./sound-data/played.json"
	queuePath = "./sound-data/queue.json"
	queueMu   sync.Mutex
)

// Init initializes the playlist configuration with a custom data directory.
// This allows different services to point to the correct volume mount location.
func Init(dataDir string) {
	mu.Lock()
	defer mu.Unlock()
	queueMu.Lock()
	defer queueMu.Unlock()

	filePath = filepath.Join(dataDir, "played.json")
	queuePath = filepath.Join(dataDir, "queue.json")
}

// ensureDir creates the directory if it doesn't exist
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// GetPlayedItems reads the list of played items from the JSON file.
func GetPlayedItems() ([]PlayedItem, error) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlayedItem{}, nil // Return empty list if file doesn't exist
		}
		return nil, err
	}

	if len(data) == 0 {
		return []PlayedItem{}, nil
	}

	var items []PlayedItem
	if err := json.Unmarshal(data, &items); err != nil {
		// If unmarshalling fails, it might be a corrupted file.
		// Return empty and let the next AddPlayedItem overwrite it.
		return []PlayedItem{}, nil
	}
	return items, nil
}

// AddPlayedItem adds a new item to the played list, removes old ones, and saves to file.
func AddPlayedItem(item PlayedItem, retentionPeriod time.Duration) error {
	mu.Lock()
	defer mu.Unlock()

	// Read existing items
	data, err := os.ReadFile(filePath)
	var items []PlayedItem
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File doesn't exist, start with an empty list
		items = []PlayedItem{}
	} else if len(data) > 0 {
		if err := json.Unmarshal(data, &items); err != nil {
			// Corrupted file, start fresh
			items = []PlayedItem{}
		}
	}

	// Add new item
	items = append(items, item)

	// Filter out old items
	var recentItems []PlayedItem
	cutoff := time.Now().Add(-retentionPeriod)
	for _, i := range items {
		if i.Timestamp.After(cutoff) {
			recentItems = append(recentItems, i)
		}
	}

	// Write back to file
	newData, err := json.MarshalIndent(recentItems, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := ensureDir(filePath); err != nil {
		return err
	}

	return os.WriteFile(filePath, newData, 0644)
}

// AddToQueue adds an item to the playback queue.
func AddToQueue(item QueueItem) error {
	queueMu.Lock()
	defer queueMu.Unlock()

	// Read existing queue
	data, err := os.ReadFile(queuePath)
	var queue []QueueItem
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		queue = []QueueItem{}
	} else if len(data) > 0 {
		if err := json.Unmarshal(data, &queue); err != nil {
			queue = []QueueItem{}
		}
	}

	// Add new item
	queue = append(queue, item)

	// Write back to file
	newData, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := ensureDir(queuePath); err != nil {
		return err
	}

	return os.WriteFile(queuePath, newData, 0644)
}

// GetNextQueueItem retrieves and removes the first item from the queue.
func GetNextQueueItem() (*QueueItem, error) {
	queueMu.Lock()
	defer queueMu.Unlock()

	data, err := os.ReadFile(queuePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No queue file means empty queue
		}
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var queue []QueueItem
	if err := json.Unmarshal(data, &queue); err != nil {
		return nil, err
	}

	if len(queue) == 0 {
		return nil, nil
	}

	// Get first item
	item := queue[0]
	queue = queue[1:]

	// Write remaining items back
	newData, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := ensureDir(queuePath); err != nil {
		return nil, err
	}

	if err := os.WriteFile(queuePath, newData, 0644); err != nil {
		return nil, err
	}

	return &item, nil
}

// GetQueueItems retrieves all items in the queue without removing them.
func GetQueueItems() ([]QueueItem, error) {
	queueMu.Lock()
	defer queueMu.Unlock()

	data, err := os.ReadFile(queuePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []QueueItem{}, nil // No queue file means empty queue
		}
		return nil, err
	}

	if len(data) == 0 {
		return []QueueItem{}, nil
	}

	var queue []QueueItem
	if err := json.Unmarshal(data, &queue); err != nil {
		return []QueueItem{}, nil
	}

	return queue, nil
}
