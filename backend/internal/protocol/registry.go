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

package protocol

import (
	"fmt"
	"sync"
)

// ConfigSchema 协议配置 Schema（用于前端动态渲染配置表单）
type ConfigSchema struct {
	Fields []ConfigField `json:"fields"`
}

// ConfigField 单个配置字段
type ConfigField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // string, number, boolean, select, textarea
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"` // for select type
}

// ProtocolFactory 协议工厂接口
type ProtocolFactory interface {
	// Name 协议唯一标识
	Name() string
	// DisplayName 协议显示名称
	DisplayName() string
	// Description 协议描述
	Description() string
	// ConfigSchema 返回协议配置 Schema
	ConfigSchema() ConfigSchema
}

// ProtocolInfo 协议信息（用于 API 响应）
type ProtocolInfo struct {
	Name         string       `json:"name"`
	DisplayName  string       `json:"displayName"`
	Description  string       `json:"description"`
	ConfigSchema ConfigSchema `json:"configSchema"`
}

// Registry 协议注册中心
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProtocolFactory
}

// NewRegistry 创建协议注册中心
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProtocolFactory),
	}
}

// Register 注册协议工厂
func (r *Registry) Register(factory ProtocolFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[factory.Name()] = factory
}

// Get 根据名称获取协议工厂
func (r *Registry) Get(name string) (ProtocolFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("protocol %q not found", name)
	}
	return f, nil
}

// List 列出所有已注册的协议
func (r *Registry) List() []ProtocolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ProtocolInfo, 0, len(r.factories))
	for _, f := range r.factories {
		result = append(result, ProtocolInfo{
			Name:         f.Name(),
			DisplayName:  f.DisplayName(),
			Description:  f.Description(),
			ConfigSchema: f.ConfigSchema(),
		})
	}
	return result
}

// RegisterBuiltinProtocols 注册所有内置协议
func RegisterBuiltinProtocols(registry *Registry) {
	registry.Register(&HTTPProtocol{})
	registry.Register(&GRPCProtocol{})
	registry.Register(&LocalProtocol{})
	registry.Register(&MCPProtocol{})
	registry.Register(&RemoteFuncProtocol{})
}
