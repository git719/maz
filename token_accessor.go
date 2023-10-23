// token_accessor.go

// The https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.2.0/apps/cache/cache.go just defines the
// types, and expect you to craft a cache accessor implementation of your own. You can base yours on below examples:
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.2.0/apps/tests/integration/cache_accessor.go
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.2.0/apps/tests/devapps/sample_cache_accessor.go

// This is Microsoft's above apps/tests/integration/cache_accessor.go file, verbatim, but in this package

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package maz

import (
	"context"
	"log"
	"os"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type TokenCache struct {
	file string
}

func (t *TokenCache) Replace(ctx context.Context, cache cache.Unmarshaler, hints cache.ReplaceHints) error {
	data, err := os.ReadFile(t.file)
	if err != nil {
		log.Println(err)
	}
	return cache.Unmarshal(data)
}

func (t *TokenCache) Export(ctx context.Context, cache cache.Marshaler, hints cache.ExportHints) error {
	data, err := cache.Marshal()
	if err != nil {
		log.Println(err)
	}
	return os.WriteFile(t.file, data, 0600)
}

func (t *TokenCache) Print() string {
	data, err := os.ReadFile(t.file)
	if err != nil {
		return err.Error()
	}
	return string(data)
}
