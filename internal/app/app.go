package app

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"

	actionapp "github.com/sieryo/invoice-extractor/internal/app/action"
	"github.com/sieryo/invoice-extractor/internal/app/auth"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	"github.com/sieryo/invoice-extractor/internal/app/bukpot/parsers"
	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
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
	"github.com/sieryo/invoice-extractor/internal/config"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	repository "github.com/sieryo/invoice-extractor/internal/infra/persistence/sqlite"
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

	BuyerRegistryService         *buyer.BuyerRegistryService
	TaxAccountService            *appcashflow.TaxAccountService
	CashflowProfileConfigService *appcashflow.ProfileConfigService
	CashflowBillCategoryService  *appcashflowbill.CategoryAccountService
	CashflowBillProfileService   *appcashflowbill.ProfileConfigService
	BukpotRequestConfigService   *appbukpot.RequestConfigService
	SettingsService              *config.SettingsService

	TemplateRegistryService *template.TemplateRegistryService
	DocumentProcessors      *document.Registry

	JobRunner *jobrunner.JobQueueRunner
	Features  config.FeatureFlags
}

func New(db *sql.DB, logger *slog.Logger, rootDir string, cfg config.Config) *App {
	document.SetFeatureFlags(document.FeatureFlags{
		EnableCashflowXLSX: cfg.Features.EnableCashflowXLSX,
	})

	templateRegistry := template.NewRegistry()
	templateRegistry.Register(seamakeup.NewSeaMakeupTemplate())
	templateRegistry.Register(goodsaletech.NewGoodSaleTechTemplate())
	templateRegistry.Register(giaprima.NewGiaPrimaTemplate())

	fs := filestore.NewLocalFileStore(filepath.Join(rootDir, "storage"))
	invoiceExporter := excel.NewExcelExporter(templateRegistry)

	profileRepo := repository.NewProfileRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	jobRepo := repository.NewJobRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	fileRepo := repository.NewFileRepository(db)
	uploadSessionRepo := repository.NewUploadSessionRepository(db)
	uploadChunkRepo := repository.NewUploadChunkRepository(db)
	documentRepoV2 := repository.NewDocumentRepositoryV2(db)
	collectionHistoryRepo := repository.NewCollectionHistoryRepository(db)
	collectionActionRepo := repository.NewCollectionActionRepository(db)

	authService := auth.NewService(profileRepo, sessionRepo, logger, rootDir)
	settingsService := config.NewSettingsService(rootDir, cfg.Features)
	buyerRegistryService := buyer.NewBuyerRegistryService(rootDir)
	bukpotRequestConfigService := appbukpot.NewRequestConfigService(rootDir)
	invoiceExtractService := invoiceextract.NewInvoiceExtractService(templateRegistry, buyerRegistryService)
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
	collectionService := collection.NewCollectionService(collectionRepo, documentRepoV2, fs)
	fileService := appfile.NewFileService(fs, fileRepo, collectionRepo)

	taxAccountService := appcashflow.NewTaxAccountService(rootDir)
	cashflowProfileConfigService := appcashflow.NewProfileConfigService(rootDir)
	cashflowBillCategoryService := appcashflowbill.NewCategoryAccountService(rootDir)
	cashflowBillProfileService := appcashflowbill.NewProfileConfigService(rootDir)
	templateRegistryService := template.NewTemplateRegistryService(templateRegistry)
	documentRegistry := document.NewRegistry()
	documentRegistry.MustRegister(document.NewPDFInvoiceProcessor(invoiceExtractService, invoiceService, fs))
	documentRegistry.MustRegister(document.NewPDFTaxInvoiceProcessor(taxInvoiceExtractService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.CollectionKindBukpotBPPU, bukpotService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.CollectionKindBukpotBP21, bukpotService, fs))
	documentRegistry.MustRegister(document.NewPDFBukpotProcessor(document.CollectionKindBukpotBPA1, bukpotService, fs))
	documentRegistry.MustRegister(document.NewXLSXBukpotRequestProcessor(fs, rootDir, bukpotRequestConfigService))
	documentRegistry.MustRegister(document.NewXLSXCashflowProcessor(fs, taxAccountService, cashflowBillCategoryService))
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
		taxAccountService,
		cashflowProfileConfigService,
		cashflowBillCategoryService,
		cashflowBillProfileService,
		bukpotRequestConfigService,
		fs,
		1,
	)
	actionService.StartPool(context.Background())

	dispatcher := jobrunner.NewDispatcher()
	invoiceHandler := invoiceextract.NewInvoiceExtractJob(invoiceExtractService, fs, fileRepo)
	renameTaxInvoiceHandler := rename.NewTaxInvoiceRenameJob(renameTaxInvoiceService, fs, fileRepo)

	dispatcher.MustRegister(jobdomain.JobTypeExtractInvoice, invoiceHandler)
	dispatcher.MustRegister(jobdomain.JobTypeRenameTaxInvoice, renameTaxInvoiceHandler)

	jobRunner := jobrunner.NewJobQueueRunner(jobRepo, dispatcher, 2)
	ctx := context.Background()
	jobRunner.StartPool(ctx)

	jobService := jobapp.NewJobService(jobRepo, jobRunner, fs)

	return &App{
		RootDir:                      rootDir,
		AuthService:                  authService,
		CollectionService:            collectionService,
		FileService:                  fileService,
		IngestService:                ingestService,
		ActionService:                actionService,
		JobService:                   jobService,
		InvoiceService:               invoiceService,
		Logger:                       logger,
		FileStore:                    fs,
		JobRunner:                    jobRunner,
		BuyerRegistryService:         buyerRegistryService,
		TaxAccountService:            taxAccountService,
		CashflowProfileConfigService: cashflowProfileConfigService,
		CashflowBillCategoryService:  cashflowBillCategoryService,
		CashflowBillProfileService:   cashflowBillProfileService,
		BukpotRequestConfigService:   bukpotRequestConfigService,
		SettingsService:              settingsService,
		TemplateRegistryService:      templateRegistryService,
		DocumentProcessors:           documentRegistry,
		Features:                     cfg.Features,
	}
}
