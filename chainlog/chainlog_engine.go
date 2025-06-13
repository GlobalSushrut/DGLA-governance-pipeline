package chainlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"
)

// LogEntry represents a single entry in the chain log
type LogEntry struct {
	EntryID   string                 `json:"entry_id"`
	RuleHash  string                 `json:"rule_hash"`
	ProofHash string                 `json:"proof_hash"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AnchorData represents an anchored log entry
type AnchorData struct {
	MerkleRoot string    `json:"merkle_root"`
	RuleIndex  string    `json:"rule_index"`
	Timestamp  time.Time `json:"timestamp"`
	AnchorTxID string    `json:"anchor_tx_id"`
	Target     string    `json:"target"` // Ethereum, Celestia, IPFS
}

// ChainLogEngine provides an append-only DAG for audit logs
type ChainLogEngine struct {
	logs        []LogEntry
	anchors     []AnchorData
	mu          sync.RWMutex
	storagePath string
}

// NewChainLogEngine creates a new ChainLog engine
func NewChainLogEngine(storagePath string) *ChainLogEngine {
	return &ChainLogEngine{
		logs:        make([]LogEntry, 0),
		anchors:     make([]AnchorData, 0),
		storagePath: storagePath,
	}
}

// AppendEntry adds a new log entry to the chain
func (c *ChainLogEngine) AppendEntry(entry LogEntry) error {
	if entry.EntryID == "" {
		return errors.New("entry_id is required")
	}

	if entry.RuleHash == "" {
		return errors.New("rule_hash is required")
	}

	if entry.ProofHash == "" {
		return errors.New("proof_hash is required")
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for duplicate entry IDs
	for _, e := range c.logs {
		if e.EntryID == entry.EntryID {
			return fmt.Errorf("duplicate entry_id: %s", entry.EntryID)
		}
	}

	c.logs = append(c.logs, entry)
	return nil
}

// GetLogs returns all log entries
func (c *ChainLogEngine) GetLogs() []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	logs := make([]LogEntry, len(c.logs))
	copy(logs, c.logs)
	return logs
}

// GetLogsByDate returns log entries for a specific date
func (c *ChainLogEngine) GetLogsByDate(date time.Time) []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	year, month, day := date.Date()
	result := make([]LogEntry, 0)

	for _, entry := range c.logs {
		entryYear, entryMonth, entryDay := entry.Timestamp.Date()
		if entryYear == year && entryMonth == month && entryDay == day {
			result = append(result, entry)
		}
	}

	return result
}

// AddAnchor adds a blockchain anchor for a set of logs
func (c *ChainLogEngine) AddAnchor(anchor AnchorData) error {
	if anchor.MerkleRoot == "" {
		return errors.New("merkle_root is required")
	}

	if anchor.Target == "" {
		return errors.New("target is required")
	}

	if anchor.Timestamp.IsZero() {
		anchor.Timestamp = time.Now()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.anchors = append(c.anchors, anchor)
	return nil
}

// GetAnchors returns all anchors
func (c *ChainLogEngine) GetAnchors() []AnchorData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	anchors := make([]AnchorData, len(c.anchors))
	copy(anchors, c.anchors)
	return anchors
}

// ExportLogsJSON exports logs to a JSON file
func (c *ChainLogEngine) ExportLogsJSON(filename string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c.logs, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// ExportAnchorsJSON exports anchors to a JSON file
func (c *ChainLogEngine) ExportAnchorsJSON(filename string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c.anchors, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// LoadLogsFromJSON loads logs from a JSON file
func (c *ChainLogEngine) LoadLogsFromJSON(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var logs []LogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = logs

	return nil
}

// SaveLogsToFile saves the current logs to a file
func (c *ChainLogEngine) SaveLogsToFile() error {
	if c.storagePath == "" {
		return errors.New("storage path not configured")
	}

	return c.ExportLogsJSON(c.storagePath + "/chainlog.json")
}

// LoadLogsFromFile loads logs from a file
func (c *ChainLogEngine) LoadLogsFromFile() error {
	if c.storagePath == "" {
		return errors.New("storage path not configured")
	}

	return c.LoadLogsFromJSON(c.storagePath + "/chainlog.json")
}
