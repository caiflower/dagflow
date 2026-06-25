package daemon

import (
	"context"
	"time"

	"github.com/caiflower/common-tools/cluster"
	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/global"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/crontab"
	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/dagflow/constants"
	"github.com/caiflower/dagflow/internal/dao"
	"github.com/caiflower/dagflow/internal/dao/model"
	"github.com/caiflower/dagflow/internal/service"
	"github.com/caiflower/dagflow/taskx"
)

func NewTaskxDaemon() *TaskxDaemon {
	return &TaskxDaemon{ProviderSync: crontab.NewRegularJob("ProviderSync", func() {
		InitFlowProviders()
	}, crontab.WithInterval(time.Minute), crontab.WithImmediately())}
}

// TaskxDaemon wraps taskx receiver as a global.DefaultResourceManger daemon.
type TaskxDaemon struct {
	Cluster      cluster.ICluster `autowired:""`
	ProviderSync crontab.RegularJob
}

func (d *TaskxDaemon) Name() string { return "TaskxDaemon" }

func (d *TaskxDaemon) Start() error {
	tracker := cluster.NewDefaultJobTracker(5, taskx.SingletonTaskDispatcher)
	if err := d.Cluster.AddJobTracker(tracker); err != nil {
		return err
	}

	err := taskx.StartReceiver()
	if err != nil {
		return err
	}
	if err = d.Cluster.Start(); err != nil {
		return err
	}

	d.ProviderSync.Run()

	return nil
}

func (d *TaskxDaemon) Close() {
	logger.Info("TaskxDaemon shutting down...")
	taskx.StopReceiver()
	logger.Info("TaskxDaemon stopped")
	d.Cluster.Close()
	d.ProviderSync.Stop()
}

func RegisterTaskxDaemons(dbClient *dbv1.Client) {
	clusterCfg := constants.DefaultConfig.ClusterConfig
	c, err := cluster.NewClusterWithArgs(clusterCfg, logger.NewLogger(&logger.Config{}))
	if err != nil {
		logger.Error("InitTaskx: failed to create cluster: %v", err)
		panic("failed to create cluster: " + err.Error())
	}

	taskx.InitTaskDispatcherWithDB(&taskx.Config{
		SubtaskWorker:            50,
		SubtaskQueueSize:         200,
		SubtaskRollbackWorker:    10,
		SubtaskRollbackQueueSize: 100,
		TaskWorker:               5,
		TaskQueueSize:            100,
		RemoteCallTimeout:        3 * time.Second,
	}, dbClient)

	bean.AddBean(c)

	taskxDaemon := NewTaskxDaemon()
	bean.AddBean(taskxDaemon)
	global.DefaultResourceManger.AddDaemon(taskxDaemon)
}

// ===== Flow Provider Registration =====

// InitFlowProviders 从 DB 加载所有 Flow 并注册 Provider。
// 在 bean.Ioc() 之后调用。
func InitFlowProviders() {
	flowDAO := bean.GetBeanT[*dao.FlowDAO]()
	if flowDAO == nil {
		logger.Warn("FlowDAO not available, skipping flow provider registration")
		return
	}

	flows, _, err := flowDAO.List(context.Background(), &model.FlowFilter{DisablePage: true})
	if err != nil {
		logger.Error("InitFlowProviders: list flows failed: %v", err)
		return
	}

	for i := range flows {
		service.RegisterFlowProviders(&flows[i])
	}
	logger.Info("InitFlowProviders: registered providers for %d flows", len(flows))
}
