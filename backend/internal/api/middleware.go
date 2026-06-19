/*
 * Copyright 2026 caiflower Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/common-tools/web/app"
)

// corsMiddleware CORS 跨域中间件
func corsMiddleware(c context.Context, ctx *app.RequestContext) {
	ctx.SetHeader("Access-Control-Allow-Origin", "*")
	ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	ctx.SetHeader("Access-Control-Max-Age", "86400")

	if ctx.GetMethod() == "OPTIONS" {
		ctx.Abort()
		return
	}
	ctx.Next(c)
}

// requestLogMiddleware 请求日志中间件
func requestLogMiddleware(c context.Context, ctx *app.RequestContext) {
	start := time.Now()
	ctx.Next(c)
	duration := time.Since(start)

	logger.Info(fmt.Sprintf("[HTTP] %s %s %d %s",
		ctx.GetMethod(),
		ctx.GetPath(),
		ctx.GetStatusCode(),
		duration.String(),
	))
}

// recoveryMiddleware 错误恢复中间件
func recoveryMiddleware(c context.Context, ctx *app.RequestContext) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("[PANIC] %s %s: %v", ctx.GetMethod(), ctx.GetPath(), err))
			ctx.AbortWithMsg("Internal Server Error", 500)
		}
	}()
	ctx.Next(c)
}
