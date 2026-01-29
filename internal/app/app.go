package app

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/app/auth"
	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/app/collection"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/exporter/excel"
	invoiceextract "github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/rename"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template/goodsaletech"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template/seamakeup"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	jobapp "github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	repository "github.com/sieryo/invoice-extractor/internal/infra/persistence/sqlite"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
	"github.com/sieryo/invoice-extractor/internal/infra/watcher"
)

type App struct {
	RootDir string

	AuthService       *auth.AuthService
	JobService        *job.JobService
	InvoiceService    *invoice.InvoiceService
	CollectionService *collection.CollectionService

	Logger    *slog.Logger
	FileStore file.FileStore

	BuyerRegistry *buyer.Registry
	BuyerStore    *storage.BuyerCSVStore

	JobRunner *jobrunner.JobQueueRunner
}

func New(db *sql.DB, logger *slog.Logger, rootDir string) *App {
	// Path
	csvPath := filepath.Join(rootDir, "buyers.csv")

	// Registry
	templateRegistry := template.NewRegistry()
	templateRegistry.Register(seamakeup.NewSeaMakeupTemplate())
	templateRegistry.Register(goodsaletech.NewGoodSaleTechTemplate())
	buyerRegistry := buyer.NewRegistry()

	// infra
	fs := filestore.NewLocalFileStore(filepath.Join(rootDir, "storage"))
	buyerStore := storage.NewBuyerCSVStore(csvPath)

	watcher := watcher.NewCSVWatcher(
		buyerRegistry,
		buyerStore,
		csvPath,
	)

	go func() {
		if err := watcher.Run(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	invoiceExporter := excel.NewExcelExporter(templateRegistry)

	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	jobRepo := repository.NewJobRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)

	// services
	authService := auth.NewService(userRepo, sessionRepo, logger)
	invoiceExtractService := invoiceextract.NewInvoiceExtractService(templateRegistry, buyerRegistry)
	invoiceService := invoice.NewInvoiceService(invoiceExporter, fs)
	taxInvoiceExtractService := extract.NewTaxInvoiceExtractService()
	renameTaxInvoiceService := rename.NewTaxInvoiceRenameService(taxInvoiceExtractService)
	collectionService := collection.NewCollectionService(collectionRepo)

	// dispatcher & handler
	dispatcher := jobrunner.NewDispatcher()
	invoiceHandler := invoiceextract.NewInvoiceExtractJob(invoiceExtractService, fs)

	renameTaxInvoiceHandler := rename.NewTaxInvoiceRenameJob(renameTaxInvoiceService, fs)

	dispatcher.MustRegister(jobdomain.JobTypeExtractInvoice, invoiceHandler)
	dispatcher.MustRegister(jobdomain.JobTypeRenameTaxInvoice, renameTaxInvoiceHandler)

	jobRunner := jobrunner.NewJobQueueRunner(jobRepo, dispatcher, 2)
	ctx := context.Background()
	jobRunner.StartPool(ctx)

	// job service
	jobService := jobapp.NewJobService(jobRepo, jobRunner)

	return &App{
		RootDir:           rootDir,
		AuthService:       authService,
		CollectionService: collectionService,
		JobService:        jobService,
		InvoiceService:    invoiceService,
		Logger:            logger,
		FileStore:         fs,
		JobRunner:         jobRunner,
		BuyerRegistry:     buyerRegistry,
		BuyerStore:        buyerStore,
	}
}
