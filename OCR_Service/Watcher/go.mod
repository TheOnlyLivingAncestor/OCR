module ocr/ocr_service/watcher

go 1.24.0

require ocr/packages/queue v0.0.0

require (
	github.com/Azure/go-amqp v1.5.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/rabbitmq/rabbitmq-amqp-go-client v1.0.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
)

replace ocr/packages/queue => ../../Packages/Queue
