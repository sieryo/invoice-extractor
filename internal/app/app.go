package app

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"path/filepath"

	actionapp "github.com/sieryo/invoice-extractor/internal/app/action"
	"github.com/sieryo/invoice-extractor/internal/app/auth"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	"github.com/sieryo/invoice-extractor/internal/app/bukpot/parsers"
	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/app/collection"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	appfile "github.com/sieryo/invoice-extractor/internal/app/file"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/exporter/excel"
	invoiceextract "github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/rename"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	giaprima "github.com/sieryo/invoice-extractor/internal/app/invoice/template/giaprimaindonesia"
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
	FileService       *appfile.FileService
	IngestService     *ingest.IngestService
	ActionService     *actionapp.Service

	Logger    *slog.Logger
	FileStore file.FileStore

	BuyerRegistry        *buyer.Registry
	BuyerStore           *storage.BuyerCSVStore
	BuyerRegistryService *buyer.BuyerRegistryService

	TemplateRegistryService *template.TemplateRegistryService
	DocumentProcessors      *document.Registry

	JobRunner *jobrunner.JobQueueRunner
}

func New(db *sql.DB, logger *slog.Logger, rootDir string) *App {
	// Path
	csvPath := filepath.Join(rootDir, "buyers.csv")

	// Registry
	templateRegistry := template.NewRegistry()
	templateRegistry.Register(seamakeup.NewSeaMakeupTemplate())
	templateRegistry.Register(goodsaletech.NewGoodSaleTechTemplate())
	templateRegistry.Register(giaprima.NewGiaPrimaTemplate())
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
	fileRepo := repository.NewFileRepository(db)
	uploadSessionRepo := repository.NewUploadSessionRepository(db)
	uploadChunkRepo := repository.NewUploadChunkRepository(db)
	documentRepoV2 := repository.NewDocumentRepositoryV2(db)
	collectionHistoryRepo := repository.NewCollectionHistoryRepository(db)
	collectionActionRepo := repository.NewCollectionActionRepository(db)

	// services
	authService := auth.NewService(userRepo, sessionRepo, logger)
	invoiceExtractService := invoiceextract.NewInvoiceExtractService(templateRegistry, buyerRegistry)
	invoiceService := invoice.NewInvoiceService(invoiceExporter, fs)
	taxInvoiceExtractService := extract.NewTaxInvoiceExtractService()
	renameTaxInvoiceService := rename.NewTaxInvoiceRenameService(taxInvoiceExtractService)
	bukpotService := appbukpot.NewService(appbukpot.NewPDFToolExtractor())
	if err := bukpotService.RegisterParser(parsers.NewBPPUParser()); err != nil {
		panic(err)
	}
	if err := bukpotService.RegisterParser(parsers.NewBP21Parser()); err != nil {
		panic(err)
	}
	if err := bukpotService.RegisterParser(parsers.NewBPA1Parser()); err != nil {
		panic(err)
	}
	collectionService := collection.NewCollectionService(collectionRepo, fs)
	fileService := appfile.NewFileService(fs, fileRepo, collectionRepo)

	// registry services
	buyerRegistryService := buyer.NewBuyerRegistryService(buyerRegistry, buyerStore, rootDir)
	templateRegistryService := template.NewTemplateRegistryService(templateRegistry)
	documentRegistry := document.NewRegistry()
	documentRegistry.MustRegister(document.NewPDFInvoiceProcessor(invoiceExtractService, invoiceService, fs))
	documentRegistry.MustRegister(document.NewPDFTaxInvoiceProcessor(taxInvoiceExtractService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.DocumentTypePDFBukpotBPPU, bukpotService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.DocumentTypePDFBukpotBP21, bukpotService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.DocumentTypePDFBukpotBPA1, bukpotService, fs))
	ingestService := ingest.NewIngestService(
		uploadSessionRepo,
		uploadChunkRepo,
		documentRepoV2,
		collectionHistoryRepo,
		collectionRepo,
		fs,
		documentRegistry,
		1,
	)
	ingestService.StartPool(context.Background())
	actionService := actionapp.NewService(
		collectionActionRepo,
		collectionRepo,
		documentRegistry,
		buyerRegistryService,
		1,
	)
	actionService.StartPool(context.Background())

	// dispatcher & handler
	dispatcher := jobrunner.NewDispatcher()
	invoiceHandler := invoiceextract.NewInvoiceExtractJob(invoiceExtractService, fs, fileRepo)

	renameTaxInvoiceHandler := rename.NewTaxInvoiceRenameJob(renameTaxInvoiceService, fs, fileRepo)

	dispatcher.MustRegister(jobdomain.JobTypeExtractInvoice, invoiceHandler)
	dispatcher.MustRegister(jobdomain.JobTypeRenameTaxInvoice, renameTaxInvoiceHandler)

	jobRunner := jobrunner.NewJobQueueRunner(jobRepo, dispatcher, 2)
	ctx := context.Background()
	jobRunner.StartPool(ctx)

	// job service
	jobService := jobapp.NewJobService(jobRepo, jobRunner, fs)

	return &App{
		RootDir:                 rootDir,
		AuthService:             authService,
		CollectionService:       collectionService,
		FileService:             fileService,
		IngestService:           ingestService,
		ActionService:           actionService,
		JobService:              jobService,
		InvoiceService:          invoiceService,
		Logger:                  logger,
		FileStore:               fs,
		JobRunner:               jobRunner,
		BuyerRegistry:           buyerRegistry,
		BuyerStore:              buyerStore,
		BuyerRegistryService:    buyerRegistryService,
		TemplateRegistryService: templateRegistryService,
		DocumentProcessors:      documentRegistry,
	}
}
