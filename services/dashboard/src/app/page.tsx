import { PortfolioOverview } from "@/components/portfolio/PortfolioOverview";
import { PositionList } from "@/components/positions/PositionList";
import { RiskStatus } from "@/components/risk/RiskStatus";
import { QuickActions } from "@/components/risk/QuickActions";
import { SystemHealth } from "@/components/health/SystemHealth";
import { OpportunityFeed } from "@/components/opportunities/OpportunityFeed";
import { AppShell } from "@/components/layout/AppShell";

export default function DashboardPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <SystemHealth />
        <PortfolioOverview />
        <RiskStatus />
        <QuickActions />
        <PositionList />
        <OpportunityFeed />
      </div>
    </AppShell>
  );
}
