package protocol

// HTTPProtocol HTTP 协议
type HTTPProtocol struct{}

func (p *HTTPProtocol) Name() string        { return "http" }
func (p *HTTPProtocol) DisplayName() string  { return "HTTP" }
func (p *HTTPProtocol) Description() string  { return "通过 HTTP REST API 调用远程服务" }
func (p *HTTPProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "url", Label: "请求地址", Type: "string", Required: true, Description: "HTTP 请求 URL"},
			{Name: "method", Label: "请求方法", Type: "select", Required: true, Default: "POST", Options: []string{"GET", "POST", "PUT", "DELETE"}},
			{Name: "headers", Label: "请求头", Type: "textarea", Required: false, Description: "JSON 格式的请求头"},
			{Name: "timeout", Label: "超时时间(秒)", Type: "number", Required: false, Default: "30"},
		},
	}
}

// GRPCProtocol gRPC 协议
type GRPCProtocol struct{}

func (p *GRPCProtocol) Name() string        { return "grpc" }
func (p *GRPCProtocol) DisplayName() string  { return "gRPC" }
func (p *GRPCProtocol) Description() string  { return "通过 gRPC 调用远程服务" }
func (p *GRPCProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "address", Label: "服务地址", Type: "string", Required: true, Description: "gRPC 服务地址 (host:port)"},
			{Name: "service", Label: "服务名", Type: "string", Required: true, Description: "gRPC 服务全限定名"},
			{Name: "method", Label: "方法名", Type: "string", Required: true, Description: "gRPC 方法名"},
			{Name: "timeout", Label: "超时时间(秒)", Type: "number", Required: false, Default: "30"},
		},
	}
}

// LocalProtocol 本地函数协议
type LocalProtocol struct{}

func (p *LocalProtocol) Name() string        { return "local" }
func (p *LocalProtocol) DisplayName() string  { return "Local" }
func (p *LocalProtocol) Description() string  { return "调用本地注册的函数" }
func (p *LocalProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "funcName", Label: "函数名", Type: "string", Required: true, Description: "本地注册的函数名称"},
		},
	}
}

// MCPProtocol MCP 工具调用协议
type MCPProtocol struct{}

func (p *MCPProtocol) Name() string        { return "mcp" }
func (p *MCPProtocol) DisplayName() string  { return "MCP" }
func (p *MCPProtocol) Description() string  { return "通过 MCP 协议调用工具" }
func (p *MCPProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "server", Label: "MCP Server", Type: "string", Required: true, Description: "MCP Server 地址"},
			{Name: "tool", Label: "工具名", Type: "string", Required: true, Description: "要调用的工具名称"},
			{Name: "timeout", Label: "超时时间(秒)", Type: "number", Required: false, Default: "60"},
		},
	}
}
