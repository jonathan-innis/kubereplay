package parser

import (
	"time"

	"github.com/joinnis/kubereplay/pkg/audit"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectParser interface {
	Extract(event audit.Event) ParsedEvent
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

func ParseEvents(events []audit.Event) []ParsedEvent {
	return lo.Filter(lop.Map(events, func(e audit.Event, _ int) ParsedEvent {
		var parser ObjectParser
		switch e.ObjectRef.Resource {
		case "pods":
			parser = Pod{}
		}
		return parser.Extract(e)
	}), func(pe ParsedEvent, _ int) bool { return lo.IsNotEmpty(pe.NamespaceName) })
}
