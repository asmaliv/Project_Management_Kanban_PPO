import React, { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../auth/useAuth'
import { fetchRisks, fetchRiskSummary, closeRisk } from './api'
import { fetchProjects } from '../projects/api'
import { RiskHeatmap } from './RiskHeatmap'
import { RiskTable } from './RiskTable'
import { CreateRiskModal } from './CreateRiskModal'
import { AssessRiskModal } from './AssessRiskModal'
import { MitigationModal } from './MitigationModal'
import { RiskDetailDrawer } from './RiskDetailDrawer'
import type { Risk, RiskSummary } from './types'
import type { Project } from '../projects/types'

export const RiskRegisterPage: React.FC = () => {
  const { user } = useAuth()
  const userRole = user?.role || ''

  const [activeTab, setActiveTab] = useState<'table' | 'heatmap'>('table')
  const [risks, setRisks] = useState<Risk[]>([])
  const [totalRisks, setTotalRisks] = useState(0)
  const [page, setPage] = useState(1)
  const [perPage] = useState(20)
  const [isLoading, setIsLoading] = useState(true)
  const [summary, setSummary] = useState<RiskSummary | null>(null)
  const [isSummaryLoading, setIsSummaryLoading] = useState(true)

  // Filters
  const [projects, setProjects] = useState<Project[]>([])
  const [filterProject, setFilterProject] = useState('')
  const [filterLevel, setFilterLevel] = useState('')
  const [filterStatus, setFilterStatus] = useState('')
  const [filterCategory, setFilterCategory] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCell, setSelectedCell] = useState<{ likelihood: number; impact: number } | null>(null)

  // Modals & Drawers
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [selectedRisk, setSelectedRisk] = useState<Risk | null>(null)
  const [isDetailDrawerOpen, setIsDetailDrawerOpen] = useState(false)
  const [isAssessModalOpen, setIsAssessModalOpen] = useState(false)
  const [isMitigateModalOpen, setIsMitigateModalOpen] = useState(false)
  const [riskToAssess, setRiskToAssess] = useState<Risk | null>(null)
  const [riskToMitigate, setRiskToMitigate] = useState<Risk | null>(null)

  // Notification Banner
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null)

  const showNotification = (type: 'success' | 'error', message: string) => {
    setNotification({ type, message })
    setTimeout(() => {
      setNotification(null)
    }, 4500)
  }

  // Load projects for filter
  useEffect(() => {
    fetchProjects()
      .then((data) => setProjects(data))
      .catch(() => {})
  }, [])

  // Load summary
  const loadSummary = useCallback(async () => {
    setIsSummaryLoading(true)
    try {
      const data = await fetchRiskSummary(filterProject || undefined)
      setSummary(data)
    } catch {
      // Ignore
    } finally {
      setIsSummaryLoading(false)
    }
  }, [filterProject])

  // Load risks
  const loadRisks = useCallback(async () => {
    setIsLoading(true)
    try {
      const res = await fetchRisks({
        project_id: filterProject || undefined,
        level: filterLevel || undefined,
        status: filterStatus || undefined,
        category: filterCategory || undefined,
        search: searchQuery.trim() || undefined,
        page,
        per_page: perPage,
      })
      setRisks(res.risks)
      setTotalRisks(res.total)
    } catch (err) {
      showNotification('error', err instanceof Error ? err.message : 'Gagal memuat daftar risiko.')
    } finally {
      setIsLoading(false)
    }
  }, [filterProject, filterLevel, filterStatus, filterCategory, searchQuery, page, perPage])

  useEffect(() => {
    loadRisks()
  }, [loadRisks])

  useEffect(() => {
    loadSummary()
  }, [loadSummary])

  // If a cell is clicked on the heatmap, filter table
  const handleSelectHeatmapCell = (cell: { likelihood: number; impact: number } | null) => {
    setSelectedCell(cell)
    if (cell) {
      // Find score/level for this cell or filter client-side/query
      const score = cell.likelihood * cell.impact
      let lvl = 'low'
      if (score <= 4) lvl = 'low'
      else if (score <= 9) lvl = 'medium'
      else if (score <= 15) lvl = 'high'
      else lvl = 'critical'
      setFilterLevel(lvl)
    } else {
      setFilterLevel('')
    }
  }

  const handleResetFilters = () => {
    setFilterProject('')
    setFilterLevel('')
    setFilterStatus('')
    setFilterCategory('')
    setSearchQuery('')
    setSelectedCell(null)
    setPage(1)
  }

  const handleCloseRiskAction = async (target: Risk) => {
    if (!window.confirm(`Yakin ingin menutup risiko ${target.code} (${target.title})?`)) {
      return
    }

    try {
      await closeRisk(target.id)
      showNotification('success', `Risiko ${target.code} berhasil ditutup.`)
      loadRisks()
      loadSummary()
      if (selectedRisk?.id === target.id) {
        setIsDetailDrawerOpen(false)
      }
    } catch (err) {
      showNotification('error', err instanceof Error ? err.message : 'Gagal menutup risiko.')
    }
  }

  const canRaiseRisk =
    userRole === 'project_manager' ||
    userRole === 'team_leader' ||
    userRole === 'risk_management'

  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {notification && (
        <div
          className={`p-4 rounded-xl text-sm font-medium flex items-center justify-between border shadow-sm ${
            notification.type === 'success'
              ? 'bg-emerald-50 text-emerald-800 border-emerald-200 dark:bg-emerald-950/60 dark:text-emerald-300 dark:border-emerald-800'
              : 'bg-rose-50 text-rose-800 border-rose-200 dark:bg-rose-950/60 dark:text-rose-300 dark:border-rose-800'
          }`}
        >
          <span>{notification.message}</span>
          <button
            onClick={() => setNotification(null)}
            className="text-xs opacity-70 hover:opacity-100 ml-4"
          >
            ✕
          </button>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Risk Register & Heatmap
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Pantau, nilai, mitigasi, dan kelola risiko proyek secara proaktif dan patuh standar perbankan.
          </p>
        </div>

        {canRaiseRisk && (
          <button
            type="button"
            onClick={() => setIsCreateModalOpen(true)}
            className="self-start sm:self-auto inline-flex items-center gap-2 px-4 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold shadow-sm transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
          >
            <span>+</span>
            <span>Ajukan Risiko</span>
          </button>
        )}
      </div>

      {/* Navigation Tabs */}
      <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-800">
        <button
          type="button"
          onClick={() => setActiveTab('table')}
          className={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-all flex items-center gap-2 ${
            activeTab === 'table'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <span>Daftar Risiko</span>
          <span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 font-mono">
            {totalRisks}
          </span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('heatmap')}
          className={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-all flex items-center gap-2 ${
            activeTab === 'heatmap'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <span>Matriks Heatmap (5×5)</span>
        </button>
      </div>

      {/* Heatmap Section */}
      {activeTab === 'heatmap' && (
        <RiskHeatmap
          summary={summary}
          isLoading={isSummaryLoading}
          selectedCell={selectedCell}
          onSelectCell={handleSelectHeatmapCell}
        />
      )}

      {/* Filters Bar */}
      <div className="bg-white dark:bg-slate-900 p-4 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="flex-1 min-w-[200px]">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value)
              setPage(1)
            }}
            placeholder="Cari kode, judul, atau deskripsi risiko..."
            className="w-full text-xs px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>

        {/* Project Filter */}
        <div className="w-48">
          <select
            value={filterProject}
            onChange={(e) => {
              setFilterProject(e.target.value)
              setPage(1)
            }}
            className="w-full text-xs px-2.5 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="">Semua Proyek</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.code} - {p.name}
              </option>
            ))}
          </select>
        </div>

        {/* Level Filter */}
        <div className="w-36">
          <select
            value={filterLevel}
            onChange={(e) => {
              setFilterLevel(e.target.value)
              setSelectedCell(null)
              setPage(1)
            }}
            className="w-full text-xs px-2.5 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="">Semua Level</option>
            <option value="low">Low (1–4)</option>
            <option value="medium">Medium (5–9)</option>
            <option value="high">High (10–15)</option>
            <option value="critical">Critical (16–25)</option>
          </select>
        </div>

        {/* Status Filter */}
        <div className="w-36">
          <select
            value={filterStatus}
            onChange={(e) => {
              setFilterStatus(e.target.value)
              setPage(1)
            }}
            className="w-full text-xs px-2.5 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="">Semua Status</option>
            <option value="identified">Teridentifikasi</option>
            <option value="assessed">Dinilai</option>
            <option value="mitigating">Mitigasi</option>
            <option value="accepted">Diterima</option>
            <option value="closed">Ditutup</option>
          </select>
        </div>

        {/* Category Filter */}
        <div className="w-36">
          <select
            value={filterCategory}
            onChange={(e) => {
              setFilterCategory(e.target.value)
              setPage(1)
            }}
            className="w-full text-xs px-2.5 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="">Semua Kategori</option>
            <option value="operational">Operasional</option>
            <option value="schedule">Jadwal</option>
            <option value="technical">Teknis</option>
            <option value="compliance">Kepatuhan</option>
            <option value="financial">Finansial</option>
            <option value="security">Keamanan</option>
          </select>
        </div>

        {/* Reset */}
        {(filterProject || filterLevel || filterStatus || filterCategory || searchQuery || selectedCell) && (
          <button
            type="button"
            onClick={handleResetFilters}
            className="text-xs text-rose-600 dark:text-rose-400 hover:underline px-2 font-medium"
          >
            Reset
          </button>
        )}
      </div>

      {/* Risk Table */}
      <RiskTable
        risks={risks}
        total={totalRisks}
        page={page}
        perPage={perPage}
        isLoading={isLoading}
        userRole={userRole}
        onPageChange={(newPage) => setPage(newPage)}
        onSelectRisk={(r) => {
          setSelectedRisk(r)
          setIsDetailDrawerOpen(true)
        }}
        onAssessRisk={(r) => {
          setRiskToAssess(r)
          setIsAssessModalOpen(true)
        }}
        onUpdateMitigation={(r) => {
          setRiskToMitigate(r)
          setIsMitigateModalOpen(true)
        }}
        onCloseRisk={handleCloseRiskAction}
      />

      {/* Modals & Drawer */}
      <CreateRiskModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={(created) => {
          showNotification('success', `Risiko ${created.code} berhasil diajukan.`)
          loadRisks()
          loadSummary()
        }}
      />

      <AssessRiskModal
        isOpen={isAssessModalOpen}
        risk={riskToAssess}
        onClose={() => {
          setIsAssessModalOpen(false)
          setRiskToAssess(null)
        }}
        onSuccess={(updated) => {
          showNotification('success', `Risiko ${updated.code} berhasil dinilai: Skor ${updated.score} (${updated.level?.toUpperCase()}).`)
          loadRisks()
          loadSummary()
          if (selectedRisk?.id === updated.id) {
            setSelectedRisk(updated)
          }
        }}
      />

      <MitigationModal
        isOpen={isMitigateModalOpen}
        risk={riskToMitigate}
        onClose={() => {
          setIsMitigateModalOpen(false)
          setRiskToMitigate(null)
        }}
        onSuccess={(updated) => {
          showNotification('success', `Rencana mitigasi untuk ${updated.code} berhasil diperbarui.`)
          loadRisks()
          loadSummary()
          if (selectedRisk?.id === updated.id) {
            setSelectedRisk(updated)
          }
        }}
      />

      <RiskDetailDrawer
        isOpen={isDetailDrawerOpen}
        risk={selectedRisk}
        userRole={userRole}
        onClose={() => {
          setIsDetailDrawerOpen(false)
          setSelectedRisk(null)
        }}
        onAssess={(r) => {
          setRiskToAssess(r)
          setIsAssessModalOpen(true)
        }}
        onMitigate={(r) => {
          setRiskToMitigate(r)
          setIsMitigateModalOpen(true)
        }}
        onCloseRisk={handleCloseRiskAction}
      />
    </div>
  )
}
export default RiskRegisterPage
