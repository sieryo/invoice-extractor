package app

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/sieryo/invoice-extractor/internal/app/auth"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	repository "github.com/sieryo/invoice-extractor/internal/infra/persistence/sqlite"
)

type App struct {
	AuthService *auth.AuthService
	JobService  *job.JobService
	Logger      *slog.Logger

	JobRunner *jobrunner.JobQueueRunner
}

func New(db *sql.DB, logger *slog.Logger) *App {
	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	jobRepo := repository.NewJobRepository(db)

	// services
	authService := auth.NewService(userRepo, sessionRepo, logger)
	invoiceExtractService := extract.NewInvoiceExtractService()

	// dispatcher & handler
	dispatcher := jobrunner.NewDispatcher()
	invoiceHandler := extract.NewInvoiceExtractJob(invoiceExtractService)

	dispatcher.Register("INVOICE_EXTRACT", invoiceHandler)

	jobRunner := jobrunner.NewJobQueueRunner(jobRepo, dispatcher, 5)
	ctx := context.Background()
	jobRunner.StartPool(ctx)

	// job service
	jobService := job.NewJobService(jobRepo, jobRunner)

	return &App{
		AuthService: authService,
		JobService:  jobService,
		Logger:      logger,
		JobRunner:   jobRunner,
	}
}
