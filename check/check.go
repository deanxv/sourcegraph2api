package check

import (
	"sourcegraph2api/common/config"
	logger "sourcegraph2api/common/loggger"
)

func CheckEnvVariable() {
	logger.SysLog("environment variable checking...")

	if config.SGCookie == "" {
		logger.FatalLog("环境变量 SG_COOKIE 未设置")
	}

	logger.SysLog("environment variable check passed.")
}
