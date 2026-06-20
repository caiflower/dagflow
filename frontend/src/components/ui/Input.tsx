import { type InputHTMLAttributes, forwardRef } from 'react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className = '', ...rest }, ref) => {
    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
            {label}
          </label>
        )}
        <input
          ref={ref}
          className={[
            'w-full px-3 py-2 text-sm rounded-[var(--radius-md)] outline-none transition-all duration-150',
            'border focus:border-[var(--accent-primary)] focus:ring-2 focus:ring-[var(--accent-primary)]/20',
            'placeholder:text-[var(--text-muted)]',
            className,
          ].join(' ')}
          style={{
            background: 'var(--bg-input)',
            borderColor: error ? 'var(--danger)' : 'var(--border-default)',
            color: 'var(--text-primary)',
          }}
          {...rest}
        />
        {error && <span className="text-xs" style={{ color: 'var(--danger)' }}>{error}</span>}
        {hint && !error && <span className="text-xs" style={{ color: 'var(--text-muted)' }}>{hint}</span>}
      </div>
    );
  },
);

Input.displayName = 'Input';
export default Input;

// Textarea variant
import { type TextareaHTMLAttributes } from 'react';

interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  hint?: string;
}

export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(
  ({ label, error, hint, className = '', ...rest }, ref) => {
    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          className={[
            'w-full px-3 py-2 text-sm rounded-[var(--radius-md)] outline-none transition-all duration-150 resize-y',
            'border focus:border-[var(--accent-primary)] focus:ring-2 focus:ring-[var(--accent-primary)]/20',
            'placeholder:text-[var(--text-muted)] font-mono',
            className,
          ].join(' ')}
          style={{
            background: 'var(--bg-input)',
            borderColor: error ? 'var(--danger)' : 'var(--border-default)',
            color: 'var(--text-primary)',
          }}
          {...rest}
        />
        {error && <span className="text-xs" style={{ color: 'var(--danger)' }}>{error}</span>}
        {hint && !error && <span className="text-xs" style={{ color: 'var(--text-muted)' }}>{hint}</span>}
      </div>
    );
  },
);

TextArea.displayName = 'TextArea';
