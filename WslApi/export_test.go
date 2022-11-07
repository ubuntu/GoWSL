package WslApi

// Types
type Flags = winWslFlags

// Configuration methods

func (c *Configuration) UnpackFlags(f Flags) {
	c.unpackFlags(f)
}

func (c *Configuration) PackFlags() (Flags, error) {
	return c.packFlags()
}
