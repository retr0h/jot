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

package jot_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/retr0h/jot/internal/jot"
)

type fakeProvider struct {
	data map[string]string
}

func (f *fakeProvider) Get(_ context.Context, key string) (string, error) {
	v, ok := f.data[key]
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

func (f *fakeProvider) Put(_ context.Context, key, value string) error {
	f.data[key] = value
	return nil
}

func (f *fakeProvider) List(_ context.Context) ([]string, error) {
	var keys []string
	for k := range f.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (f *fakeProvider) Delete(_ context.Context, key string) error {
	if _, ok := f.data[key]; !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	delete(f.data, key)
	return nil
}

func (f *fakeProvider) Name() string { return "test" }

func TestSecureStoreWriteRead(t *testing.T) {
	provider := &fakeProvider{data: make(map[string]string)}
	ss := jot.NewSecureStoreFromProvider(provider)

	ctx := context.Background()
	if err := ss.WriteNote(ctx, "secret-note", "top secret"); err != nil {
		t.Fatalf("WriteNote: %v", err)
	}
	body, err := ss.ReadNote(ctx, "secret-note")
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if body != "top secret" {
		t.Errorf("body = %q, want %q", body, "top secret")
	}
}

func TestSecureStoreDelete(t *testing.T) {
	provider := &fakeProvider{data: make(map[string]string)}
	ss := jot.NewSecureStoreFromProvider(provider)

	ctx := context.Background()
	ss.WriteNote(ctx, "secret-note", "content")
	if err := ss.DeleteNote(ctx, "secret-note"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	_, err := ss.ReadNote(ctx, "secret-note")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
