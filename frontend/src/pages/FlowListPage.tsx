import { useEffect, useState } from 'react';
import { toast } from '../store/toastStore';
import { useNavigate } from 'react-router-dom';
import { Plus, Search, Trash2, GitBranch } from 'lucide-react';
import { useFlowStore } from '../store';
import { ApiError } from '../api/client';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Input from '../components/ui/Input';
import Badge from '../components/ui/Badge';
import { statusBadgeVariant } from '../utils/stateColor';

export default function FlowListPage() {
  const { flows, total, loading, loadFlows, deleteFlow, createFlow } = useFlowStore();
  const navigate = useNavigate();
  const [searchName, setSearchName] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');

  useEffect(() => { loadFlows(1, searchName); }, [loadFlows, searchName]);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    const defaultNodes = [
      { id: 'start', name: '开始', type: 'start' as const, protocol: '', config: {}, position: { x: 250, y: 50 } },
      { id: 'end', name: '结束', type: 'end' as const, protocol: '', config: {}, position: { x: 250, y: 400 } },
    ];
    try {
    await createFlow({ name: newName, description: newDesc, nodes: defaultNodes, edges: [] });
    setShowCreate(false);
    setNewName('');
    setNewDesc('');
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Failed to create flow';
      toast.error(msg);
    }
  };

  return (
    <div className="max-w-5xl mx-auto px-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight" style={{ color: 'var(--text-primary)' }}>
            Flows
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--text-muted)' }}>
            管理和编排你的工作流
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)} size="md">
          <Plus size={16} /> 新建 Flow
        </Button>
      </div>

      {/* Search */}
      <div className="relative mb-6">
        <Search
          size={16}
          className="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
          style={{ color: 'var(--text-muted)' }}
        />
        <input
          type="text"
          placeholder="搜索 Flow..."
          value={searchName}
          onChange={(e) => setSearchName(e.target.value)}
          className="w-full pl-10 pr-4 py-2.5 text-sm rounded-[var(--radius-md)] border outline-none transition-all"
          style={{
            background: 'var(--bg-input)',
            borderColor: 'var(--border-default)',
            color: 'var(--text-primary)',
          }}
          onFocus={(e) => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; }}
          onBlur={(e) => { e.currentTarget.style.borderColor = 'var(--border-default)'; }}
        />
      </div>

      {/* Flow Cards */}
      {loading ? (
        <div className="text-center py-16" style={{ color: 'var(--text-muted)' }}>
          <div className="animate-spin w-6 h-6 border-2 border-current border-t-transparent rounded-full mx-auto mb-3" />
          加载中...
        </div>
      ) : (
        <div className="grid gap-3">
          {flows.map((flow) => (
            <div
              key={flow.id}
              className="group rounded-[var(--radius-lg)] p-4 cursor-pointer transition-all duration-200"
              style={{
                background: 'var(--bg-secondary)',
                border: '1px solid var(--border-subtle)',
              }}
              onClick={() => navigate(`/flows/${flow.id}`)}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = 'var(--accent-primary)';
                e.currentTarget.style.boxShadow = 'var(--shadow-md)';
                e.currentTarget.style.transform = 'translateY(-1px)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border-subtle)';
                e.currentTarget.style.boxShadow = '';
                e.currentTarget.style.transform = '';
              }}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <div
                    className="p-2 rounded-[var(--radius-md)] flex-shrink-0 mt-0.5"
                    style={{ background: 'var(--accent-subtle)' }}
                  >
                    <GitBranch size={18} style={{ color: 'var(--accent-primary)' }} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-base" style={{ color: 'var(--text-primary)' }}>
                      {flow.name}
                    </h3>
                    <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>
                      {flow.description || '无描述'}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Badge variant={statusBadgeVariant(flow.status)} dot>
                    {flow.status === 1 ? '启用' : '禁用'}
                  </Badge>
                  <span className="text-xs" style={{ color: 'var(--text-muted)' }}>v{flow.version}</span>
                  <button
                    onClick={(e) => { e.stopPropagation(); deleteFlow(flow.id); }}
                    className="p-1.5 rounded-[var(--radius-sm)] opacity-0 group-hover:opacity-100 transition-all"
                    style={{ color: 'var(--text-muted)' }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = 'var(--danger-subtle)';
                      e.currentTarget.style.color = 'var(--danger)';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = '';
                      e.currentTarget.style.color = 'var(--text-muted)';
                    }}
                    title="删除"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            </div>
          ))}
          {flows.length === 0 && (
            <div className="text-center py-16 rounded-[var(--radius-lg)]" style={{ background: 'var(--bg-secondary)', border: '1px dashed var(--border-default)' }}>
              <GitBranch size={40} className="mx-auto mb-3" style={{ color: 'var(--text-muted)' }} />
              <p style={{ color: 'var(--text-muted)' }}>暂无 Flow，点击上方按钮创建</p>
            </div>
          )}
        </div>
      )}

      <div className="mt-4 text-xs" style={{ color: 'var(--text-muted)' }}>
        共 {total} 条记录
      </div>

      {/* Create Modal */}
      <Modal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        title="新建 Flow"
        footer={
          <>
            <Button variant="secondary" onClick={() => setShowCreate(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={!newName.trim()}>创建</Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Flow 名称"
            placeholder="输入 Flow 名称"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            autoFocus
          />
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>描述（可选）</label>
            <textarea
              placeholder="描述你的工作流..."
              value={newDesc}
              onChange={(e) => setNewDesc(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 text-sm rounded-[var(--radius-md)] border outline-none transition-all resize-none"
              style={{
                background: 'var(--bg-input)',
                borderColor: 'var(--border-default)',
                color: 'var(--text-primary)',
              }}
              onFocus={(e) => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; }}
              onBlur={(e) => { e.currentTarget.style.borderColor = 'var(--border-default)'; }}
            />
          </div>
        </div>
      </Modal>
    </div>
  );
}
