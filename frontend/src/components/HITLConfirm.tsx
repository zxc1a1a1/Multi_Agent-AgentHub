import { useState, useEffect, useCallback } from 'react';

export interface HITLConfirmProps {
  runId: string;
  actionId: string;
  riskLevel: 'medium' | 'high';
  actionName: string;
  description: string;
  parameters?: Record<string, unknown>;
  timeoutMs?: number;
  onConfirm: (actionId: string) => void;
  onReject: (actionId: string, reason: string) => void;
  onTimeout: (actionId: string) => void;
}

/**
 * HITLConfirm renders a human-in-the-loop confirmation dialog for
 * medium and high risk actions that require user approval.
 */
export function HITLConfirm({
  runId,
  actionId,
  riskLevel,
  actionName,
  description,
  parameters,
  timeoutMs = 30000,
  onConfirm,
  onReject,
  onTimeout,
}: HITLConfirmProps) {
  const [remainingMs, setRemainingMs] = useState(timeoutMs);
  const [rejectReason, setRejectReason] = useState('');

  // Countdown timer (display-only). Backend SSE events
  // (RUN_ERROR / RUN_FINISHED) determine the actual timeout, avoiding a
  // race between frontend and backend clocks.
  useEffect(() => {
    const timer = setInterval(() => {
      setRemainingMs((prev) => Math.max(0, prev - 1000));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const handleConfirm = useCallback(() => {
    onConfirm(actionId);
  }, [actionId, onConfirm]);

  const handleReject = useCallback(() => {
    onReject(actionId, rejectReason || 'User rejected');
  }, [actionId, rejectReason, onReject]);

  const seconds = Math.ceil(remainingMs / 1000);
  const riskBadge = riskLevel === 'high' ? '🔴 高风险' : '🟡 中风险';

  return (
    <div className="hitl-confirm-overlay">
      <div className="hitl-confirm-dialog">
        <div className="hitl-confirm-header">
          <h3>操作确认</h3>
          <span className={`hitl-risk-badge hitl-risk-${riskLevel}`}>
            {riskBadge}
          </span>
        </div>

        <div className="hitl-confirm-body">
          <div className="hitl-confirm-action">
            <strong>操作：</strong>
            {actionName}
          </div>
          <div className="hitl-confirm-desc">
            <strong>描述：</strong>
            {description}
          </div>
          {parameters && Object.keys(parameters).length > 0 && (
            <div className="hitl-confirm-params">
              <strong>参数预览：</strong>
              <pre>{JSON.stringify(parameters, null, 2)}</pre>
            </div>
          )}
        </div>

        <div className="hitl-confirm-footer">
          <div className="hitl-confirm-timer">
            ⏱ 剩余 {seconds} 秒
          </div>
          <div className="hitl-confirm-actions">
            <button
              className="hitl-btn hitl-btn-reject"
              onClick={handleReject}
            >
              取消
            </button>
            <button
              className="hitl-btn hitl-btn-confirm"
              onClick={handleConfirm}
            >
              确认执行
            </button>
          </div>
        </div>

        <div className="hitl-confirm-reason">
          <input
            type="text"
            placeholder="取消原因（可选）"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
          />
        </div>
      </div>
    </div>
  );
}
