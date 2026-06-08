import { useState, useEffect, useCallback } from 'react'
import { Menu } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'
import { Sidebar } from '@/components/layout/Sidebar'
import { TopBar } from '@/components/layout/TopBar'
import { StatsCards } from '@/components/dashboard/StatsCards'
import { ScenarioRunner } from '@/components/scenarios/ScenarioRunner'
import { AlertLogsTable } from '@/components/records/AlertLogsTable'
import { FalsePositivesTable } from '@/components/records/FalsePositivesTable'
import { WhitelistTable } from '@/components/records/WhitelistTable'
import { useScenarios } from '@/hooks/useScenarios'
import { useRecords } from '@/hooks/useRecords'
import type { Scenario } from '@/types'

export default function App() {
  const [token, setToken] = useState('')
  const { scenarios, activeScenario, steps, results, runState, error, runScenario } = useScenarios()
  const { alerts, falsePositives, whitelist, stats, loading, refreshRecords } = useRecords()

  const handleRefresh = useCallback(() => {
    refreshRecords(token)
  }, [refreshRecords, token])

  const handleScenarioClick = useCallback(
    async (scenario: Scenario) => {
      await runScenario(scenario, token, () => refreshRecords(token))
    },
    [runScenario, token, refreshRecords]
  )

  useEffect(() => {
    refreshRecords(token)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden bg-[#0F1116]">
        {/* ---- Desktop Sidebar ---- */}
        <Sidebar
          scenarios={scenarios}
          activeScenarioId={activeScenario?.id ?? null}
          runState={runState}
          token={token}
          onTokenChange={setToken}
          onScenarioClick={handleScenarioClick}
          className="hidden lg:flex w-[260px] shrink-0"
        />

        {/* ---- Mobile top bar + Sheet drawer ---- */}
        <Sheet>
          <div className="lg:hidden fixed top-0 inset-x-0 z-30 flex items-center justify-between px-4 py-3 bg-[#0A0C10] border-b border-[#2D3748]">
            <div className="flex items-center gap-2">
              <div className="flex items-center justify-center w-7 h-7 rounded bg-[#3B82F6]/10">
                <span className="text-[#60A5FA] text-[11px] font-bold">DLP</span>
              </div>
              <span className="text-sm font-semibold text-[#F9FAFB]">DLP Console</span>
            </div>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon" className="lg:hidden h-8 w-8 text-[#9CA3AF] hover:text-[#D1D5DB] hover:bg-[#252A34]">
                <Menu className="w-4 h-4" />
              </Button>
            </SheetTrigger>
          </div>
          <SheetContent side="left" className="w-[280px] p-0 bg-[#0A0C10] border-[#2D3748]">
            <Sidebar
              scenarios={scenarios}
              activeScenarioId={activeScenario?.id ?? null}
              runState={runState}
              token={token}
              onTokenChange={setToken}
              onScenarioClick={handleScenarioClick}
              className="w-full h-full border-none"
            />
          </SheetContent>
        </Sheet>

        {/* ---- Main Content ---- */}
        <main className="flex-1 flex flex-col min-w-0 overflow-auto lg:pt-0 pt-14">
          <div className="flex-1 p-4 sm:p-6 lg:p-6 max-w-[1600px] w-full mx-auto animate-fade-up">
            <TopBar
              activeTitle={activeScenario?.title ?? '选择一个业务场景'}
              runState={runState}
              scenarioBadge={activeScenario?.id ?? '等待触发'}
              loading={loading}
              onRefresh={handleRefresh}
            />

            <StatsCards stats={stats} loading={loading} />

            <ScenarioRunner steps={steps} results={results} runState={runState} error={error} />

            <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
              <AlertLogsTable data={alerts} loading={loading} />
              <FalsePositivesTable data={falsePositives} loading={loading} />
              <WhitelistTable data={whitelist} loading={loading} />
            </div>
          </div>
        </main>
      </div>
      <Toaster />
    </TooltipProvider>
  )
}
