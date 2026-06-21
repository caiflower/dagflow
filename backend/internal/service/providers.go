package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/caiflower/dagflow/internal/node_registry"
	remote_executor2 "github.com/caiflower/dagflow/internal/protocol/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

var nodeRegistry *node_registry.NodeRegistry
var remoteExecutorPool *remote_executor2.ConnPool

func SetNodeRegistry(r *node_registry.NodeRegistry) {
	nodeRegistry = r
}

func SetRemoteExecutorPool(p *remote_executor2.ConnPool) {
	remoteExecutorPool = p
}

// createProvider 根据协议和配置创建执行器
func createProvider(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
	switch protocol {
	case "local", "":
		return &localProvider{config: config}, nil
	case "http":
		return newHTTPProvider(config)
	case "grpc":
		return &stubProvider{protocol: protocol}, nil
	case "remoteFunc":
		funcName, _ := config["funcName"].(string)
		timeout := 30 * time.Second
		if t, ok := config["timeout"].(float64); ok && t > 0 {
			timeout = time.Duration(t) * time.Second
		}
		return &remote_executor2.RemoteFuncProvider{
			FuncName: funcName,
			Timeout:  timeout,
			Registry: nodeRegistry,
			Pool:     remoteExecutorPool,
		}, nil
	case "mcp":
		return &stubProvider{protocol: protocol}, nil
	default:
		return &stubProvider{protocol: protocol}, nil
	}
}

// ===== Local Provider — 执行本地命令 =====

type localProvider struct {
	config map[string]any
}

func (p *localProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	command, _ := p.config["command"].(string)
	if command == "" {
		return map[string]string{"status": "ok", "message": "no command configured"}, nil
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return map[string]string{"status": "ok", "message": "empty command"}, nil
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// 将前驱节点的输出作为 stdin 传入
	if data != nil && len(data.Subtasks) > 0 {
		inputBytes, _ := json.Marshal(data.Subtasks)
		cmd.Stdin = bytes.NewReader(inputBytes)
	}

	output, err := cmd.CombinedOutput()
	result := map[string]string{
		"command": command,
		"output":  strings.TrimSpace(string(output)),
	}
	if err != nil {
		result["error"] = err.Error()
		return result, fmt.Errorf("command %q failed: %w", command, err)
	}
	return result, nil
}

func (p *localProvider) Protocol() executor.Protocol { return executor.ProtocolLocal }

// ===== HTTP Provider — 发起 HTTP 请求 =====

type httpProvider struct {
	url     string
	method  string
	headers map[string]string
	timeout time.Duration
}

func newHTTPProvider(config map[string]any) (*httpProvider, error) {
	url, _ := config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("HTTP provider requires 'url' in config")
	}
	method, _ := config["method"].(string)
	if method == "" {
		method = "POST"
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if h, ok := config["headers"].(map[string]any); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	timeout := 30 * time.Second
	if t, ok := config["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	return &httpProvider{
		url:     url,
		method:  method,
		headers: headers,
		timeout: timeout,
	}, nil
}

func (p *httpProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	var body io.Reader
	if data != nil && data.Input != "" {
		body = strings.NewReader(data.Input)
	} else if data != nil && len(data.Subtasks) > 0 {
		inputBytes, _ := json.Marshal(data.Subtasks)
		body = bytes.NewReader(inputBytes)
	} else {
		body = strings.NewReader("{}")
	}

	req, err := http.NewRequestWithContext(ctx, p.method, p.url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s failed: %w", p.url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		// 非 JSON 响应，返回原始文本
		return map[string]any{"status": resp.StatusCode, "body": string(respBody)}, nil
	}
	return result, nil
}

func (p *httpProvider) Protocol() executor.Protocol { return executor.ProtocolHTTP }

// ===== Stub Provider — 占位执行器 =====

type stubProvider struct {
	protocol string
}

func (p *stubProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	return map[string]string{"status": "stub", "protocol": p.protocol}, nil
}

func (p *stubProvider) Protocol() executor.Protocol {
	return executor.Protocol(p.protocol)
}
