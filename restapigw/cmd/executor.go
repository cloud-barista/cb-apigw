package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloud-barista/cb-apigw/restapigw/pkg/api"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/config"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/core"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/errors"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/logging"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/auth"
	ginCors "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/cors/gin"
	httpcache "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/httpcache"
	httpsecure "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/httpsecure/gin"
	ginMetrics "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/metrics/gin"
	opencensus "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/opencensus"
	ginOpencensus "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/opencensus/router/gin"
	ratelimitProxy "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/ratelimit/proxy"
	ratelimit "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/ratelimit/router/gin"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/proxy"
	ginRouter "github.com/cloud-barista/cb-apigw/restapigw/pkg/router/gin"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/server"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/transport/http/client"
	"github.com/gin-gonic/gin"

	// Opencensus 연동을 위한 Exporter 로드 및 초기화
	_ "github.com/cloud-barista/cb-apigw/restapigw/pkg/middlewares/opencensus/exporters/jaeger"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// contextWithSignal - System Interrupt Signal을 반영한 Context 생
func contextWithSignal(ctx context.Context) context.Context {
	newCtx, cancel := context.WithCancel(ctx)
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signals:
			cancel()
			close(signals)
		}
	}()
	return newCtx
}

// newHandlerFactory - Middleware들과 Opencensus 처리를 반영한 Gin Endpoint Handler 생성
func newHandlerFactory(logger logging.Logger, metricsProducer *ginMetrics.Metrics) ginRouter.HandlerFactory {
	// Rate Limit 처리용 RouterHandlerFactory 구성
	handlerFactory := ratelimit.HandlerFactory(ginRouter.EndpointHandler, logger)
	// TODO: JWT Auth, JWT Rejector

	// 임시로 HMAC을 활용한 Auth 인증 처리용 RouteHandlerFactory 구성
	handlerFactory = auth.HandlerFactory(handlerFactory, logger)

	// metricsProducer 활용하는 RouteHandlerFactory 구성
	handlerFactory = metricsProducer.HandlerFactory(handlerFactory, logger)
	// Opencensus를 활용하는 RouteHandlerFactory 구성
	handlerFactory = ginOpencensus.HandlerFactory(handlerFactory, logger)
	return handlerFactory
}

// setupRepository - HTTP Server와 Admin Server 운영에 사용할 API Route 정보 리파지토리 구성
func setupRepository(sConf config.ServiceConfig, log logging.Logger) (api.Repository, error) {
	// API Routing 정보를 관리하는 Repository 구성
	repo, err := api.BuildRepository(sConf.Repository.DSN, sConf.Cluster.UpdateFrequency)
	if nil != err {
		return nil, errors.Wrap(err, "could not build a repository for the database or file or CB-Store")
	}

	return repo, nil
}

// newServer - HTTP Server 및 Admin Server 운영과 연관되는 Middleware 구성을 처리할 서버 생성
func newServer(sConf config.ServiceConfig, logger logging.Logger) *gin.Engine {
	// TODO: Create Server
	engine := gin.Default()
	engine.RedirectTrailingSlash = true
	engine.RedirectFixedPath = true
	engine.HandleMethodNotAllowed = true

	// CORS Middleware 반영
	ginCors.New(sConf.Middleware, engine)

	// HTTPSecure Middleware 반영
	if err := httpsecure.Register(sConf.Middleware, engine); err != nil {
		logger.Warning(err)
	}

	return engine
}

// newProxyFactory - 지정된 BackendFactory를 기반으로 동작하는 ProxyFactory 생성
func newProxyFactory(logger logging.Logger, backendFactory proxy.BackendFactory, metricsProducer *ginMetrics.Metrics) proxy.Factory {
	proxyFactory := proxy.NewDefaultFactory(backendFactory, logger)

	// Metrics 연동 기반의 ProxyFactory 설정
	proxyFactory = metricsProducer.ProxyFactory("pipe", proxyFactory)

	// Opencensus 연동 기반의 ProxyFactory 설정
	proxyFactory = opencensus.ProxyFactory(proxyFactory)
	return proxyFactory
}

// newBackendFactoryWithContext - 지정된 Context 기반에서 활용가능한 middleware들이 적용된 BackendFactory 생성
func newBackendFactoryWithContext(ctx context.Context, logger logging.Logger, metricsProducer *ginMetrics.Metrics) proxy.BackendFactory {
	requestExecutorFactory := func(bConf *config.BackendConfig) client.HTTPRequestExecutor {
		var clientFactory client.HTTPClientFactory
		// TODO: Backend Auth

		// HTTPCache 가 적용된 HTTP Client
		clientFactory = httpcache.NewHTTPClient(bConf)
		// Opencensus 와 연계된 HTTP Request Executor
		return opencensus.HTTPRequestExecutor(clientFactory)
	}

	// Opencensus HTTPRequestExecutor를 사용하는 Base BackendFactory 설정
	backendFactory := func(bConf *config.BackendConfig) proxy.Proxy {
		return proxy.NewHTTPProxyWithHTTPExecutor(bConf, requestExecutorFactory(bConf), bConf.Decoder)
	}

	// TODO: Martian for Backend

	// Backend 호출에 대한 Rate Limit Middleware 설정
	backendFactory = ratelimitProxy.BackendFactory(backendFactory)

	// TODO: Circuit-Breaker for Backend

	// Metrics 연동 기반의 BackendFactory 설정
	backendFactory = metricsProducer.BackendFactory("backend", backendFactory)

	// Opencensus 연동 기반의 BackendFactory 설정
	backendFactory = opencensus.BackendFactory(backendFactory)
	return backendFactory
}

// setupPipe - Router Pipeline 설정 구성
func setupPipe(ctx context.Context, sConf config.ServiceConfig, log *logging.Logger) error {
	// Sets up the Metrics
	// metricsProducer := ginMetrics.SetupAndRun(ctx, sConf, *log)

	// if metricsProducer.Config != nil {
	// 	// Sets up the InfluxDB Client for Metrics
	// 	influxdb.SetupAndRun(ctx, metricsProducer.Config.InfluxDB, func() interface{} { return metricsProducer.Snapshot() }, log)
	// } else {
	// 	log.Warn("Skip the influxdb setup and running because the no metrics configuration or incorrect")
	// }

	// // Sets up the Opencensus
	// if err := opencensus.Setup(ctx, sConf); err != nil {
	// 	log.Fatal(err)
	// }

	// Sets up the Pipeline (Router (Endpoint Handler) - Proxy - Backend)
	// pipeConfig := ginRouter.PipeConfig{
	// 	Engine:         newServer(sConf, *log),
	// 	ProxyFactory:   newProxyFactory(*log, newBackendFactoryWithContext(ctx, *log, metricsProducer), metricsProducer),
	// 	Middlewares:    []gin.HandlerFunc{},
	// 	Logger:         log,
	// 	HandlerFactory: newHandlerFactory(*log, metricsProducer),
	// 	Context:        contextWithSignal(ctx),
	// }

	// PipeConfig 정보를 기준으로 HTTP Server 실행 (Gin Router + Endpoint Handler, Pipeline)
	//pipeConfig.Run(sConf)

	return nil
}

// ===== [ Public Functions ] =====

// SetupAndRun - API Gateway 서비스를 위한 Router 및 Pipeline 구성과 HTTP Server 구동
func SetupAndRun(ctx context.Context, sConf config.ServiceConfig) error {
	// Sets up the Logger (CB-LOG)
	log := logging.NewLogger()

	// API 운영을 위한 라우팅 리파지토리 구성
	repo, err := setupRepository(sConf, *log)
	if nil != err {
		return err
	}
	defer repo.Close()

	// API G/W 및 Admin 운영을 위한 서버 구성 및 실행
	svr := server.New(
		server.WithSystemConfig(&sConf),
		server.WithProvider(repo),
	)
	defer svr.Close()

	ctx = contextWithSignal(ctx)
	svr.StartWithContext(ctx)

	svr.Wait()
	log.Info(core.AppName + " - Shuting down")

	return nil
}
