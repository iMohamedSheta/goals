import * as React from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

/**
 * Sheet — side panel sliding in from the inline-end side
 * (right in LTR, left in RTL). Replaces centered modals.
 */
export function Sheet({ open, onClose, children, className, narrow }) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === 'Escape' && onClose?.();
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-[2px] animate-fade-in"
        onMouseDown={(e) => e.target === e.currentTarget && onClose?.()}
      />
      <div
        className={cn(
          'sheet-panel surf absolute inset-y-0 end-0 flex w-full flex-col border-s bg-popover text-popover-foreground shadow-2xl',
          narrow ? 'max-w-sm' : 'max-w-md',
          className
        )}
      >
        {children}
      </div>
    </div>
  );
}

export function SheetHeader({ className, title, description, onClose }) {
  return (
    <div className={cn('flex items-start gap-3 border-b px-6 py-5', className)}>
      <div className="min-w-0 flex-1">
        <h2 className="text-lg font-bold leading-tight tracking-tight">{title}</h2>
        {description && <p className="mt-1 text-[13px] leading-relaxed text-muted-foreground">{description}</p>}
      </div>
      <button
        onClick={onClose}
        className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        aria-label="Close"
      >
        <X size={17} />
      </button>
    </div>
  );
}

export function SheetBody({ className, ...props }) {
  return <div className={cn('flex-1 space-y-4 overflow-y-auto px-6 py-5', className)} {...props} />;
}

export function SheetFooter({ className, ...props }) {
  return (
    <div className={cn('flex items-center justify-end gap-2 border-t px-6 py-4', className)} {...props} />
  );
}
