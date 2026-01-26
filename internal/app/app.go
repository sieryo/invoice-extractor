package app

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/app/auth"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template/seamakeup"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	repository "github.com/sieryo/invoice-extractor/internal/infra/persistence/sqlite"
)

type App struct {
	AuthService *auth.AuthService
	JobService  *job.JobService
	Logger      *slog.Logger
	FileStore   filestore.FileStore

	JobRunner *jobrunner.JobQueueRunner
}

func New(db *sql.DB, logger *slog.Logger, rootDir string) *App {
	// Registry
	templateRegistry := template.NewRegistry()
	templateRegistry.Register(seamakeup.NewSeaMakeupTemplate())

	// infra
	fs := filestore.NewLocalFileStore(filepath.Join(rootDir, "storage"))

	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	jobRepo := repository.NewJobRepository(db)

	// services
	authService := auth.NewService(userRepo, sessionRepo, logger)
	invoiceExtractService := extract.NewInvoiceExtractService(templateRegistry)

	// dispatcher & handler
	dispatcher := jobrunner.NewDispatcher()
	invoiceHandler := extract.NewInvoiceExtractJob(invoiceExtractService, fs)

	dispatcher.Register("INVOICE_EXTRACT", invoiceHandler)

	jobRunner := jobrunner.NewJobQueueRunner(jobRepo, dispatcher, 2)
	ctx := context.Background()
	jobRunner.StartPool(ctx)

	// job service
	jobService := job.NewJobService(jobRepo, jobRunner)

	return &App{
		AuthService: authService,
		JobService:  jobService,
		Logger:      logger,
		FileStore:   fs,
		JobRunner:   jobRunner,
	}
}
