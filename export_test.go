package wsl

// This file exports private functions used for unit testing

// Types
type Flags = wslFlags

// Configuration methods

func (c *Configuration) UnpackFlags(f Flags) {
	c.unpackFlags(f)
}

func (c *Configuration) PackFlags() (Flags, error) {
	return c.packFlags()
}
