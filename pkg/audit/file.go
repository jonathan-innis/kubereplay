package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

type FileProvider struct {
	logPath string
}

func NewFileProvider(logPath string) (*FileProvider, error) {
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audit log file does not exist: %s", logPath)
	}
	return &FileProvider{logPath: logPath}, nil
}

func (p *FileProvider) GetEvents(_ context.Context, _, _ time.Duration, _ types.NamespacedName) ([]Event, error) {
	file, err := os.Open(p.logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var auditEvent Event
		if err := json.Unmarshal(scanner.Bytes(), &auditEvent); err != nil {
			continue
		}
		events = append(events, auditEvent)
	}
	return events, scanner.Err()
}
