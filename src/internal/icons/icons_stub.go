//go:build !windows

package icons

import "image"

func load(string) image.Image { return nil }
