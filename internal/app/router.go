package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/cliapi"
	"github.com/idelium/idelium-api-go/internal/health"
	"github.com/idelium/idelium-api-go/internal/httpx"
	"github.com/idelium/idelium-api-go/internal/identity"
	"github.com/idelium/idelium-api-go/internal/legacyapikeys"
	"github.com/idelium/idelium-api-go/internal/platforms"
	"github.com/idelium/idelium-api-go/internal/serviceaccounts"
)

// NewRouter builds the API router and common middleware chain.
func NewRouter(
	logger *slog.Logger,
	checker health.Checker,
	info buildinfo.Info,
	catalogRepository platforms.CatalogRepository,
	legacyKeyRepository auth.LegacyKeyRepository,
	testCycleRepository cliapi.TestCycleRepository,
	performedCycleRepository cliapi.PerformedCycleRepository,
	testRepository cliapi.TestRepository,
	performedTestRepository cliapi.PerformedTestRepository,
	performedStepRepository cliapi.PerformedStepRepository,
	stepRepository cliapi.StepRepository,
	pluginRepository cliapi.PluginRepository,
	environmentRepository cliapi.EnvironmentRepository,
	browserAuthRepository browserauth.Repository,
) http.Handler {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID)
	router.Use(httpx.SecureHeaders)
	router.Use(httpx.AccessLogger(logger))
	router.Use(httpx.Recoverer(logger))

	healthHandler := health.NewHandler(checker, info)
	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)
	browserAuthHandler := browserauth.NewHandler(browserAuthRepository, browserAuthRepository, logger)
	router.Get("/sanctum/csrf-cookie", browserAuthHandler.CSRF)
	router.Post("/login", browserAuthHandler.Login)
	router.Post("/logout", browserAuthHandler.Logout)
	router.Get("/user", browserAuthHandler.CurrentUser)
	router.Get("/me/capabilities", browserAuthHandler.Capabilities)
	router.Get("/menu/header", browserAuthHandler.Header)
	router.Put("/menu/header/{idCostumer}", browserAuthHandler.ChangeCustomer)
	router.Get("/menu/sidebar", browserAuthHandler.Sidebar)
	router.Get("/admin/roles", browserAuthHandler.Roles)
	router.Get("/admin/profile", browserAuthHandler.Profile)
	router.Put("/admin/profile", browserAuthHandler.UpdateProfile)
	router.Get("/admin/accounts", browserAuthHandler.Accounts)
	router.Get("/admin/projects", browserAuthHandler.Projects)
	router.Get("/admin/projects/{idProject}", browserAuthHandler.ShowProject)
	router.Post("/admin/projects", browserAuthHandler.CreateProject)
	router.Put("/admin/projects/{idProject}", browserAuthHandler.UpdateProject)
	router.Delete("/admin/projects/{idProject}", browserAuthHandler.DeleteProject)
	router.Get("/admin/agents", browserAuthHandler.Agents)
	router.Put("/admin/agents/{agentRegistration}/status", browserAuthHandler.UpdateAgentStatus)
	router.Post("/admin/accounts", browserAuthHandler.CreateAccount)
	router.Put("/admin/accounts/{idUser}", browserAuthHandler.UpdateAccount)
	router.Delete("/admin/accounts/{idUser}", browserAuthHandler.DeleteAccount)
	router.Get("/admin/costumers", browserAuthHandler.Customers)
	router.Post("/admin/costumers", browserAuthHandler.CreateCustomer)
	router.Put("/admin/costumers/{idCostumer}", browserAuthHandler.UpdateCustomer)
	router.Delete("/admin/costumers/{idCostumer}", browserAuthHandler.DeleteCustomer)
	router.Get("/admin/testcycles/{idProject}", browserAuthHandler.TestCycles)
	router.Get("/admin/testcycles/{idProject}/{testcycle}", browserAuthHandler.ShowTestCycle)
	router.Put("/admin/testcycles/{idProject}/{testcycle}", browserAuthHandler.UpdateTestCycle)
	router.Post("/admin/testcycles", browserAuthHandler.CreateTestCycle)
	router.Post("/admin/steps/{idProject}/updateorder", browserAuthHandler.ReorderSteps)
	router.Get("/admin/steps/{idProject}", browserAuthHandler.Steps)
	router.Get("/admin/steps/{idProject}/{step}", browserAuthHandler.ShowStep)
	router.Post("/admin/steps", browserAuthHandler.CreateStep)
	router.Put("/admin/steps/{idProject}/{step}", browserAuthHandler.UpdateStep)
	router.Delete("/admin/steps/{idProject}/{step}", browserAuthHandler.DeleteStep)
	router.Get("/admin/tests/{idProject}", browserAuthHandler.Tests)
	router.Get("/admin/tests/{idProject}/{test}", browserAuthHandler.ShowTest)
	router.Put("/admin/tests/{idProject}/{test}", browserAuthHandler.UpdateTest)
	router.Post("/admin/tests", browserAuthHandler.CreateTest)
	router.Post("/admin/importtest", browserAuthHandler.ImportTest)
	router.Get("/admin/testcyclesperfomed/{idTestCyclePerformed}", browserAuthHandler.PerformedCycles)
	router.Get("/admin/testsperfomed/{idTestPerformed}", browserAuthHandler.PerformedTests)
	router.Get("/admin/stepsperfomed/{idTestPerformed}", browserAuthHandler.PerformedSteps)
	router.Post("/admin/result-exports", browserAuthHandler.CreateResultExport)
	router.Get("/admin/result-exports/{resultExport}", browserAuthHandler.ShowResultExport)
	router.Get("/admin/result-exports/{resultExport}/download", browserAuthHandler.DownloadResultExport)
	router.Get("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts", browserAuthHandler.ArtifactDescriptors)
	router.Get("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}", browserAuthHandler.ShowArtifactDescriptor)
	router.Post("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive", browserAuthHandler.ArchiveArtifact)
	router.Post("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker", browserAuthHandler.MarkArtifactDeleted)
	router.Put("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold", browserAuthHandler.SetArtifactLegalHold)
	router.Post("/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore", browserAuthHandler.RestoreArtifact)
	router.Post("/admin/grid/query-snapshots", browserAuthHandler.CreateGridQuerySnapshot)
	router.Post("/admin/grid/bulk-jobs", browserAuthHandler.CreateGridBulkJob)
	router.Get("/admin/grid/bulk-jobs/{jobId}", browserAuthHandler.ShowGridBulkJob)
	router.Get("/admin/grid/bulk-jobs/{jobId}/export", browserAuthHandler.ExportGridBulkJob)
	router.Get("/admin/projects/{idProject}/integrations", browserAuthHandler.IntegrationEndpoints)
	router.Post("/admin/projects/{idProject}/integrations", browserAuthHandler.CreateIntegrationEndpoint)
	router.Post("/admin/projects/{idProject}/integrations/{integrationEndpoint}/test", browserAuthHandler.TestIntegrationEndpoint)
	router.Put("/admin/projects/{idProject}/integrations/{integrationEndpoint}/status", browserAuthHandler.UpdateIntegrationEndpointStatus)
	router.Post("/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret", browserAuthHandler.RotateIntegrationEndpointSecret)
	router.Get("/admin/projects/{idProject}/integration-deliveries", browserAuthHandler.IntegrationDeliveries)
	router.Post("/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay", browserAuthHandler.ReplayIntegrationDelivery)
	router.Get("/audit-events", browserAuthHandler.AuditEvents)
	router.Get("/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}", browserAuthHandler.AssetImpact)
	router.Get("/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}", browserAuthHandler.AssetVersions)
	router.Get("/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}", browserAuthHandler.DiffAssetVersions)
	router.Post("/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events", browserAuthHandler.TransitionAssetVersionReview)
	router.Get("/admin/projects/{idProject}/asset-versions/{assetVersion}", browserAuthHandler.ShowAssetVersion)
	router.Get("/admin/projects/{idProject}/parallel-runs", browserAuthHandler.ParallelRuns)
	router.Post("/admin/projects/{idProject}/parallel-runs", browserAuthHandler.CreateParallelRun)
	router.Post("/admin/projects/{idProject}/parallel-runs/matrix", browserAuthHandler.CreateParallelRunMatrix)
	router.Get("/admin/projects/{idProject}/parallel-runs/{parallelRun}", browserAuthHandler.ShowParallelRun)
	router.Post("/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim", browserAuthHandler.ClaimParallelRun)
	router.Post("/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat", browserAuthHandler.HeartbeatParallelRunWorker)
	router.Post("/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel", browserAuthHandler.CancelParallelRun)

	platformHandler := platforms.NewHandler(catalogRepository, logger)
	router.Get("/admin/platforms/types", platformHandler.Types)
	router.Get("/admin/platforms/status", platformHandler.Statuses)
	router.Get("/admin/platforms/locations", platformHandler.Locations)
	router.Get("/admin/platforms/brands", platformHandler.Brands)
	router.Get("/admin/platforms/models/{idBrand}", platformHandler.Models)
	router.Get("/admin/platforms/os/{idType}", platformHandler.OperatingSystems)
	router.Get("/admin/platforms/osversion/{idOs}", platformHandler.OperatingSystemVersions)
	router.Get("/admin/platforms/browsers/{idOs}", platformHandler.Browsers)
	router.Get("/admin/platforms/browserversions/{idBrowser}", platformHandler.BrowserVersions)
	router.Get("/admin/platforms/manageplatforms/{type}", platformHandler.ManagedPlatforms)
	router.Get("/admin/launch/targets/{idProject}", platformHandler.LaunchTargets)

	identityHandler := identity.NewHandler(logger)
	router.Get("/admin/identity/providers", identityHandler.Providers)
	router.Post("/admin/identity/providers", identityHandler.Providers)
	router.Put("/admin/identity/accounts/{user}/break-glass", identityHandler.BreakGlass)
	router.Post("/admin/identity/accounts/{user}/break-glass/test", identityHandler.BreakGlassTest)
	router.Post("/admin/identity/providers/{identityProvider}/scim/users", identityHandler.SCIMUsers)
	router.Post("/admin/profile/mfa/enroll", identityHandler.MFAEnroll)
	router.Post("/admin/profile/mfa/confirm", identityHandler.MFAConfirm)
	router.Post("/admin/profile/mfa/step-up", identityHandler.MFAStepUp)
	router.Post("/oidc/token-exchange", identityHandler.OIDCTokenExchange)
	router.Post("/sso/{identityProvider}/start", identityHandler.SSOStart)
	router.Post("/sso/{identityProvider}/oidc/callback", identityHandler.OIDCCallback)
	router.Post("/sso/{identityProvider}/saml/callback", identityHandler.SAMLCallback)

	legacyAPIKeyHandler := legacyapikeys.NewHandler(logger)
	router.Get("/admin/apikey", legacyAPIKeyHandler.Show)
	router.Head("/admin/apikey", legacyAPIKeyHandler.Show)
	router.Put("/admin/apikey", legacyAPIKeyHandler.Replace)

	serviceAccountHandler := serviceaccounts.NewHandler(logger)
	router.Get("/admin/service-accounts", serviceAccountHandler.Index)
	router.Post("/admin/service-accounts", serviceAccountHandler.Store)
	router.Post("/admin/service-accounts/{serviceAccount}/revoke", serviceAccountHandler.Revoke)

	cliHandler := cliapi.NewHandler(testCycleRepository, performedCycleRepository, testRepository, performedTestRepository, performedStepRepository, stepRepository, pluginRepository, environmentRepository, logger)
	cliAuthenticator := auth.NewLegacyKeyAuthenticator(legacyKeyRepository, logger)
	router.Group(func(router chi.Router) {
		router.Use(cliAuthenticator.Middleware)
		router.Post("/ideliumcl/testcycle", cliHandler.CreatePerformedCycle)
		router.Post("/ideliumcl/agents/register", browserAuthHandler.CLRegisterAgent)
		router.Put("/ideliumcl/testcycle", cliHandler.UpdatePerformedCycle)
		router.Post("/ideliumcl/test", cliHandler.CreatePerformedTest)
		router.Put("/ideliumcl/test", cliHandler.UpdatePerformedTest)
		router.Post("/ideliumcl/step", cliHandler.CreatePerformedStep)
		router.Put("/ideliumcl/step", cliHandler.UpdatePerformedStep)
		router.Get("/ideliumcl/testcycle/{idTestCycle}", cliHandler.TestCycle)
		router.Get("/ideliumcl/test/{idTest}", cliHandler.Test)
		router.Get("/ideliumcl/step/{idStep}", cliHandler.Step)
		router.Get("/ideliumcl/plugins/{idProject}", cliHandler.Plugins)
		router.Get("/ideliumcl/plugin/{idPlugin}", cliHandler.Plugin)
		router.Get("/ideliumcl/environments/{idProject}", cliHandler.Environments)
		router.Get("/ideliumcl/environment/{idEnvironment}", cliHandler.Environment)
		router.Get("/ideliumcl/projects/{idProject}/parallel-runs", browserAuthHandler.CLParallelRuns)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs", browserAuthHandler.CLCreateParallelRun)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/matrix", browserAuthHandler.CLCreateParallelRunMatrix)
		router.Get("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}", browserAuthHandler.CLShowParallelRun)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim", browserAuthHandler.CLClaimParallelRun)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat", browserAuthHandler.CLHeartbeatParallelRunWorker)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel", browserAuthHandler.CLCancelParallelRun)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens", browserAuthHandler.CLIssueParallelRunToken)
		router.Post("/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke", browserAuthHandler.CLRevokeParallelRunToken)
		router.Post("/ideliumrunner/claim", browserAuthHandler.RunnerClaim)
		router.Post("/ideliumrunner/heartbeat", browserAuthHandler.RunnerHeartbeat)
		router.Put("/ideliumrunner/worker", browserAuthHandler.RunnerUpdateWorker)
	})

	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested route does not exist.")
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The HTTP method is not supported for this route.")
	})

	return router
}
