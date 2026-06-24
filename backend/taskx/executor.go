package taskx

import (
	"context"
	"errors"

	"fmt"
	"github.com/caiflower/common-tools/pkg/tools"
	"strings"

	"sync"

	"github.com/caiflower/dagflow/taskx/executor"
)

var (
	ErrNonRetryable = errors.New("non-retryable error")
)

// _providerRegistry 全局执行器注册表（集群框架要求，receiver 从数据库读取时查找）
// taskName -> subTaskName -> ExecutorProvider
var (
	_providerRegistry = struct {
		sync.RWMutex
		providers map[string]map[string]executor.ExecutorProvider
	}{providers: make(map[string]map[string]executor.ExecutorProvider)}
)

// registerProvider 注册子任务执行器（全局）
func registerProvider(taskName, subTaskName string, p executor.ExecutorProvider) {
	_providerRegistry.Lock()
	defer _providerRegistry.Unlock()
	if _providerRegistry.providers[taskName] == nil {
		_providerRegistry.providers[taskName] = make(map[string]executor.ExecutorProvider)
	}
	_providerRegistry.providers[taskName][subTaskName] = p
}

// registerProviders 批量注册执行器（全局）
func registerProviders(taskName string, providers map[string]executor.ExecutorProvider) {
	_providerRegistry.Lock()
	defer _providerRegistry.Unlock()
	if _providerRegistry.providers[taskName] == nil {
		_providerRegistry.providers[taskName] = make(map[string]executor.ExecutorProvider)
	}
	for name, p := range providers {
		_providerRegistry.providers[taskName][name] = p
	}
}

// getProvider 查找子任务执行器（全局）
func getProvider(taskName, subTaskName string) executor.ExecutorProvider {
	_providerRegistry.RLock()
	defer _providerRegistry.RUnlock()
	if providers, ok := _providerRegistry.providers[taskName]; ok {
		return providers[subTaskName]
	}
	return nil
}

// registerRollbackProvider 注册回滚执行器（全局）
var (
	_rollbackRegistry = struct {
		sync.RWMutex
		providers map[string]executor.ExecutorProvider // key: taskName/subTaskName
	}{providers: make(map[string]executor.ExecutorProvider)}
)

func registerRollbackProvider(taskName, subTaskName string, p executor.ExecutorProvider) {
	_rollbackRegistry.Lock()
	defer _rollbackRegistry.Unlock()
	_rollbackRegistry.providers[taskName+"/"+subTaskName] = p
}

func getRollbackProvider(taskName, subTaskName string) executor.ExecutorProvider {
	_rollbackRegistry.RLock()
	defer _rollbackRegistry.RUnlock()
	return _rollbackRegistry.providers[taskName+"/"+subTaskName]
}

// 全局 TaskExecutor 注册表（集群框架：receiver 从数据库读取时需要）
var (
	_taskExecutorRegistry = struct {
		sync.RWMutex
		executors map[string]TaskExecutor
	}{executors: make(map[string]TaskExecutor)}
)

// _customRollbackRegistry 全局自定义回滚函数注册表（集群框架：dispatcher 从 DB 重建 Task 时恢复）
// taskName -> custom rollback function
var (
	_customRollbackRegistry = struct {
		sync.RWMutex
		funcs map[string]func(completed []string, failed string) []string
	}{funcs: make(map[string]func(completed []string, failed string) []string)}
)

func registerCustomRollback(taskName string, fn func(completed []string, failed string) []string) {
	_customRollbackRegistry.Lock()
	defer _customRollbackRegistry.Unlock()
	_customRollbackRegistry.funcs[taskName] = fn
}

func getCustomRollback(taskName string) func(completed []string, failed string) []string {
	_customRollbackRegistry.RLock()
	defer _customRollbackRegistry.RUnlock()
	return _customRollbackRegistry.funcs[taskName]
}

func registerTaskExecutor(e TaskExecutor) {
	_taskExecutorRegistry.Lock()
	defer _taskExecutorRegistry.Unlock()
	_taskExecutorRegistry.executors[e.Name()] = e
}

func getTaskExecutor(taskName string) TaskExecutor {
	_taskExecutorRegistry.RLock()
	defer _taskExecutorRegistry.RUnlock()
	return _taskExecutorRegistry.executors[taskName]
}

// ClearProviders 清理指定 taskName 的全局执行器注册表
func ClearProviders(taskName string) {
	_providerRegistry.Lock()
	defer _providerRegistry.Unlock()
	delete(_providerRegistry.providers, taskName)

	_rollbackRegistry.Lock()
	defer _rollbackRegistry.Unlock()
	for key := range _rollbackRegistry.providers {
		if strings.HasPrefix(key, taskName+"/") {
			delete(_rollbackRegistry.providers, key)
		}
	}

	_customRollbackRegistry.Lock()
	defer _customRollbackRegistry.Unlock()
	delete(_customRollbackRegistry.funcs, taskName)
}

// ClearAllProviders 清理所有全局执行器注册表
func ClearAllProviders() {
	_providerRegistry.Lock()
	defer _providerRegistry.Unlock()
	_providerRegistry.providers = make(map[string]map[string]executor.ExecutorProvider)

	_rollbackRegistry.Lock()
	defer _rollbackRegistry.Unlock()
	_rollbackRegistry.providers = make(map[string]executor.ExecutorProvider)

	_customRollbackRegistry.Lock()
	defer _customRollbackRegistry.Unlock()
	_customRollbackRegistry.funcs = make(map[string]func(completed []string, failed string) []string)
}

// ClearTaskExecutors 清理指定 taskName 的全局 TaskExecutor 注册表
func ClearTaskExecutors(taskName string) {
	_taskExecutorRegistry.Lock()
	defer _taskExecutorRegistry.Unlock()
	delete(_taskExecutorRegistry.executors, taskName)
}

// ClearAllTaskExecutors 清理所有全局 TaskExecutor 注册表
func ClearAllTaskExecutors() {
	_taskExecutorRegistry.Lock()
	defer _taskExecutorRegistry.Unlock()
	_taskExecutorRegistry.executors = make(map[string]TaskExecutor)
}

// _processorRegistry 全局处理器注册表（集群框架：receiver 从数据库恢复时查找 preProcessor/postProcessor）
var (
	_preProcessorRegistry = struct {
		sync.RWMutex
		processors map[string]map[string]Processor // taskName -> subTaskName -> Processor
	}{processors: make(map[string]map[string]Processor)}

	_postProcessorRegistry = struct {
		sync.RWMutex
		processors map[string]map[string]Processor // taskName -> subTaskName -> Processor
	}{processors: make(map[string]map[string]Processor)}
)

func registerPreProcessor(taskName, subTaskName string, p Processor) {
	_preProcessorRegistry.Lock()
	defer _preProcessorRegistry.Unlock()
	if _preProcessorRegistry.processors[taskName] == nil {
		_preProcessorRegistry.processors[taskName] = make(map[string]Processor)
	}
	_preProcessorRegistry.processors[taskName][subTaskName] = p
}

func getPreProcessor(taskName, subTaskName string) Processor {
	_preProcessorRegistry.RLock()
	defer _preProcessorRegistry.RUnlock()
	if processors, ok := _preProcessorRegistry.processors[taskName]; ok {
		return processors[subTaskName]
	}
	return nil
}

func registerPostProcessor(taskName, subTaskName string, p Processor) {
	_postProcessorRegistry.Lock()
	defer _postProcessorRegistry.Unlock()
	if _postProcessorRegistry.processors[taskName] == nil {
		_postProcessorRegistry.processors[taskName] = make(map[string]Processor)
	}
	_postProcessorRegistry.processors[taskName][subTaskName] = p
}

func getPostProcessor(taskName, subTaskName string) Processor {
	_postProcessorRegistry.RLock()
	defer _postProcessorRegistry.RUnlock()
	if processors, ok := _postProcessorRegistry.processors[taskName]; ok {
		return processors[subTaskName]
	}
	return nil
}

// executeBranchCondition reads the branch subtask's Settings, resolves the
// condition provider from the global provider registry, executes it, and returns the selected key.
// Returns the selected end node key on success, or an error.

// resultToString converts a provider result to string.
// For remote providers (SDK), the result is JSON-encoded bytes (e.g. `"echo"`).
// For local providers, the result is the native Go string.
func resultToString(result any) string {
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		// Remote providers return JSON-encoded output; try to decode as string first.
		var s string
		if err := tools.Unmarshal(v, &s); err == nil {
			return s
		}
		return string(v)
	default:
		return ""
	}
}

func executeBranchCondition(taskName, subtaskName, nodeKey, settingsJSON string, conditionInput any) (string, error) {
	if settingsJSON == "" {
		return "", errors.New("branch: empty settings JSON")
	}

	var settings SubtaskSettings
	if err := tools.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return "", fmt.Errorf("branch: failed to parse settings: %w", err)
	}
	if settings.BranchConfig == nil {
		return "", errors.New("branch: no BranchConfig in settings")
	}

	// Resolve condition provider from global provider registry (registered under branch subtask name)
	provider := getProvider(taskName, subtaskName)
	if provider == nil {
		return "", fmt.Errorf("branch: condition provider not found for %s/%s", taskName, subtaskName)
	}

	// Prepare TaskData with the condition input (parent node output)
	taskData := &executor.TaskData{
		SubTaskId: nodeKey,
	}
	if conditionInput != nil {
		if s, ok := conditionInput.(string); ok {
			taskData.Input = s
		}
	}

	result, err := provider.Execute(context.Background(), taskData)
	if err != nil {
		return "", fmt.Errorf("branch: condition execution failed: %w", err)
	}

	selected := resultToString(result)
	if selected == "" {
		return "", fmt.Errorf("branch: condition returned non-string result: %v", result)
	}

	return selected, nil
}
