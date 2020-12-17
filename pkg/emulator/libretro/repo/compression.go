package repo

type CompressionType string

func (c *CompressionType) GetExt() string {
	return (string)(*c)
}
