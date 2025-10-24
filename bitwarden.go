package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BitwardenClient handles interactions with Bitwarden CLI
type BitwardenClient struct {
	sessionKey string
}

// BitwardenItem represents a Bitwarden vault item
type BitwardenItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   int    `json:"type"`
	Notes  string `json:"notes"`
	Fields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Type  int    `json:"type"`
	} `json:"fields"`
}

// NewBitwardenClient creates a new Bitwarden client
func NewBitwardenClient() *BitwardenClient {
	sessionKey := os.Getenv("BW_SESSION")
	return &BitwardenClient{sessionKey: sessionKey}
}

// unlockVault prompts for password and unlocks the vault, returning session key
func (bw *BitwardenClient) unlockVault() error {
	// Check current vault status
	cmd := exec.Command("bw", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Bitwarden status: %w", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("failed to parse status: %w", err)
	}

	statusStr, _ := status["status"].(string)
	if statusStr == "unlocked" && bw.sessionKey == "" {
		fmt.Println("Getting session key from unlocked vault...")
	} else if statusStr != "unlocked" {
		fmt.Println("Bitwarden vault is locked. Please enter your master password:")
	}

	// Unlock to get session key
	cmd = exec.Command("bw", "unlock", "--raw")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to unlock Bitwarden: %w", err)
	}

	bw.sessionKey = strings.TrimSpace(string(output))
	fmt.Printf("\n✓ Vault unlocked. To skip password entry on future runs:\n")
	fmt.Printf("  export BW_SESSION=\"%s\"\n\n", bw.sessionKey)

	return nil
}

// Login authenticates with Bitwarden and returns session key
func (bw *BitwardenClient) Login() error {
	return bw.unlockVault()
}

// GetItem retrieves an item by name
func (bw *BitwardenClient) GetItem(name string) (*BitwardenItem, error) {
	// Try with current session (if any)
	cmd := exec.Command("bw", "get", "item", name, "--session", bw.sessionKey)
	output, err := cmd.CombinedOutput()

	// Check if it's an auth error (could be in err or in output)
	if (err != nil || bw.isAuthError(string(output))) && bw.isAuthError(string(output)) {
		if unlockErr := bw.unlockVault(); unlockErr != nil {
			return nil, unlockErr
		}
		// Retry with new session
		cmd = exec.Command("bw", "get", "item", name, "--session", bw.sessionKey)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return nil, fmt.Errorf("item not found: %s", name)
	}

	var item BitwardenItem
	if err := json.Unmarshal(output, &item); err != nil {
		// If parse failed due to auth prompt, retry after unlock
		if bw.isAuthError(string(output)) {
			if unlockErr := bw.unlockVault(); unlockErr != nil {
				return nil, unlockErr
			}
			cmd = exec.Command("bw", "get", "item", name, "--session", bw.sessionKey)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("item not found: %s", name)
			}
			if err := json.Unmarshal(output, &item); err != nil {
				return nil, fmt.Errorf("failed to parse item: %w", err)
			}
			return &item, nil
		}
		return nil, fmt.Errorf("failed to parse item: %w", err)
	}

	return &item, nil
}

// isAuthError checks if error output indicates authentication failure
func (bw *BitwardenClient) isAuthError(output string) bool {
	authErrors := []string{
		"not logged in",
		"invalid session",
		"session has expired",
		"unauthorized",
		"? master password:", // Vault is locked, prompting for password
		"invalid character '?' looking for beginning of value", // JSON parse error from password prompt
	}
	outputLower := strings.ToLower(output)
	for _, errMsg := range authErrors {
		if strings.Contains(outputLower, errMsg) {
			return true
		}
	}
	return false
}

// CreateSecureNote creates a secure note in Bitwarden
func (bw *BitwardenClient) CreateSecureNote(name string, fields map[string]string, notes string) error {
	// Build the item JSON
	itemFields := []map[string]interface{}{}
	for key, value := range fields {
		itemFields = append(itemFields, map[string]interface{}{
			"name":  key,
			"value": value,
			"type":  0, // text field
		})
	}

	item := map[string]interface{}{
		"type":   2, // secure note
		"name":   name,
		"notes":  notes,
		"fields": itemFields,
		"secureNote": map[string]interface{}{
			"type": 0,
		},
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to create item JSON: %w", err)
	}

	// Encode the item
	encodeCmd := exec.Command("bw", "encode")
	encodeCmd.Stdin = strings.NewReader(string(itemJSON))
	encoded, err := encodeCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to encode item: %w", err)
	}

	// Try to create the item
	createCmd := exec.Command("bw", "create", "item", string(encoded), "--session", bw.sessionKey)
	output, err := createCmd.CombinedOutput()

	// Retry once if auth error
	if err != nil && bw.isAuthError(string(output)) {
		if unlockErr := bw.unlockVault(); unlockErr != nil {
			return unlockErr
		}
		createCmd = exec.Command("bw", "create", "item", string(encoded), "--session", bw.sessionKey)
		output, err = createCmd.CombinedOutput()
	}

	if err != nil {
		return fmt.Errorf("failed to create item: %s", string(output))
	}

	// Sync
	syncCmd := exec.Command("bw", "sync", "--session", bw.sessionKey)
	if err := syncCmd.Run(); err != nil {
		fmt.Println("Warning: failed to sync with Bitwarden server")
	}

	return nil
}

// DeleteItem deletes an item by ID
func (bw *BitwardenClient) DeleteItem(itemID string) error {
	cmd := exec.Command("bw", "delete", "item", itemID, "--session", bw.sessionKey)
	output, err := cmd.CombinedOutput()

	// Retry once if auth error
	if err != nil && bw.isAuthError(string(output)) {
		if unlockErr := bw.unlockVault(); unlockErr != nil {
			return unlockErr
		}
		cmd = exec.Command("bw", "delete", "item", itemID, "--session", bw.sessionKey)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return fmt.Errorf("failed to delete item: %s", string(output))
	}

	// Sync
	syncCmd := exec.Command("bw", "sync", "--session", bw.sessionKey)
	if err := syncCmd.Run(); err != nil {
		fmt.Println("Warning: failed to sync with Bitwarden server")
	}

	return nil
}

// ListItems lists all items matching a search term
func (bw *BitwardenClient) ListItems(search string) ([]BitwardenItem, error) {
	cmd := exec.Command("bw", "list", "items", "--search", search, "--session", bw.sessionKey)
	output, err := cmd.CombinedOutput()

	// Retry once if auth error
	if err != nil && bw.isAuthError(string(output)) {
		if unlockErr := bw.unlockVault(); unlockErr != nil {
			return nil, unlockErr
		}
		cmd = exec.Command("bw", "list", "items", "--search", search, "--session", bw.sessionKey)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	var items []BitwardenItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}

	return items, nil
}

// GetFieldValue retrieves a field value from an item
func GetFieldValue(item *BitwardenItem, fieldName string) string {
	for _, field := range item.Fields {
		if field.Name == fieldName {
			return field.Value
		}
	}
	return ""
}
