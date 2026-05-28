package persistence

import "github.com/useryege/athena/internal/application/model"

type ProjectReport = model.ProjectReport
type SimulateResult = model.SimulateResult

type Publisher = PersistenceEventPublisher
type Writer = PersistenceEventWriter
