import type { ReactNode } from 'react';

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'subtle';

interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
  dot?: boolean;
}

const variantStyles: Record<BadgeVariant, { bg: string; color: string; dot: string }> = {
  default:  { bg: 'var(--bg-tertiary)', color: 'var(--text-secondary)', dot: 'var(--text-muted)' },
  success:  { bg: 'var(--success-subtle)', color: 'var(--success)', dot: 'var(--success)' },
  warning:  { bg: 'var(--warning-subtle)', color: 'var(--warning)', dot: 'var(--warning)' },
  danger:   { bg: 'var(--danger-subtle)', color: 'var(--danger)', dot: 'var(--danger)' },
  info:     { bg: 'var(--info-subtle)', color: 'var(--info)', dot: 'var(--info)' },
  subtle:   { bg: 'var(--accent-subtle)', color: 'var(--accent-primary)', dot: 'var(--accent-primary)' },
};

export default function Badge({ variant = 'default', children, className = '', dot = false }: BadgeProps) {
  const s = variantStyles[variant];
  return (
    <span
      className={[
        'inline-flex items-center gap-1.5 px-2.5 py-0.5 text-xs font-medium rounded-full',
        className,
      ].join(' ')}
      style={{ background: s.bg, color: s.color }}
    >
      {dot && (
        <span
          className="w-1.5 h-1.5 rounded-full flex-shrink-0"
          style={{ background: s.dot }}
        />
      )}
      {children}
    </span>
  );
}
