package desktopapp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

const privacyPinFileName = "privacy-pin.json"

type PrivacyPinFile struct {
	PinHash string `json:"pinHash"`
}

func privacyPinPath(ctx *VaultContext) string {
	return filepath.Join(ctx.VeloDir, privacyPinFileName)
}

func loadPrivacyPin(ctx *VaultContext) (PrivacyPinFile, error) {
	path := privacyPinPath(ctx)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return PrivacyPinFile{}, nil
	}
	if err != nil {
		return PrivacyPinFile{}, err
	}
	var file PrivacyPinFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return PrivacyPinFile{}, fmt.Errorf("read privacy pin: %w", err)
	}
	return file, nil
}

func savePrivacyPin(ctx *VaultContext, hash string) error {
	file := PrivacyPinFile{PinHash: hash}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	path := privacyPinPath(ctx)
	return os.WriteFile(path, append(raw, '\n'), 0600)
}

func setPrivacyPin(ctx *VaultContext, pin string) error {
	if len(pin) < 4 {
		return fmt.Errorf("PIN must be at least 4 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash PIN: %w", err)
	}
	return savePrivacyPin(ctx, string(hash))
}

func verifyPrivacyPin(ctx *VaultContext, pin string) (bool, error) {
	file, err := loadPrivacyPin(ctx)
	if err != nil {
		return false, err
	}
	if file.PinHash == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(file.PinHash), []byte(pin))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func hasPrivacyPin(ctx *VaultContext) (bool, error) {
	file, err := loadPrivacyPin(ctx)
	if err != nil {
		return false, err
	}
	return file.PinHash != "", nil
}

func isPrivateAndLocked(ctx *VaultContext, private bool) bool {
	if ctx == nil || !private {
		return false
	}
	if ctx.PrivateUnlocked {
		return false
	}
	hasPin, err := hasPrivacyPin(ctx)
	if err != nil || !hasPin {
		return false
	}
	return true
}

func redactPrivateMemo(memo MemoRecord) MemoRecord {
	memo.Content = "[PRIVATE]"
	memo.Tags = []string{}
	memo.References = []string{}
	return memo
}

func redactPrivateComment(comment MemoCommentRecord) MemoCommentRecord {
	comment.Content = "[PRIVATE]"
	comment.Tags = []string{}
	comment.References = []string{}
	return comment
}

func redactPrivateTask(task TaskRecord) TaskRecord {
	task.Title = "[PRIVATE]"
	task.Notes = ""
	task.Tags = []string{}
	task.Contexts = []string{}
	task.Links = []TaskLink{}
	return task
}
