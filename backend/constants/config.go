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

package constants

import (
	"github.com/caiflower/common-tools/global/config"
)

// DefaultConfig 基础设施配置（由 etc/default.yaml 加载）
var DefaultConfig config.DefaultConfig

// Config 业务配置（由 etc/config.yaml 加载）
type Config struct {
	GoMaxProcs int `yaml:"go_max_procs" json:"go_max_procs"`
}

// Prop 业务配置实例
var Prop Config

// InitConfig 加载配置文件
func InitConfig() {
	if err := config.LoadDefaultConfig(&DefaultConfig); err != nil {
		panic("failed to load default config: " + err.Error())
	}
	if err := config.LoadYamlFile("config.yaml", &Prop); err != nil {
		panic("failed to load config.yaml: " + err.Error())
	}
}
