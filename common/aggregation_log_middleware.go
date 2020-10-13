package common

// import (
// 	"log"

// 	"github.com/incognitochain/incognito-chain/common/aggregatelog"
// 	"github.com/pkg/errors"
// )

// var LevelMap = map[string]string{
// 	"DBG": aggregatelog.DEBUG_LEVEL,
// 	"INF": aggregatelog.INFO_LEVEL,
// 	"WRN": aggregatelog.WARNING_LEVEL,
// 	"ERR": aggregatelog.ERROR_LEVEL,
// 	"CRT": aggregatelog.FATAL_LEVEL,
// }

// var ElasticLogService *aggregatelog.LogService
// var SentryLogService *aggregatelog.LogService

// func AggregationLogInit() {
// 	// INIT ELASTIC LOG SERVICE
// 	logService, err := aggregatelog.GetService(aggregatelog.ELASTIC_LOG_SERVICENAME)
// 	if err != nil {
// 		log.Println("GEt elastic service error:", err)
// 	} else {
// 		logParams := map[string]interface{}{
// 			"elastic_url": aggregatelog.ELASTIC_URL,
// 		}
// 		err := logService.InitService(logParams)
// 		if err != nil {
// 			log.Println("aggregation log service init err", err)
// 		} else {
// 			ElasticLogService = logService
// 		}
// 	}
// 	// INIT SENTRY LOG SERVICE
// 	logService, err = aggregatelog.GetService(aggregatelog.SENTRY_LOG_SERVICENAME)
// 	if err != nil {
// 		log.Println("GEt elastic service error:", err)
// 	} else {
// 		logParams := map[string]interface{}{
// 			"DSN": aggregatelog.SENTRY_DSN,
// 		}
// 		err := logService.InitService(logParams)
// 		if err != nil {
// 			log.Println("aggregation log service init err", err)
// 		} else {
// 			SentryLogService = logService
// 		}
// 	}
// }

// func HandleCaptureMessage(message, level string) error {
// 	lvl, ok := LevelMap[level]
// 	if !ok {
// 		return nil
// 	}
// 	if ElasticLogService == nil || SentryLogService == nil {
// 		return nil
// 	}
// 	switch lvl {
// 	case aggregatelog.ERROR_LEVEL:
// 		return ElasticLogService.CaptureError(errors.New(message))
// 	case aggregatelog.INFO_LEVEL:
// 		return ElasticLogService.CaptureMessage(message)
// 	case aggregatelog.DEBUG_LEVEL:
// 		return ElasticLogService.CaptureDebug(message)
// 	case aggregatelog.WARNING_LEVEL:
// 		return ElasticLogService.CaptureWarning(message)
// 	case aggregatelog.FATAL_LEVEL:
// 		return ElasticLogService.CaptureFatal(message)
// 	}
// 	return nil
// }
