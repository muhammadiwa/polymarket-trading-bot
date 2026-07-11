import { PortfolioOverview } from "@/components/portfolio/PortfolioOverview";
import { PositionList } from "@/components/positions/PositionList";
import { RiskStatus } from "@/components/risk/RiskStatus";
import { QuickActions } from "@/components/risk/QuickActions";
import { SystemHealth } from "@/components/health/SystemHealth";
import { OpportunityFeed } from "@/components/opportunities/OpportunityFeed";
import { AuthGuard } from "@/lib/auth/auth-guard";
import { ErrorBoundary } from "@/components/ui/ErrorBoundary";

export default function DashboardPage() {
  return (
    <AuthGuard>
      <ErrorBoundary>
        <main className="mx-auto max-w-7xl px-4 py-8 space-y-8">
          <header className="flex items-center justify-between">
            <h1 className="text-2xl font-bold text-white">PQAP Dashboard</h1>
          </header>
          <SystemHealth />
          <PortfolioOverview />
          <RiskStatus />
          <QuickActions />
          <PositionList />
          <OpportunityFeed />
        </main>
      </ErrorBoundary>
    </AuthGuard>
  );
}
