package webstream

type WebStreamBacker_File struct {
	mu          sync.Mutex
  BackingFile string
}

func(wb * WebStreamBacker_File) WriteBack(data []byte) error {
  wb.mu.Lock()
  defer wb.mu.Unlock()
	writer, err := os.Create(wb.BackingFile)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = writer.Write(data)
  return err
	// if err != nil {
	// 	return err
	// }
  // return nil
}

func(wb * Web
