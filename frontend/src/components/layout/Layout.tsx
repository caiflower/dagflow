import { Link, Outlet, useLocation } from 'react-router-dom';

export default function Layout() {
  const location = useLocation();
  const isActive = (path: string) =>
    location.pathname.startsWith(path) ? 'bg-blue-600 text-white' : 'text-gray-300 hover:bg-gray-700 hover:text-white';

  return (
    <div className="flex h-screen bg-gray-900 text-gray-100">
      {/* Sidebar */}
      <nav className="w-56 bg-gray-800 flex flex-col border-r border-gray-700">
        <div className="p-4 border-b border-gray-700">
          <h1 className="text-xl font-bold text-blue-400">DAGFlow</h1>
          <p className="text-xs text-gray-500 mt-1">可视化工作流平台</p>
        </div>
        <div className="flex-1 py-4">
          <Link to="/flows" className={`flex items-center px-4 py-2.5 mx-2 rounded-md text-sm ${isActive('/flows')}`}>
            <span className="mr-2">📋</span> Flow 列表
          </Link>
          <Link to="/executions" className={`flex items-center px-4 py-2.5 mx-2 rounded-md text-sm ${isActive('/executions')}`}>
            <span className="mr-2">▶️</span> 执行记录
          </Link>
          <Link to="/protocols" className={`flex items-center px-4 py-2.5 mx-2 rounded-md text-sm ${isActive('/protocols')}`}>
            <span className="mr-2">🔌</span> 协议管理
          </Link>
        </div>
      </nav>
      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
