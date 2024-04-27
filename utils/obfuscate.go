package utils

import (
	"fmt"
	"math/rand"
	"sync"
)

// Generate a random sequence of lowercase letters up to the given amount
func RandomAsciiName(chars int) string {
	result := make([]byte, chars)
	for i := range chars {
		result[i] = byte(97 + rand.Intn(26))
	}
	return string(result)
}

// A simple container for generating mappings between keys
// Basically a threadsafe dictionary with random keys (from RandomAsciiName)
// for things where some user-generated string is "secret" but we perhaps
// want special readonly access to the string without exposing it
type ObfuscatedKeys struct {
	associations          map[string]string // The mapping of public key to 'value'
	reverseassoc          map[string]string // The mapping of 'value' to public key
	lock                  sync.Mutex
	DefaultLength         int
	RetryToLengthIncrease int
}

// Generate the default random associator
func GetDefaultObfuscation() *ObfuscatedKeys {
	return &ObfuscatedKeys{
		associations:          make(map[string]string),
		reverseassoc:          make(map[string]string),
		DefaultLength:         5,
		RetryToLengthIncrease: 20,
	}
}

// Generate a random key to obfuscate the given user generated one
func (r *ObfuscatedKeys) GetObfuscatedKey(item string) string {
	r.lock.Lock()
	defer r.lock.Unlock()

	k, ok := r.reverseassoc[item]
	if ok {
		return k
	}

	retries := 0

	for {
		k = RandomAsciiName(r.DefaultLength + (retries / r.RetryToLengthIncrease))
		_, ok := r.associations[k]
		if !ok {
			// Not found, so this is good
			r.associations[k] = item
			r.reverseassoc[item] = k
			return k
		}
		retries += 1
	}
}

// Get the user generated key from the obfuscated key
func (r *ObfuscatedKeys) GetFromObfuscatedKey(key string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	item, ok := r.associations[key]
	if !ok {
		return "", fmt.Errorf("couldn't find key %s", key)
	}
	return item, nil
}
