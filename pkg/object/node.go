package object

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	EventTypeNodeCreated = "NodeCreated"
	EventTypeNodeUpdated = "NodeUpdated"
	EventTypeNodeDeleted = "NodeDeleted"
)

type Node struct {
	Node            *v1.Node
	NamespaceName   types.NamespacedName
	CreationTime    time.Time
	LastUpdatedTime time.Time
	DeletionTime    time.Time
}

func (n Node) Describe() string {
	panic("implement me")
}

func (e Node) Get() string {
	return string(lo.Must(yaml.Marshal(e.Node)))
}

type NodeParser struct{}

func (NodeParser) Coalesce(nn types.NamespacedName, events []ParsedEvent) Object {
	n := Node{NamespaceName: nn}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	for _, e := range events {
		if e.ObjectType != ObjectTypeNode {
			continue
		}
		if e.NamespaceName.String() != nn.String() {
			continue
		}
		switch e.Event {
		case EventTypeNodeCreated:
			n.CreationTime = e.Timestamp
			n.Node = e.Object.(*v1.Node)
		case EventTypeNodeUpdated:
			n.LastUpdatedTime = e.Timestamp
			n.Node = e.Object.(*v1.Node)
		case EventTypePodDeleted:
			n.DeletionTime = e.Timestamp
		}
	}
	return n
}

func (NodeParser) Extract(event auditmodel.Event) ParsedEvent {
	pe := ParsedEvent{
		Timestamp:            event.RequestReceivedTimestamp.Time,
		ObjectType:           ObjectTypeNode,
		AdditionalProperties: map[string]string{},
	}
	var n v1.Node
	switch {
	case event.Verb == "create":
		pe.Event = EventTypeNodeCreated
		lo.Must0(json.Unmarshal(lo.Must(json.Marshal(event.ResponseObject)), &n))
		n.ManagedFields = nil
		pe.Object = &n
		pe.NamespaceName = types.NamespacedName{Name: event.ObjectRef.Name}
	case event.Verb == "update":
		pe.Event = EventTypeNodeUpdated
		lo.Must0(json.Unmarshal(lo.Must(json.Marshal(event.ResponseObject)), &n))
		n.ManagedFields = nil
		pe.Object = &n
		pe.NamespaceName = types.NamespacedName{Name: event.ObjectRef.Name}
	case event.Verb == "delete":
		pe.Event = EventTypeNodeDeleted
		pe.NamespaceName = types.NamespacedName{Name: event.ObjectRef.Name}
	default:
		return ParsedEvent{}
	}
	return pe
}

func (e NodeParser) DescribeQuery(nn types.NamespacedName) string {
	panic("implement me")
}

func (e NodeParser) GetQuery(nn types.NamespacedName) string {
	podQueryTemplate := `
fields @timestamp, @message
| filter @logStream like "apiserver"
| filter verb like /create|delete|update/
| filter requestURI like "nodes"
| filter requestURI not like "csi" and requestURI not like "cni"
| filter @message like "%s"
`
	return fmt.Sprintf(podQueryTemplate, nn.Name)
}
