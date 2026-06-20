import { Link, Outlet, useLocation } from 'react-router-dom';
import { LayoutList, Play, Plug, Sun, Moon, Workflow } from 'lucide-react';
import { useTheme } from '../ThemeProvider';

const NAV_ITEMS = [
  { path: '/flows', label: 'Flows', icon: LayoutList },
  { path: '/executions', label: '执行', icon: Play },
  { path: '/protocols', label: '协议', icon: Plug },
];

export default function Layout() {
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();

  const isActive = (path: string) => location.pathname.startsWith(path);

  return (
    <div className="flex flex-col h-screen" style={{ background: 'var(--bg-primary)' }}>
      {/* Top Bar — glassmorphism */}
      <header
        className="flex items-center justify-between px-6 h-14 flex-shrink-0"
        style={{
          background: 'var(--glass-bg)',
          backdropFilter: 'saturate(180%) blur(20px)',
          WebkitBackdropFilter: 'saturate(180%) blur(20px)',
          borderBottom: '1px solid var(--border-subtle)',
        }}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5">
          <Workflow size={22} style={{ color: 'var(--accent-primary)' }} strokeWidth={2} />
          <span className="text-base font-semibold tracking-tight" style={{ color: 'var(--text-primary)' }}>
            DAGFlow
          </span>
        </div>

        {/* Navigation Tabs */}
        <nav className="flex items-center gap-1">
          {NAV_ITEMS.map(({ path, label, icon: Icon }) => {
            const active = isActive(path);
            return (
              <Link
                key={path}
                to={path}
                className="flex items-center gap-1.5 px-3.5 py-1.5 text-sm font-medium rounded-full transition-all duration-150"
                style={{
                  background: active ? 'var(--accent-subtle)' : 'transparent',
                  color: active ? 'var(--accent-primary)' : 'var(--text-secondary)',
                }}
                onMouseEnter={(e) => {
                  if (!active) e.currentTarget.style.background = 'var(--bg-secondary)';
                }}
                onMouseLeave={(e) => {
                  if (!active) e.currentTarget.style.background = 'transparent';
                }}
              >
                <Icon size={15} strokeWidth={active ? 2.5 : 2} />
                <span>{label}</span>
              </Link>
            );
          })}
        </nav>

        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          className="p-2 rounded-full transition-all duration-200"
          style={{ color: 'var(--text-secondary)' }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'var(--bg-secondary)';
            e.currentTarget.style.color = 'var(--text-primary)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = '';
            e.currentTarget.style.color = 'var(--text-secondary)';
          }}
          title={theme === 'light' ? '切换深色模式' : '切换浅色模式'}
        >
          {theme === 'light' ? <Moon size={18} /> : <Sun size={18} />}
        </button>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-auto animate-fade-in">
        <Outlet />
      </main>
    </div>
  );
}
