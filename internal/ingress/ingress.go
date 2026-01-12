package ingress

import (
	"encoding/json"
	"net/http"
	"time"

	"eon/platform/internal/event"
	"eon/platform/internal/log"
)

// handler handles the http ingress   
type Handler struct {
	Log       log.EventLog
	Partition string 
}

//v1 events endpoint   
type IngressResponse struct {
	EventID   string `json:"event_id"`
	Position  int64  `json:"position"`
	AcceptedAt string `json:"accepted_at"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter,r *http.Request) {
	if r.Method!=http.MethodPost {
		http.Error(w,"method not allowed",http.StatusMethodNotAllowed)
		return
	}
	var req event.IngressRequest
	if err:=json.NewDecoder(r.Body).Decode(&req); err!=nil {
		http.Error(w,"invalid JSON: "+err.Error(),http.StatusBadRequest)
		return
	}
	if req.IdempotencyKey==""||req.EntityKey =="" ||req.Type=="" {
		http.Error(w,"idempotency_key,entity_key,and type are required",http.StatusBadRequest)
		return
	}
	ev:=event.NewEvent(req.IdempotencyKey,req.EntityKey,req.Type,req.Version,req.Payload)
	body,err:=json.Marshal(ev)
	if err!=nil {
		http.Error(w,"serialization error :(............",http.StatusInternalServerError)
		return
	}
	partition:=h.Partition
	if partition=="" {
		partition=ev.EntityKey
	}
	pos,err:=h.Log.AppendToPartition(partition,body)
	if err!=nil {
		http.Error(w,"append failed: "+err.Error(),http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusAccepted)
	_=json.NewEncoder(w).Encode(IngressResponse{
		EventID:    ev.EventID,
		Position:   pos,
		AcceptedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
