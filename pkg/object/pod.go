package object

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	EventTypePodCreated = "PodCreated"
	EventTypePodUpdated = "PodUpdated"
	EventTypePodBound   = "PodBound"
	EventTypePodEvicted = "PodEvicted"
	EventTypePodDeleted = "PodDeleted"
)

type Pod struct {
	Pod             *v1.Pod
	NamespaceName   types.NamespacedName
	NodeName        string
	CreationTime    time.Time
	LastUpdatedTime time.Time
	BindTime        time.Time
	EvictionTime    time.Time
	DeletionTime    time.Time
}

func (p Pod) Describe() string {
	return fmt.Sprintf(`
%s
%s
NodeName: %s

CreationTime: %s
LastUpdatedTime: %s
BindTime: %s
EvictionTime: %s
DeletionTime: %s

Nominations
------------
%s
`,
		p.NamespaceName,
		strings.Repeat("-", len(p.NamespaceName.String())),
		lo.Ternary(p.NodeName == "", "N/A", p.NodeName),
		lo.Ternary(p.CreationTime.IsZero(), "N/A", p.CreationTime.UTC().Format(time.RFC3339)),
		lo.Ternary(p.LastUpdatedTime.IsZero(), "N/A", p.CreationTime.UTC().Format(time.RFC3339)),
		lo.Ternary(p.BindTime.IsZero(), "N/A", p.BindTime.UTC().Format(time.RFC3339)),
		lo.Ternary(p.EvictionTime.IsZero(), "N/A", p.EvictionTime.UTC().Format(time.RFC3339)),
		lo.Ternary(p.DeletionTime.IsZero(), "N/A", p.DeletionTime.UTC().Format(time.RFC3339)),
		"<fill-in-nominations-here>",
	)
}

func (p Pod) Get() string {
	return string(lo.Must(yaml.Marshal(p.Pod)))
}

type PodParser struct{}

func (PodParser) Coalesce(nn types.NamespacedName, events []ParsedEvent) Object {
	p := Pod{NamespaceName: nn}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	for _, e := range events {
		if e.ObjectType != ObjectTypePod {
			continue
		}
		if e.NamespaceName.String() != nn.String() {
			continue
		}
		switch e.Event {
		case EventTypePodCreated:
			p.CreationTime = e.Timestamp
			p.Pod = e.Object.(*v1.Pod)
		case EventTypePodUpdated:
			p.LastUpdatedTime = e.Timestamp
			p.Pod = e.Object.(*v1.Pod)
		case EventTypePodBound:
			p.BindTime = e.Timestamp
			p.NodeName = e.AdditionalProperties["NodeName"]
		case EventTypePodEvicted:
			p.EvictionTime = e.Timestamp
		case EventTypePodDeleted:
			p.DeletionTime = e.Timestamp
		}
	}
	return p
}

func (PodParser) Extract(event auditmodel.Event) ParsedEvent {
	pe := ParsedEvent{
		Timestamp:            event.RequestReceivedTimestamp.Time,
		ObjectType:           ObjectTypePod,
		AdditionalProperties: map[string]string{},
	}
	var p v1.Pod
	switch {
	case event.Verb == "create" && strings.Contains(event.RequestURI, "binding"):
		pe.Event = EventTypePodBound
		pe.AdditionalProperties["NodeName"] = event.RequestObject["target"].(map[string]interface{})["name"].(string)
		pe.NamespaceName = types.NamespacedName{Namespace: event.ObjectRef.Namespace, Name: event.ObjectRef.Name}
	case event.Verb == "create" && strings.Contains(event.RequestURI, "eviction"):
		pe.Event = EventTypePodEvicted
		pe.NamespaceName = types.NamespacedName{Namespace: event.ObjectRef.Namespace, Name: event.ObjectRef.Name}
	case event.Verb == "create":
		pe.Event = EventTypePodCreated
		lo.Must0(json.Unmarshal(lo.Must(json.Marshal(event.ResponseObject)), &p))
		p.ManagedFields = nil
		pe.Object = &p
		pe.NamespaceName = client.ObjectKeyFromObject(&p)
	case event.Verb == "update":
		pe.Event = EventTypePodUpdated
		lo.Must0(json.Unmarshal(lo.Must(json.Marshal(event.ResponseObject)), &p))
		p.ManagedFields = nil
		pe.Object = &p
		pe.NamespaceName = client.ObjectKeyFromObject(&p)
	case event.Verb == "delete":
		pe.Event = EventTypePodDeleted
		pe.NamespaceName = types.NamespacedName{Namespace: event.ObjectRef.Namespace, Name: event.ObjectRef.Name}
	default:
		return ParsedEvent{}
	}
	return pe
}

func (PodParser) DescribeQuery(nn types.NamespacedName) string {
	podQueryTemplate := `
fields @timestamp, @message
| filter @logStream like "apiserver"
| filter verb like /create|delete/
| filter requestURI like "pods"
| filter @message like "%s"
| filter requestURI like "%s"
`
	return fmt.Sprintf(podQueryTemplate, nn.Name, nn.Namespace)
}

func (PodParser) GetQuery(nn types.NamespacedName) string {
	podQueryTemplate := `
fields @timestamp, @message
| filter @logStream like "apiserver"
| filter verb like /create|delete/
| filter requestURI like "pods"
| filter @message like "%s"
| filter requestURI like "%s"
`
	return fmt.Sprintf(podQueryTemplate, nn.Name, nn.Namespace)
}
