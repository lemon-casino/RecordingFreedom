//go:build !darwin && !linux

package secrets

func backendStatus(s *Store) (Status, error) {
	dir, err := diskDir(s)
	if err != nil {
		return Status{}, err
	}
	return Status{Backend: diskBackendName(), Dir: dir}, nil
}

func backendSave(s *Store, name string, secret string) error {
	return diskSave(s, name, secret)
}

func backendLoad(s *Store, name string) (string, bool, error) {
	return diskLoad(s, name)
}

func backendDelete(s *Store, name string) error {
	return diskDelete(s, name)
}
