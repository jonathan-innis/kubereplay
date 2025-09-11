package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuditEvent struct {
	Kind           string                 `json:"kind"`
	APIVersion     string                 `json:"apiVersion"`
	Level          string                 `json:"level"`
	AuditID        string                 `json:"auditID"`
	Stage          string                 `json:"stage"`
	RequestURI     string                 `json:"requestURI"`
	Verb           string                 `json:"verb"`
	User           User                   `json:"user"`
	ObjectRef      *ObjectReference       `json:"objectRef,omitempty"`
	ResponseStatus *metav1.Status         `json:"responseStatus,omitempty"`
	RequestObject  map[string]interface{} `json:"requestObject,omitempty"`
	ResponseObject map[string]interface{} `json:"responseObject,omitempty"`
	RequestReceivedTimestamp metav1.Time  `json:"requestReceivedTimestamp"`
	StageTimestamp           metav1.Time  `json:"stageTimestamp"`
}

type User struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

type ObjectReference struct {
	Resource        string `json:"resource"`
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	UID             string `json:"uid"`
	APIGroup        string `json:"apiGroup"`
	APIVersion      string `json:"apiVersion"`
	ResourceVersion string `json:"resourceVersion"`
}

type PodEvent struct {
	Timestamp   time.Time
	Event       string
	Description string
	Node        string
	Details     map[string]interface{}
}

func ParsePodEvents(auditLogPath, podName, namespace string) ([]PodEvent, error) {
	file, err := os.Open(auditLogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	var events []PodEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var auditEvent AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &auditEvent); err != nil {
			continue
		}

		if !isPodRelated(auditEvent, podName, namespace) {
			continue
		}

		podEvent := extractPodEvent(auditEvent)
		if podEvent != nil {
			events = append(events, *podEvent)
		}
	}

	return events, scanner.Err()
}

func isPodRelated(event AuditEvent, podName, namespace string) bool {
	if event.ObjectRef == nil {
		return false
	}

	return event.ObjectRef.Resource == "pods" &&
		event.ObjectRef.Name == podName &&
		event.ObjectRef.Namespace == namespace
}

func extractPodEvent(event AuditEvent) *PodEvent {
	podEvent := &PodEvent{
		Timestamp: event.StageTimestamp.Time,
		Details:   make(map[string]interface{}),
	}

	switch {
	case event.Verb == "create":
		podEvent.Event = "Pod Created"
		podEvent.Description = "Pod was created"
	case event.Verb == "update" && isBindingEvent(event):
		podEvent.Event = "Pod Bound"
		podEvent.Node = extractNodeName(event)
		podEvent.Description = fmt.Sprintf("Pod bound to node %s", podEvent.Node)
	case isKarpenterEvent(event):
		podEvent.Event = "Karpenter Nomination"
		podEvent.Description = "Pod nominated by Karpenter"
		podEvent.Node = extractKarpenterNode(event)
	case event.Verb == "patch" && isStatusUpdate(event):
		podEvent.Event = "Status Update"
		podEvent.Description = extractStatusChange(event)
	default:
		return nil
	}

	return podEvent
}

func isBindingEvent(event AuditEvent) bool {
	if event.RequestObject == nil {
		return false
	}
	spec, ok := event.RequestObject["spec"].(map[string]interface{})
	return ok && spec["nodeName"] != nil
}

func extractNodeName(event AuditEvent) string {
	if event.RequestObject == nil {
		return ""
	}
	if spec, ok := event.RequestObject["spec"].(map[string]interface{}); ok {
		if nodeName, ok := spec["nodeName"].(string); ok {
			return nodeName
		}
	}
	return ""
}

func isKarpenterEvent(event AuditEvent) bool {
	return strings.Contains(event.User.Username, "karpenter") ||
		strings.Contains(event.RequestURI, "karpenter")
}

func extractKarpenterNode(event AuditEvent) string {
	// Extract node information from Karpenter events
	return extractNodeName(event)
}

func isStatusUpdate(event AuditEvent) bool {
	return strings.Contains(event.RequestURI, "/status")
}

func extractStatusChange(event AuditEvent) string {
	if event.RequestObject == nil {
		return "Status updated"
	}
	if status, ok := event.RequestObject["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			return fmt.Sprintf("Phase changed to %s", phase)
		}
	}
	return "Status updated"
}
