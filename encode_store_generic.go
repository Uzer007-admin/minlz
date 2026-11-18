//go:build !amd64 || appengine || noasm || !gc || purego

package minlz

func preferCachedStore() bool { return false }
