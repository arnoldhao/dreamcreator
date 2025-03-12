package preferences

import (
	"path/filepath"
)

func (s *Service) GetPrefrenceConfigPath() string {
	fileName := s.pref.ConfigPath()
	return filepath.Dir(fileName)
}
