//go:build !windows && !linux
// +build !windows,!linux

package uggo

//not implemented!
func IsPipingStdin() bool {
	return false
}
