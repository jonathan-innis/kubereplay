package audit

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Provider interface {
	GetEvents(context.Context, time.Duration, time.Duration, types.NamespacedName) ([]Event, error)
}

type Event struct {
	Kind                     string                 `json:"kind"`
	APIVersion               string                 `json:"apiVersion"`
	Level                    string                 `json:"level"`
	AuditID                  string                 `json:"auditID"`
	Stage                    string                 `json:"stage"`
	RequestURI               string                 `json:"requestURI"`
	Verb                     string                 `json:"verb"`
	User                     User                   `json:"user"`
	ObjectRef                *ObjectReference       `json:"objectRef,omitempty"`
	ResponseStatus           *metav1.Status         `json:"responseStatus,omitempty"`
	RequestObject            map[string]interface{} `json:"requestObject,omitempty"`
	ResponseObject           map[string]interface{} `json:"responseObject,omitempty"`
	RequestReceivedTimestamp metav1.Time            `json:"requestReceivedTimestamp"`
	StageTimestamp           metav1.Time            `json:"stageTimestamp"`
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
