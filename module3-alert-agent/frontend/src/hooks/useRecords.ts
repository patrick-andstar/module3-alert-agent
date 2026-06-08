import { useState, useCallback } from 'react'
import type { AlertRecord, FalsePositiveRecord, WhitelistRule, AlertsQueryResponse, DashboardStats } from '@/types'

async function getJSON(path: string, token: string): Promise<unknown> {
  const headers: Record<string, string> = {}
  if (token.trim()) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const response = await fetch(path, { headers })
  if (!response.ok) return []
  return response.json()
}

export function useRecords() {
  const [alerts, setAlerts] = useState<AlertRecord[]>([])
  const [alertsTotal, setAlertsTotal] = useState(0)
  const [falsePositives, setFalsePositives] = useState<FalsePositiveRecord[]>([])
  const [whitelist, setWhitelist] = useState<WhitelistRule[]>([])
  const [loading, setLoading] = useState(false)

  const stats: DashboardStats = {
    alertCount: alertsTotal,
    fpCount: falsePositives.length,
    whitelistCount: whitelist.length,
  }

  const refreshRecords = useCallback(async (token: string) => {
    setLoading(true)
    try {
      const [alertsResult, fps, wl] = await Promise.all([
        (async (): Promise<AlertsQueryResponse> => {
          const response = await fetch('/api/alerts/query', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...(token.trim() ? { Authorization: `Bearer ${token}` } : {}),
            },
            body: JSON.stringify({ page: 1, page_size: 20, order_by: 'timestamp', order: 'desc' }),
          })
          if (!response.ok) return { data: [], total: 0, page: 1, page_size: 20 }
          return response.json()
        })(),
        getJSON('/api/false-positives', token) as Promise<FalsePositiveRecord[]>,
        getJSON('/api/whitelist', token) as Promise<WhitelistRule[]>,
      ])

      const alertRows = Array.isArray(alertsResult.data) ? alertsResult.data : []
      setAlerts(alertRows)
      setAlertsTotal(alertsResult.total || alertRows.length || 0)
      setFalsePositives(Array.isArray(fps) ? fps : [])
      setWhitelist(Array.isArray(wl) ? wl : [])
    } catch {
      // silently handle error
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    alerts,
    alertsTotal,
    falsePositives,
    whitelist,
    stats,
    loading,
    refreshRecords,
  }
}
