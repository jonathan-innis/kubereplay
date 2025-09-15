package provider

import (
	"context"
	"time"

	auditmodel "github.com/joinnis/kubereplay/pkg/audit/model"
	"github.com/joinnis/kubereplay/pkg/object"
	"k8s.io/apimachinery/pkg/types"
)

type Provider interface {
	GetEvents(ctx context.Context, parser object.ObjectParser, cmdType string, start, end time.Duration, nn types.NamespacedName) ([]auditmodel.Event, error)
}
