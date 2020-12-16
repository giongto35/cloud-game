package core

type Data struct {
	Url         string
	Compression CompressionType
}

type CompressionType string

func (c *CompressionType) GetExt() string {
	return (string)(*c)
}

type Repository interface {
	GetCoreData(file string, info ArchInfo) Data
}
