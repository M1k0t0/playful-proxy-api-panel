import { type ChangeEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { IconDownload, IconRefreshCw, IconUpload } from '@/components/ui/icons';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { usageApi } from '@/services/api';
import { useAuthStore, useNotificationStore } from '@/stores';
import type {
  UsageApiSnapshot,
  UsageExportPayload,
  UsageModelSnapshot,
  UsageRequestDetail,
  UsageStatisticsResponse,
  UsageStatisticsSnapshot,
} from '@/types';
import styles from './UsagePage.module.scss';

interface ApiUsageRow {
  name: string;
  totalRequests: number;
  totalTokens: number;
  cacheHitRate: number;
  averageLatencyMs: number;
  averageFirstByteLatencyMs: number;
  tps: number;
  modelCount: number;
  failedRequests: number;
}

interface ModelUsageRow extends ApiUsageRow {
  apiName: string;
}

interface DetailUsageRow {
  key: string;
  apiName: string;
  modelName: string;
  detail: UsageRequestDetail;
}

interface TimelineRow {
  label: string;
  value: number;
}

const emptyUsage: UsageStatisticsSnapshot = {
  total_requests: 0,
  success_count: 0,
  failure_count: 0,
  total_tokens: 0,
  total_input_tokens: 0,
  total_cached_tokens: 0,
  cache_hit_rate: 0,
  average_latency_ms: 0,
  average_first_byte_latency_ms: 0,
  tps: 0,
  apis: {},
  requests_by_day: {},
  requests_by_hour: {},
  tokens_by_day: {},
  tokens_by_hour: {},
};

const safeNumber = (value: number | null | undefined): number =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const countFailedDetails = (models: Record<string, UsageModelSnapshot> | undefined): number =>
  Object.values(models ?? {}).reduce(
    (total, model) => total + (model.details ?? []).filter((detail) => detail.failed).length,
    0
  );

const detailTimestamp = (detail: UsageRequestDetail): number => {
  const timestamp = Date.parse(detail.timestamp);
  return Number.isFinite(timestamp) ? timestamp : 0;
};

const createApiRow = (name: string, api: UsageApiSnapshot): ApiUsageRow => ({
  name,
  totalRequests: safeNumber(api.total_requests),
  totalTokens: safeNumber(api.total_tokens),
  cacheHitRate: safeNumber(api.cache_hit_rate),
  averageLatencyMs: safeNumber(api.average_latency_ms),
  averageFirstByteLatencyMs: safeNumber(api.average_first_byte_latency_ms),
  tps: safeNumber(api.tps),
  modelCount: Object.keys(api.models ?? {}).length,
  failedRequests: countFailedDetails(api.models),
});

const timelineRows = (input: Record<string, number> | undefined): TimelineRow[] =>
  Object.entries(input ?? {})
    .map(([label, value]) => ({ label, value: safeNumber(value) }))
    .sort((a, b) => a.label.localeCompare(b.label));

export function UsagePage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const { showNotification } = useNotificationStore();

  const [response, setResponse] = useState<UsageStatisticsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState('');

  const importInputRef = useRef<HTMLInputElement | null>(null);

  const usage = response?.usage ?? emptyUsage;
  const failedRequests = safeNumber(response?.failed_requests ?? usage.failure_count);
  const hasUsage =
    safeNumber(usage.total_requests) > 0 ||
    Object.keys(usage.apis ?? {}).length > 0 ||
    timelineRows(usage.requests_by_day).length > 0;

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(i18n.language || undefined),
    [i18n.language]
  );

  const formatNumber = useCallback(
    (value: number | null | undefined) => numberFormatter.format(safeNumber(value)),
    [numberFormatter]
  );

  const formatPercent = useCallback((value: number | null | undefined) => {
    const numeric = safeNumber(value);
    return `${numeric.toFixed(numeric >= 10 ? 1 : 2)}%`;
  }, []);

  const formatLatency = useCallback(
    (value: number | null | undefined) => `${formatNumber(Math.round(safeNumber(value)))} ms`,
    [formatNumber]
  );

  const formatTps = useCallback((value: number | null | undefined) => safeNumber(value).toFixed(2), []);

  const formatTimestamp = useCallback(
    (timestamp: string) => {
      const date = new Date(timestamp);
      if (!Number.isFinite(date.getTime())) return '-';
      return date.toLocaleString(i18n.language || undefined);
    },
    [i18n.language]
  );

  const apiRows = useMemo(
    () =>
      Object.entries(usage.apis ?? {})
        .map(([name, api]) => createApiRow(name, api))
        .sort((a, b) => b.totalRequests - a.totalRequests),
    [usage.apis]
  );

  const modelRows = useMemo(
    () =>
      Object.entries(usage.apis ?? {})
        .flatMap(([apiName, api]) =>
          Object.entries(api.models ?? {}).map(([modelName, model]) => ({
            ...createApiRow(modelName, {
              total_requests: model.total_requests,
              total_tokens: model.total_tokens,
              total_input_tokens: model.total_input_tokens,
              total_cached_tokens: model.total_cached_tokens,
              cache_hit_rate: model.cache_hit_rate,
              average_latency_ms: model.average_latency_ms,
              average_first_byte_latency_ms: model.average_first_byte_latency_ms,
              tps: model.tps,
              models: {},
            }),
            apiName,
            failedRequests: (model.details ?? []).filter((detail) => detail.failed).length,
          }))
        )
        .sort((a, b) => b.totalRequests - a.totalRequests)
        .slice(0, 40),
    [usage.apis]
  );

  const recentRows = useMemo(
    () =>
      Object.entries(usage.apis ?? {})
        .flatMap(([apiName, api]) =>
          Object.entries(api.models ?? {}).flatMap(([modelName, model]) =>
            (model.details ?? []).map((detail, detailIndex) => ({
              key: `${apiName}:${modelName}:${detail.timestamp}:${detailIndex}`,
              apiName,
              modelName,
              detail,
            }))
          )
        )
        .sort((a, b) => detailTimestamp(b.detail) - detailTimestamp(a.detail))
        .slice(0, 50),
    [usage.apis]
  );

  const requestTimeline = useMemo(
    () => timelineRows(usage.requests_by_day).slice(-14),
    [usage.requests_by_day]
  );

  const tokenTimeline = useMemo(() => timelineRows(usage.tokens_by_day).slice(-14), [usage.tokens_by_day]);

  const loadUsage = useCallback(async () => {
    if (connectionStatus !== 'connected') {
      setLoading(false);
      setResponse(null);
      setError(t('usage_statistics.connection_required'));
      return;
    }

    setLoading(true);
    setError('');
    try {
      const data = await usageApi.getStatistics();
      setResponse(data);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.refresh_failed');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [connectionStatus, t]);

  useHeaderRefresh(loadUsage);

  useEffect(() => {
    loadUsage();
  }, [loadUsage]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const payload = await usageApi.exportStatistics();
      const exportedAt = payload.exported_at || new Date().toISOString();
      const fileSafeDate = exportedAt.replace(/[:.]/g, '-');
      const blob = new Blob([JSON.stringify(payload, null, 2)], {
        type: 'application/json;charset=utf-8',
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `ppap-usage-${fileSafeDate}.json`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      showNotification(t('usage_statistics.export_success'), 'success');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('usage_statistics.export_failed');
      showNotification(message, 'error');
    } finally {
      setExporting(false);
    }
  };

  const normalizeImportPayload = (parsed: unknown): UsageExportPayload => {
    if (!isRecord(parsed)) {
      throw new Error(t('usage_statistics.import_invalid'));
    }

    const usageValue = isRecord(parsed.usage) ? parsed.usage : parsed;
    if (!isRecord(usageValue)) {
      throw new Error(t('usage_statistics.import_invalid'));
    }

    return {
      version: typeof parsed.version === 'number' ? parsed.version : 1,
      exported_at: typeof parsed.exported_at === 'string' ? parsed.exported_at : new Date().toISOString(),
      usage: usageValue as unknown as UsageStatisticsSnapshot,
    };
  };

  const handleImportFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = '';
    if (!file) return;

    setImporting(true);
    try {
      const parsed = JSON.parse(await file.text());
      const payload = normalizeImportPayload(parsed);
      const result = await usageApi.importStatistics(payload);
      showNotification(
        t('usage_statistics.import_success', {
          added: result.added,
          skipped: result.skipped,
        }),
        'success'
      );
      await loadUsage();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('usage_statistics.import_failed');
      showNotification(message, 'error');
    } finally {
      setImporting(false);
    }
  };

  const renderMetric = (label: string, value: string, sublabel?: string) => (
    <div className={styles.metricTile}>
      <span className={styles.metricLabel}>{label}</span>
      <span className={styles.metricValue}>{value}</span>
      {sublabel && <span className={styles.metricSub}>{sublabel}</span>}
    </div>
  );

  const renderTimeline = (rows: TimelineRow[], valueLabel: string) => {
    const max = Math.max(...rows.map((row) => row.value), 0);
    if (!rows.length) {
      return <div className={styles.emptyInline}>{t('usage_statistics.no_timeline')}</div>;
    }

    return (
      <div className={styles.timelineList}>
        {rows.map((row) => {
          const width = max > 0 ? Math.max((row.value / max) * 100, 2) : 0;
          return (
            <div className={styles.timelineRow} key={row.label}>
              <span className={styles.timelineLabel}>{row.label}</span>
              <span className={styles.timelineTrack} aria-hidden="true">
                <span className={styles.timelineBar} style={{ width: `${width}%` }} />
              </span>
              <span className={styles.timelineValue}>
                {formatNumber(row.value)} {valueLabel}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className={styles.container}>
      <div className={styles.pageHeader}>
        <div>
          <h1 className={styles.pageTitle}>{t('usage_statistics.title')}</h1>
          <p className={styles.description}>{t('usage_statistics.description')}</p>
        </div>
        <div className={styles.headerActions}>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={loadUsage}
            loading={loading}
            disabled={connectionStatus !== 'connected'}
          >
            <IconRefreshCw size={16} />
            {t('common.refresh')}
          </Button>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => importInputRef.current?.click()}
            loading={importing}
            disabled={connectionStatus !== 'connected'}
          >
            <IconUpload size={16} />
            {t('usage_statistics.import')}
          </Button>
          <Button
            type="button"
            variant="primary"
            size="sm"
            onClick={handleExport}
            loading={exporting}
            disabled={connectionStatus !== 'connected'}
          >
            <IconDownload size={16} />
            {t('usage_statistics.export')}
          </Button>
          <input
            ref={importInputRef}
            className={styles.hiddenInput}
            type="file"
            accept="application/json,.json"
            onChange={handleImportFile}
          />
        </div>
      </div>

      {error && <div className={styles.errorBox}>{error}</div>}

      <section className={styles.metricGrid} aria-label={t('usage_statistics.summary')}>
        {renderMetric(t('usage_statistics.total_requests'), formatNumber(usage.total_requests))}
        {renderMetric(t('usage_statistics.failed_requests'), formatNumber(failedRequests))}
        {renderMetric(t('usage_statistics.cache_hit_rate'), formatPercent(usage.cache_hit_rate))}
        {renderMetric(t('usage_statistics.first_byte_latency'), formatLatency(usage.average_first_byte_latency_ms))}
        {renderMetric(t('usage_statistics.average_latency'), formatLatency(usage.average_latency_ms))}
        {renderMetric(t('usage_statistics.tps'), formatTps(usage.tps))}
        {renderMetric(t('usage_statistics.total_tokens'), formatNumber(usage.total_tokens))}
        {renderMetric(t('usage_statistics.cached_tokens'), formatNumber(usage.total_cached_tokens))}
      </section>

      {!loading && !hasUsage && (
        <div className={styles.emptyState}>{t('usage_statistics.empty')}</div>
      )}

      <div className={styles.sectionGrid}>
        <Card title={t('usage_statistics.requests_by_day')}>
          {renderTimeline(requestTimeline, t('usage_statistics.requests_unit'))}
        </Card>
        <Card title={t('usage_statistics.tokens_by_day')}>
          {renderTimeline(tokenTimeline, t('usage_statistics.tokens_unit'))}
        </Card>
      </div>

      <Card title={t('usage_statistics.api_breakdown')}>
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{t('usage_statistics.api')}</th>
                <th>{t('usage_statistics.requests')}</th>
                <th>{t('usage_statistics.failed')}</th>
                <th>{t('usage_statistics.tokens')}</th>
                <th>{t('usage_statistics.cache')}</th>
                <th>{t('usage_statistics.first_byte')}</th>
                <th>{t('usage_statistics.latency')}</th>
                <th>{t('usage_statistics.tps')}</th>
                <th>{t('usage_statistics.models')}</th>
              </tr>
            </thead>
            <tbody>
              {apiRows.length ? (
                apiRows.map((row) => (
                  <tr key={row.name}>
                    <td className={styles.monoCell}>{row.name}</td>
                    <td>{formatNumber(row.totalRequests)}</td>
                    <td>{formatNumber(row.failedRequests)}</td>
                    <td>{formatNumber(row.totalTokens)}</td>
                    <td>{formatPercent(row.cacheHitRate)}</td>
                    <td>{formatLatency(row.averageFirstByteLatencyMs)}</td>
                    <td>{formatLatency(row.averageLatencyMs)}</td>
                    <td>{formatTps(row.tps)}</td>
                    <td>{formatNumber(row.modelCount)}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={9} className={styles.emptyCell}>
                    {t('usage_statistics.no_api_rows')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <Card title={t('usage_statistics.model_breakdown')}>
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{t('usage_statistics.model')}</th>
                <th>{t('usage_statistics.api')}</th>
                <th>{t('usage_statistics.requests')}</th>
                <th>{t('usage_statistics.failed')}</th>
                <th>{t('usage_statistics.tokens')}</th>
                <th>{t('usage_statistics.cache')}</th>
                <th>{t('usage_statistics.first_byte')}</th>
                <th>{t('usage_statistics.tps')}</th>
              </tr>
            </thead>
            <tbody>
              {modelRows.length ? (
                modelRows.map((row: ModelUsageRow) => (
                  <tr key={`${row.apiName}:${row.name}`}>
                    <td className={styles.monoCell}>{row.name}</td>
                    <td className={styles.mutedCell}>{row.apiName}</td>
                    <td>{formatNumber(row.totalRequests)}</td>
                    <td>{formatNumber(row.failedRequests)}</td>
                    <td>{formatNumber(row.totalTokens)}</td>
                    <td>{formatPercent(row.cacheHitRate)}</td>
                    <td>{formatLatency(row.averageFirstByteLatencyMs)}</td>
                    <td>{formatTps(row.tps)}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={8} className={styles.emptyCell}>
                    {t('usage_statistics.no_model_rows')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <Card title={t('usage_statistics.recent_details')}>
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{t('usage_statistics.time')}</th>
                <th>{t('usage_statistics.status')}</th>
                <th>{t('usage_statistics.model')}</th>
                <th>{t('usage_statistics.api')}</th>
                <th>{t('usage_statistics.tokens')}</th>
                <th>{t('usage_statistics.first_byte')}</th>
                <th>{t('usage_statistics.latency')}</th>
                <th>{t('usage_statistics.source')}</th>
              </tr>
            </thead>
            <tbody>
              {recentRows.length ? (
                recentRows.map((row: DetailUsageRow) => (
                  <tr key={row.key}>
                    <td>{formatTimestamp(row.detail.timestamp)}</td>
                    <td>
                      <span className={row.detail.failed ? styles.statusFailed : styles.statusOk}>
                        {row.detail.failed ? t('usage_statistics.failed') : t('usage_statistics.success')}
                      </span>
                    </td>
                    <td className={styles.monoCell}>{row.modelName}</td>
                    <td className={styles.mutedCell}>{row.apiName}</td>
                    <td>{formatNumber(row.detail.tokens?.total_tokens)}</td>
                    <td>{formatLatency(row.detail.first_byte_latency_ms)}</td>
                    <td>{formatLatency(row.detail.latency_ms)}</td>
                    <td className={styles.mutedCell}>{row.detail.source || row.detail.auth_index || '-'}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={8} className={styles.emptyCell}>
                    {t('usage_statistics.no_recent_rows')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}
