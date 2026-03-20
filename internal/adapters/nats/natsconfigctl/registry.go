package natsconfigctl

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

type Registry struct {
	CreateDraft                  natskit.ControlSpec
	GetConfig                    natskit.ControlSpec
	GetActive                    natskit.ControlSpec
	ListActiveRuntimeProjections natskit.ControlSpec
	ListActiveIngestionBindings  natskit.ControlSpec
	ListConfigs                  natskit.ControlSpec
	ValidateDraft                natskit.ControlSpec
	ValidateConfig               natskit.ControlSpec
	CompileConfig                natskit.ControlSpec
	ActivateConfig               natskit.ControlSpec
	DraftCreated                 natskit.EventSpec
	Validated                    natskit.EventSpec
	Compiled                     natskit.EventSpec
	Activated                    natskit.EventSpec
	Deactivated                  natskit.EventSpec
	IngestionRuntimeChanged      natskit.EventSpec
	Archived                     natskit.EventSpec
	Rejected                     natskit.EventSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "CONFIGCTL_EVENTS",
		Subjects: []string{"configctl.events.config.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   24 * time.Hour,
		MaxBytes: 256 * 1024 * 1024,
	}

	return Registry{
		CreateDraft: natskit.ControlSpec{
			Subject:     "configctl.control.create_draft",
			RequestType: "configctl.command.create_draft",
			ReplyType:   "configctl.reply.create_draft",
			QueueGroup:  "configctl.control",
		},
		GetConfig: natskit.ControlSpec{
			Subject:     "configctl.control.get_config",
			RequestType: "configctl.query.get_config",
			ReplyType:   "configctl.reply.get_config",
			QueueGroup:  "configctl.control",
		},
		GetActive: natskit.ControlSpec{
			Subject:     "configctl.control.get_active",
			RequestType: "configctl.query.get_active",
			ReplyType:   "configctl.reply.get_active",
			QueueGroup:  "configctl.control",
		},
		ListActiveRuntimeProjections: natskit.ControlSpec{
			Subject:     "configctl.control.list_active_runtime_projections",
			RequestType: "configctl.query.list_active_runtime_projections",
			ReplyType:   "configctl.reply.list_active_runtime_projections",
			QueueGroup:  "configctl.control",
		},
		ListActiveIngestionBindings: natskit.ControlSpec{
			Subject:     "configctl.control.list_active_ingestion_bindings",
			RequestType: "configctl.query.list_active_ingestion_bindings",
			ReplyType:   "configctl.reply.list_active_ingestion_bindings",
			QueueGroup:  "configctl.control",
		},
		ListConfigs: natskit.ControlSpec{
			Subject:     "configctl.control.list_configs",
			RequestType: "configctl.query.list_configs",
			ReplyType:   "configctl.reply.list_configs",
			QueueGroup:  "configctl.control",
		},
		ValidateDraft: natskit.ControlSpec{
			Subject:     "configctl.control.validate_draft",
			RequestType: "configctl.command.validate_draft",
			ReplyType:   "configctl.reply.validate_draft",
			QueueGroup:  "configctl.control",
		},
		ValidateConfig: natskit.ControlSpec{
			Subject:     "configctl.control.validate_config",
			RequestType: "configctl.command.validate_config",
			ReplyType:   "configctl.reply.validate_config",
			QueueGroup:  "configctl.control",
		},
		CompileConfig: natskit.ControlSpec{
			Subject:     "configctl.control.compile_config",
			RequestType: "configctl.command.compile_config",
			ReplyType:   "configctl.reply.compile_config",
			QueueGroup:  "configctl.control",
		},
		ActivateConfig: natskit.ControlSpec{
			Subject:     "configctl.control.activate_config",
			RequestType: "configctl.command.activate_config",
			ReplyType:   "configctl.reply.activate_config",
			QueueGroup:  "configctl.control",
		},
		DraftCreated: natskit.EventSpec{
			Subject: "configctl.events.config.draft_created",
			Type:    "configctl.event.config.draft_created",
			Stream:  eventStream,
		},
		Validated: natskit.EventSpec{
			Subject: "configctl.events.config.validated",
			Type:    "configctl.event.config.validated",
			Stream:  eventStream,
		},
		Compiled: natskit.EventSpec{
			Subject: "configctl.events.config.compiled",
			Type:    "configctl.event.config.compiled",
			Stream:  eventStream,
		},
		Activated: natskit.EventSpec{
			Subject: "configctl.events.config.activated",
			Type:    "configctl.event.config.activated",
			Stream:  eventStream,
		},
		Deactivated: natskit.EventSpec{
			Subject: "configctl.events.config.deactivated",
			Type:    "configctl.event.config.deactivated",
			Stream:  eventStream,
		},
		IngestionRuntimeChanged: natskit.EventSpec{
			Subject: "configctl.events.config.ingestion_runtime_changed",
			Type:    "configctl.event.config.ingestion_runtime_changed",
			Stream:  eventStream,
		},
		Archived: natskit.EventSpec{
			Subject: "configctl.events.config.archived",
			Type:    "configctl.event.config.archived",
			Stream:  eventStream,
		},
		Rejected: natskit.EventSpec{
			Subject: "configctl.events.config.rejected",
			Type:    "configctl.event.config.rejected",
			Stream:  eventStream,
		},
	}
}

func IngestBindingConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "ingest-binding-watcher",
		Event: natskit.EventSpec{
			Subject: "configctl.events.config.ingestion_runtime_changed",
			Type:    "configctl.event.config.ingestion_runtime_changed",
			Stream: natskit.StreamSpec{
				Name: "CONFIGCTL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

func DeriveBindingConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "derive-binding-watcher",
		Event: natskit.EventSpec{
			Subject: "configctl.events.config.ingestion_runtime_changed",
			Type:    "configctl.event.config.ingestion_runtime_changed",
			Stream: natskit.StreamSpec{
				Name: "CONFIGCTL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
