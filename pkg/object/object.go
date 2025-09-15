package object

import (
	"fmt"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	Get() string
	Describe() string
}

type ObjectParser interface {
	Extract(event auditmodel.Event) ParsedEvent
	Coalesce(types.NamespacedName, []ParsedEvent) Object
	GetQuery(types.NamespacedName) string
	DescribeQuery(types.NamespacedName) string
}

type ParsedEvent struct {
	Timestamp            time.Time
	NamespaceName        types.NamespacedName
	ObjectType           ObjectType
	Object               client.Object
	Event                EventType
	AdditionalProperties map[string]string
}

type ObjectType string

const (
	ObjectTypePod = "Pod"
)

type EventType string

func ParseEvents(events []auditmodel.Event) []ParsedEvent {
	return lo.Filter(lop.Map(events, func(e auditmodel.Event, _ int) ParsedEvent {
		var parser ObjectParser
		switch e.ObjectRef.Resource {
		case "pods":
			parser = PodParser{}
		}
		return parser.Extract(e)
	}), func(pe ParsedEvent, _ int) bool { return lo.IsNotEmpty(pe.NamespaceName) })
}

func NewObjectParserFrom(objectType string) ObjectParser {
	switch objectType {
	case "pod":
		return PodParser{}
	default:
		panic(fmt.Sprintf("invalid object type: %s", objectType))
	}
}
