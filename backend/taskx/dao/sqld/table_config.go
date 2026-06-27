package sqld

// TableConfig configures the physical table names for taskx DAO models.
type TableConfig struct {
	Task            string `yaml:"task" json:"task" default:"task"`
	Subtask         string `yaml:"subtask" json:"subtask" default:"subtask"`
	TaskBak         string `yaml:"taskBak" json:"taskBak" default:"task_archive"`
	SubtaskBak      string `yaml:"subtaskBak" json:"subtaskBak" default:"subtask_archive"`
	TaskEdge        string `yaml:"taskEdge" json:"taskEdge" default:"task_edge"`
	TaskEdgeArchive string `yaml:"taskEdgeArchive" json:"taskEdgeArchive" default:"task_edge_archive"`
}

func DefaultTableConfig() *TableConfig {
	return &TableConfig{
		Task:            "task",
		Subtask:         "subtask",
		TaskBak:         "task_archive",
		SubtaskBak:      "subtask_archive",
		TaskEdge:        "task_edge",
		TaskEdgeArchive: "task_edge_archive",
	}
}

func (c *TableConfig) Normalize() *TableConfig {
	if c == nil {
		return DefaultTableConfig()
	}
	cp := *c
	if cp.Task == "" {
		cp.Task = "task"
	}
	if cp.Subtask == "" {
		cp.Subtask = "subtask"
	}
	if cp.TaskBak == "" {
		cp.TaskBak = "task_archive"
	}
	if cp.SubtaskBak == "" {
		cp.SubtaskBak = "subtask_archive"
	}
	if cp.TaskEdge == "" {
		cp.TaskEdge = "task_edge"
	}
	if cp.TaskEdgeArchive == "" {
		cp.TaskEdgeArchive = "task_edge_archive"
	}
	return &cp
}
