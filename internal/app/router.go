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
	router.Post("/admin/accounts", browserAuthHandler.CreateAccount)
	router.Put("/admin/accounts/{idUser}", browserAuthHandler.UpdateAccount)
	router.Delete("/admin/accounts/{idUser}", browserAuthHandler.DeleteAccount)
	router.Get("/admin/costumers", browserAuthHandler.Customers)
	router.Post("/admin/costumers", browserAuthHandler.CreateCustomer)
	router.Put("/admin/costumers/{idCostumer}", browserAuthHandler.UpdateCustomer)
	router.Delete("/admin/costumers/{idCostumer}", browserAuthHandler.DeleteCustomer)

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
	})

	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested route does not exist.")
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The HTTP method is not supported for this route.")
	})

	return router
}
