package object

import (
	"encoding/json"
	"fmt"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	EventTypeNodeCreated = "NodeCreated"
	EventTypeNodeDeleted = "NodeDeleted"
)

type Node struct {
	Node          *v1.Node
	NamespaceName types.NamespacedName
	CreationTime  time.Time
	DeletionTime  time.Time
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
	lop.ForEach(events, func(e ParsedEvent, _ int) {
		if e.ObjectType != ObjectTypeNode {
			return
		}
		if e.NamespaceName.String() != nn.String() {
			return
		}
		switch e.Event {
		case EventTypeNodeCreated:
			n.CreationTime = e.Timestamp
			n.Node = e.Object.(*v1.Node)
		case EventTypePodDeleted:
			n.DeletionTime = e.Timestamp
		}
	})
	return n
}

func (NodeParser) Extract(event auditmodel.Event) ParsedEvent {
	pe := ParsedEvent{
		Timestamp:            event.RequestReceivedTimestamp.Time,
		ObjectType:           ObjectTypeNode,
		AdditionalProperties: map[string]string{},
	}
	switch {
	case event.Verb == "create":
		pe.Event = EventTypeNodeCreated
		var n v1.Node
		lo.Must0(json.Unmarshal(lo.Must(json.Marshal(event.ResponseObject)), &n))
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
| filter verb like /create|delete/
| filter requestURI like "nodes"
| filter requestURI not like "csi" and requestURI not like "cni"
| filter @message like "%s"
`
	return fmt.Sprintf(podQueryTemplate, nn.Name)
}
