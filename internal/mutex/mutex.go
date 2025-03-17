// Copyright (c) 2025 Cloud-Native Toolkit
// SPDX-License-Identifier: MIT

package mutexkv

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"sync"
)

// MutexKV is a simple key/value store for arbitrary mutexes. It can be used to
// serialize changes across arbitrary collaborators that share knowledge of the
// keys they must serialize on.
//
// The initial use case is to let aws_security_group_rule resources serialize
// their access to individual security groups based on SG ID.
type MutexKV struct {
	lock  sync.Mutex
	store map[string]*sync.Mutex
}

// Lock - Locks the mutex for the given key. Caller is responsible for calling Unlock
// for the same key.
func (m *MutexKV) Lock(ctx context.Context, key string) {
	tflog.Trace(ctx, fmt.Sprintf("Locking %q", key))
	m.get(key).Lock()
	tflog.Trace(ctx, fmt.Sprintf("Locked %q", key))
}

// Unlock - Unlock the mutex for the given key. Caller must have called Lock for the same key first.
func (m *MutexKV) Unlock(ctx context.Context, key string) {
	tflog.Trace(ctx, fmt.Sprintf("Unlocking %q", key))
	m.get(key).Unlock()
	tflog.Trace(ctx, fmt.Sprintf("Unlocked %q", key))
}

// get - Returns a mutex for the given key, no guarantee of its lock status.
func (m *MutexKV) get(key string) *sync.Mutex {
	m.lock.Lock()
	defer m.lock.Unlock()
	mutex, ok := m.store[key]
	if !ok {
		mutex = &sync.Mutex{}
		m.store[key] = mutex
	}
	return mutex
}

// NewMutexKV - Returns a properly initialized MutexKV.
func NewMutexKV() *MutexKV {
	return &MutexKV{
		store: make(map[string]*sync.Mutex),
	}
}
