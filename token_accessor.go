// token_accessor.go

// The https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v0.3.1/apps/cache/cache.go just defines the
// types, and expect you to craft a cache accessor implementation of your own. Based on library example
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v0.3.1/apps/tests/devapps/sample_cache_accessor.go

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ezmsal

import (
	"io/ioutil"
	"log"
	"os"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type TokenCache struct {
	file string
}

func (t *TokenCache) Replace(cache cache.Unmarshaler, key string) {
	jsonFile, err := os.Open(t.file)
	if err != nil { log.Println(err) }
	defer jsonFile.Close()
	data, err := ioutil.ReadAll(jsonFile)
	if err != nil { log.Println(err) }
	err = cache.Unmarshal(data)
	if err != nil { log.Println(err) }
}

func (t *TokenCache) Export(cache cache.Marshaler, key string) {
	data, err := cache.Marshal()
	if err != nil { log.Println(err) }
	err = ioutil.WriteFile(t.file, data, 0600)
	if err != nil { log.Println(err) }
}
