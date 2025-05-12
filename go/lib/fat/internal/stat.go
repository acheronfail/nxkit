package internal

type stat struct {
	entry Entry
	path  string
}

// IsArchive implements fat.Stat.
func (s *stat) IsArchive() bool {
	return s.entry.IsArchive()
}

// IsDir implements fat.Stat.
func (s *stat) IsDir() bool {
	return s.entry.IsDir()
}

// IsFile implements fat.Stat.
func (s *stat) IsFile() bool {
	return s.entry.IsFile()
}

// IsHidden implements fat.Stat.
func (s *stat) IsHidden() bool {
	return s.entry.IsHidden()
}

// IsReadOnly implements fat.Stat.
func (s *stat) IsReadOnly() bool {
	return s.entry.IsReadOnly()
}

// IsSystem implements fat.Stat.
func (s *stat) IsSystem() bool {
	return s.entry.IsSystem()
}

// LongName implements fat.DirectoryEntry.
func (s *stat) LongName() string {
	return s.entry.LongName()
}

// LongName implements fat.DirectoryEntry.
func (s *stat) ShortName() string {
	return s.entry.ShortName()
}

// Size implements fat.Stat.
func (s *stat) Size() int64 {
	return s.entry.Size()
}

func (s *stat) Path() string {
	return s.path
}
