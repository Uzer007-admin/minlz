//go:build !appengine && !noasm && gc && !purego && !amd64.v2

package minlz

const prefixSSEHardware = false
