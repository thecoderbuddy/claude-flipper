// Package accounts manages the sequence.json state file that tracks all registered accounts.
package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sharvari/claude-flipper/internal/models"
	"github.com/sharvari/claude-flipper/internal/paths"
)

// Load reads sequence.json and returns the Sequence, or an empty one if it doesn't exist yet.
func Load() (*models.Sequence, error) {
	data, err := os.ReadFile(paths.SequenceFile())
	if os.IsNotExist(err) {
		return &models.Sequence{
			ActiveSlot: 0,
			Accounts:   map[string]models.AccountRecord{},
			Sequence:   []int{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sequence file: %w", err)
	}

	var seq models.Sequence
	if err := json.Unmarshal(data, &seq); err != nil {
		return nil, fmt.Errorf("parse sequence file: %w", err)
	}
	if seq.Accounts == nil {
		seq.Accounts = map[string]models.AccountRecord{}
	}
	return &seq, nil
}

// Save atomically writes the Sequence back to sequence.json.
func Save(seq *models.Sequence) error {
	if err := os.MkdirAll(filepath.Dir(paths.SequenceFile()), 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	data, err := json.MarshalIndent(seq, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sequence: %w", err)
	}
	return atomicWrite(paths.SequenceFile(), data)
}

// NextSlot returns the next unused slot number (1-based).
func NextSlot(seq *models.Sequence) int {
	used := map[int]bool{}
	for k := range seq.Accounts {
		n, _ := strconv.Atoi(k)
		used[n] = true
	}
	for i := 1; ; i++ {
		if !used[i] {
			return i
		}
	}
}

// AddAccount adds a new account record to the sequence and returns its slot number.
// Returns an error if the email is already registered.
func AddAccount(seq *models.Sequence, acct models.OAuthAccount) (int, error) {
	// Check for duplicate email.
	for _, rec := range seq.Accounts {
		if strings.EqualFold(rec.Email, acct.EmailAddress) {
			return 0, fmt.Errorf("account %q is already registered", acct.EmailAddress)
		}
	}

	slot := NextSlot(seq)
	key := strconv.Itoa(slot)
	seq.Accounts[key] = models.AccountRecord{
		Email:            acct.EmailAddress,
		UUID:             acct.UUID,
		OrganizationUUID: acct.OrganizationUUID,
		OrganizationName: acct.OrganizationName,
		AddedAt:          time.Now().UTC(),
	}
	seq.Sequence = append(seq.Sequence, slot)
	return slot, nil
}

// RemoveAccount removes the account identified by slotOrEmail and returns the slot number removed.
func RemoveAccount(seq *models.Sequence, slotOrEmail string) (int, error) {
	slot, key, err := resolveKey(seq, slotOrEmail)
	if err != nil {
		return 0, err
	}
	if seq.ActiveSlot == slot && len(seq.Sequence) > 1 {
		return 0, fmt.Errorf("cannot remove the currently active account; switch to another account first")
	}
	delete(seq.Accounts, key)

	// Remove from sequence slice.
	newSeq := seq.Sequence[:0]
	for _, s := range seq.Sequence {
		if s != slot {
			newSeq = append(newSeq, s)
		}
	}
	seq.Sequence = newSeq
	return slot, nil
}

// FindAccount resolves a slot number or email to an AccountRecord.
func FindAccount(seq *models.Sequence, slotOrEmail string) (int, models.AccountRecord, error) {
	slot, key, err := resolveKey(seq, slotOrEmail)
	if err != nil {
		return 0, models.AccountRecord{}, err
	}
	return slot, seq.Accounts[key], nil
}

// ActiveAccount returns the active slot and its AccountRecord.
func ActiveAccount(seq *models.Sequence) (int, models.AccountRecord, bool) {
	if seq.ActiveSlot == 0 {
		return 0, models.AccountRecord{}, false
	}
	key := strconv.Itoa(seq.ActiveSlot)
	rec, ok := seq.Accounts[key]
	return seq.ActiveSlot, rec, ok
}

// NextInSequence returns the slot to switch to after the current activeSlot.
// It wraps around when it reaches the end.
func NextInSequence(seq *models.Sequence) (int, models.AccountRecord, error) {
	if len(seq.Sequence) < 2 {
		return 0, models.AccountRecord{}, fmt.Errorf("need at least 2 accounts to rotate; run 'flipper add' first")
	}

	cur := seq.ActiveSlot
	idx := -1
	for i, s := range seq.Sequence {
		if s == cur {
			idx = i
			break
		}
	}

	var nextSlot int
	if idx == -1 || idx == len(seq.Sequence)-1 {
		nextSlot = seq.Sequence[0]
	} else {
		nextSlot = seq.Sequence[idx+1]
	}

	key := strconv.Itoa(nextSlot)
	rec, ok := seq.Accounts[key]
	if !ok {
		return 0, models.AccountRecord{}, fmt.Errorf("sequence references slot %d which does not exist", nextSlot)
	}
	return nextSlot, rec, nil
}

// SortedSlots returns all slot numbers in ascending order.
func SortedSlots(seq *models.Sequence) []int {
	slots := make([]int, 0, len(seq.Accounts))
	for k := range seq.Accounts {
		n, _ := strconv.Atoi(k)
		slots = append(slots, n)
	}
	sort.Ints(slots)
	return slots
}

// resolveKey resolves a slot number string or email address to (slot, mapKey, error).
func resolveKey(seq *models.Sequence, slotOrEmail string) (int, string, error) {
	// Try numeric slot first.
	if n, err := strconv.Atoi(slotOrEmail); err == nil {
		key := strconv.Itoa(n)
		if _, ok := seq.Accounts[key]; ok {
			return n, key, nil
		}
		return 0, "", fmt.Errorf("no account in slot %d", n)
	}

	// Try email match (case-insensitive).
	for k, rec := range seq.Accounts {
		if strings.EqualFold(rec.Email, slotOrEmail) {
			n, _ := strconv.Atoi(k)
			return n, k, nil
		}
	}
	return 0, "", fmt.Errorf("no account matching %q", slotOrEmail)
}

// atomicWrite writes data to path via a temp file + rename.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
