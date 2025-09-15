package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/joinnis/kubereplay/pkg/object"
	"k8s.io/apimachinery/pkg/types"
)

type File struct {
	logPath string
}

func NewFile(logPath string) (*File, error) {
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audit log file does not exist: %s", logPath)
	}
	return &File{logPath: logPath}, nil
}

func (f *File) GetEvents(_ context.Context, _ object.ObjectParser, _ string, _, _ time.Time, _ types.NamespacedName) ([]auditmodel.Event, error) {
	file, err := os.Open(f.logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	var events []auditmodel.Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var auditEvent auditmodel.Event
		if err := json.Unmarshal(scanner.Bytes(), &auditEvent); err != nil {
			continue
		}
		events = append(events, auditEvent)
	}
	return events, scanner.Err()
}
