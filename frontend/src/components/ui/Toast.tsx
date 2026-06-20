import { createPortal } from 'react-dom';
import { CheckCircle, XCircle, Info, X } from 'lucide-react';
import { useToastStore, type ToastItem } from '../../store/toastStore';

const iconMap: Record<ToastItem['type'], typeof CheckCircle> = {
  success: CheckCircle,
  error: XCircle,
  info: Info,
};

const colorMap: Record<ToastItem['type'], string> = {
  success:
    'bg-green-50 border-green-200 text-green-800 dark:bg-green-950 dark:border-green-800 dark:text-green-200',
  error:
    'bg-red-50 border-red-200 text-red-800 dark:bg-red-950 dark:border-red-800 dark:text-red-200',
  info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-950 dark:border-blue-800 dark:text-blue-200',
};

const iconColorMap: Record<ToastItem['type'], string> = {
  success: 'text-green-600 dark:text-green-400',
  error: 'text-red-600 dark:text-red-400',
  info: 'text-blue-600 dark:text-blue-400',
};

function ToastItemView({ toast: t }: { toast: ToastItem }) {
  const Icon = iconMap[t.type];
  const dismissToast = useToastStore((s) => s.dismissToast);

  return (
    <div
      className={`flex items-center gap-2 rounded-lg border px-4 py-3 shadow-md max-w-sm ${colorMap[t.type]}`}
      style={{ animation: 'slideInRight 300ms ease-out' }}
    >
      <Icon className={`h-5 w-5 shrink-0 ${iconColorMap[t.type]}`} />
      <span className="flex-1 text-sm font-medium">{t.message}</span>
      <button
        onClick={() => dismissToast(t.id)}
        className="shrink-0 rounded p-0.5 hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

export default function ToastProvider() {
  const toasts = useToastStore((s) => s.toasts);

  if (toasts.length === 0) return null;

  return createPortal(
    <>
      <style>{`
        @keyframes slideInRight {
          from { transform: translateX(100%); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
      `}</style>
      <div className="fixed top-4 right-4 z-[9999] flex flex-col gap-2">
        {toasts.map((t) => (
          <ToastItemView key={t.id} toast={t} />
        ))}
      </div>
    </>,
    document.body,
  );
}
