package record

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func newSession(root string) (string, error) {
	if root == "" {
		root = "recordings"
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, time.Now().Format("20060102-150405")+"-")
}

type part struct {
	file     *os.File
	lastSync time.Time
}

func openPart(dir string, index int, now time.Time) (*part, error) {
	name := filepath.Join(dir, fmt.Sprintf("part-%06d.ts.partial", index))
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, e
	}
	return &part{f, now}, nil
}

func (p *part) write(data []byte, now time.Time) (int, error) {
	n, e := p.file.Write(data)
	if e != nil {
		return n, e
	}
	if now.Sub(p.lastSync) >= 5*time.Second {
		if e = p.file.Sync(); e != nil {
			return n, e
		}
		p.lastSync = now
	}
	return n, nil
}

func (p *part) finalize() (string, error) {
	name := p.file.Name()
	syncErr := p.file.Sync()
	closeErr := p.file.Close()
	if syncErr != nil {
		return name, syncErr
	}
	if closeErr != nil {
		return name, closeErr
	}
	final := name[:len(name)-len(".partial")]
	if e := os.Rename(name, final); e != nil {
		return name, e
	}
	return final, nil
}
