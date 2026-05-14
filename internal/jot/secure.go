// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package jot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/retr0h/kvlt/pkg/kvlt"
)

// SecureStore wraps a kvlt Provider to store encrypted notes.
type SecureStore struct {
	provider kvlt.Provider
}

// NewSecureStore opens the "jot" vault from the kvlt store at configDir.
func NewSecureStore(configDir string) (*SecureStore, error) {
	store, err := kvlt.NewStore(configDir, nil)
	if err != nil {
		return nil, fmt.Errorf("kvlt store: %w", err)
	}
	provider, err := store.Open("jot")
	if err != nil {
		return nil, fmt.Errorf("kvlt open vault: %w", err)
	}
	return &SecureStore{provider: provider}, nil
}

// NewSecureStoreFromProvider creates a SecureStore from an existing provider.
// Used in tests with a fake provider.
func NewSecureStoreFromProvider(provider kvlt.Provider) *SecureStore {
	return &SecureStore{provider: provider}
}

// ReadNote retrieves the encrypted body of a note by slug.
func (s *SecureStore) ReadNote(ctx context.Context, slug string) (string, error) {
	body, err := s.provider.Get(ctx, slug)
	if err != nil {
		return "", fmt.Errorf("read secure note %q: %w", slug, err)
	}
	return body, nil
}

// WriteNote stores an encrypted note body under the given slug.
func (s *SecureStore) WriteNote(ctx context.Context, slug, body string) error {
	if err := s.provider.Put(ctx, slug, body); err != nil {
		return fmt.Errorf("write secure note %q: %w", slug, err)
	}
	return nil
}

// DeleteNote removes the encrypted note with the given slug.
func (s *SecureStore) DeleteNote(ctx context.Context, slug string) error {
	if err := s.provider.Delete(ctx, slug); err != nil {
		return fmt.Errorf("delete secure note %q: %w", slug, err)
	}
	return nil
}

// InitVault creates the kvlt vault named "jot" in configDir with the given recipients.
func InitVault(configDir string, recipientStrings []string) error {
	store, err := kvlt.NewStore(configDir, nil)
	if err != nil {
		return fmt.Errorf("kvlt store: %w", err)
	}
	_, err = store.Create("jot", kvlt.TypeLocal, recipientStrings)
	if err != nil {
		return fmt.Errorf("kvlt create vault: %w", err)
	}
	return nil
}

// DefaultSSHPubKey reads the user's default SSH public key.
func DefaultSSHPubKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, name := range []string{"id_ed25519.pub", "id_rsa.pub"} {
		path := filepath.Join(home, ".ssh", name)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("no SSH public key found in ~/.ssh/")
}
