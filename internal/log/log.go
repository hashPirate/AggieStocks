package log


type EventLog interface {
	Append(event []byte) (position int64,err error)
	AppendToPartition(partitionKey string,event []byte) (position int64,err error)
	Subscribe(partitionKey string,fromPosition int64) (<-chan LogEntry,func() error,error)
	Close() error
}

type LogEntry struct {
	Position int64
	Payload  []byte
}
