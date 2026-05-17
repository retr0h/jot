// Copyright (c) 2026 John Dewey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
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
	keys := make([]string, 0, len(f.data))
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

func TestSecureStore_WriteNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		body    string
		wantErr bool
	}{
		{
			name: "writes note successfully",
			slug: "secret-note",
			body: "top secret",
		},
		{
			name: "overwrites existing note",
			slug: "secret-note",
			body: "updated secret",
		},
		{
			name: "empty body is valid",
			slug: "empty-body",
			body: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &fakeProvider{data: make(map[string]string)}
			ss := jot.NewSecureStoreFromProvider(provider)
			ctx := context.Background()

			err := ss.WriteNote(ctx, tt.slug, tt.body)
			if tt.wantErr {
				if err == nil {
					t.Fatal("WriteNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteNote: unexpected error: %v", err)
			}

			got, err := ss.ReadNote(ctx, tt.slug)
			if err != nil {
				t.Fatalf("ReadNote after write: %v", err)
			}
			if got != tt.body {
				t.Errorf("body = %q, want %q", got, tt.body)
			}
		})
	}
}

func TestSecureStore_ReadNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    map[string]string
		slug    string
		want    string
		wantErr bool
	}{
		{
			name: "reads existing note",
			seed: map[string]string{"my-note": "hello world"},
			slug: "my-note",
			want: "hello world",
		},
		{
			name:    "missing note returns error",
			seed:    map[string]string{},
			slug:    "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &fakeProvider{data: tt.seed}
			ss := jot.NewSecureStoreFromProvider(provider)
			ctx := context.Background()

			got, err := ss.ReadNote(ctx, tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadNote: unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecureStore_DeleteNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    map[string]string
		slug    string
		wantErr bool
	}{
		{
			name: "deletes existing note",
			seed: map[string]string{"secret-note": "content"},
			slug: "secret-note",
		},
		{
			name:    "deleting nonexistent note returns error",
			seed:    map[string]string{},
			slug:    "ghost",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &fakeProvider{data: tt.seed}
			ss := jot.NewSecureStoreFromProvider(provider)
			ctx := context.Background()

			err := ss.DeleteNote(ctx, tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteNote: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteNote: unexpected error: %v", err)
			}

			_, err = ss.ReadNote(ctx, tt.slug)
			if err == nil {
				t.Fatal("ReadNote after delete: expected error, got nil")
			}
		})
	}
}
