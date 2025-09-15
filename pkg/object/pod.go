package object

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	EventTypePodCreated = "PodCreated"
	EventTypePodBound   = "PodBound"
	EventTypePodEvicted = "PodEvicted"
	EventTypePodDeleted = "PodDeleted"
)

type Pod struct {
	Pod           *v1.Pod
	NamespaceName types.NamespacedName
	NodeName      string
	CreationTime  time.Time
	BindTime      time.Time
	EvictionTime  time.Time
	DeletionTime  time.Time
}

func (e Pod) Describe() string {
	return fmt.Sprintf(`
%s
%s
NodeName: %s

CreationTime: %s
BindTime: %s
EvictionTime: %s
DeletionTime: %s

Nominations
------------
%s
`,
		e.NamespaceName,
		strings.Repeat("-", len(e.NamespaceName.String())),
		lo.Ternary(e.NodeName == "", "N/A", e.NodeName),
		lo.Ternary(e.CreationTime.IsZero(), "N/A", e.CreationTime.UTC().Format(time.RFC3339)),
		lo.Ternary(e.BindTime.IsZero(), "N/A", e.BindTime.UTC().Format(time.RFC3339)),
		lo.Ternary(e.EvictionTime.IsZero(), "N/A", e.EvictionTime.UTC().Format(time.RFC3339)),
		lo.Ternary(e.DeletionTime.IsZero(), "N/A", e.DeletionTime.UTC().Format(time.RFC3339)),
		"<fill-in-nominations-here>",
	)
}

func (e Pod) Get() string {
	return string(lo.Must(yaml.Marshal(e.Pod)))
}

type PodParser struct{}

func (PodParser) Coalesce(nn types.NamespacedName, events []ParsedEvent) Object {
	pd := Pod{NamespaceName: nn}
	lop.ForEach(events, func(e ParsedEvent, _ int) {
		if e.ObjectType != ObjectTypePod {
			return
		}
		if e.NamespaceName.String() != nn.String() {
			return
		}
		switch e.Event {
		case EventTypePodCreated:
			pd.CreationTime = e.Timestamp
			pd.Pod = e.Object.(*v1.Pod)
		case EventTypePodBound:
			pd.BindTime = e.Timestamp
			pd.NodeName = e.AdditionalProperties["NodeName"]
		case EventTypePodEvicted:
			pd.EvictionTime = e.Timestamp
		case EventTypePodDeleted:
			pd.DeletionTime = e.Timestamp
		}
	})
	return pd
}

func (PodParser) Extract(event auditmodel.Event) ParsedEvent {
	pe := ParsedEvent{
		Timestamp:            event.RequestReceivedTimestamp.Time,
		ObjectType:           ObjectTypePod,
		AdditionalProperties: map[string]string{},
	}
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
		var p v1.Pod
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

func (e PodParser) DescribeQuery(nn types.NamespacedName) string {
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

func (e PodParser) GetQuery(nn types.NamespacedName) string {
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
