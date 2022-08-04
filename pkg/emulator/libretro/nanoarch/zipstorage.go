package nanoarch

type (
	ZipStorage struct {
		*StateStorage
	}
	ZipReaderWriter struct {
		ReaderWriter
	}
)

const zip = ".zip"

func NewZipStorage(store *StateStorage) *ZipStorage {
	store.rw = &ZipReaderWriter{store.rw}
	return &ZipStorage{StateStorage: store}
}

func (z *ZipStorage) GetSavePath() string { return z.StateStorage.GetSavePath() + zip }
func (z *ZipStorage) GetSRAMPath() string { return z.StateStorage.GetSRAMPath() + zip }

// Write writes the state to a file with the path.
func (zrw *ZipReaderWriter) Write(path string, data []byte) error {
	return zrw.ReaderWriter.Write(path, data)
}

// Read reads the state from a file with the path.
func (zrw *ZipReaderWriter) Read(path string) ([]byte, error) {
	return zrw.ReaderWriter.Read(path)
}
