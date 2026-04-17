package dashboard

const b2bPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Shinhan B2B Finance Coach</title>
<link href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet" />
<script src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
<script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
<script src="https://unpkg.com/recharts@2/umd/Recharts.js"></script>
<script src="https://unpkg.com/@babel/standalone/babel.min.js"></script>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #fafafa; font-family: "DM Sans", sans-serif; }
#root { min-height: 100vh; }
.loading-screen { display: flex; align-items: center; justify-content: center; height: 100vh; flex-direction: column; gap: 12px; color: #6b7280; }
.loading-screen .spinner { width: 32px; height: 32px; border: 3px solid #e5e7eb; border-top-color: #0f4c81; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<div id="root"><div class="loading-screen"><div class="spinner"></div><div>Loading B2B Dashboard...</div></div></div>
<script type="text/babel">
const { useState, useEffect, useMemo } = React;
const { AreaChart, Area, BarChart, Bar, LineChart, Line, PieChart, Pie, Cell, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine } = Recharts;

const fmt = n => new Intl.NumberFormat("vi-VN").format(n);
const fM = n => n >= 1e9 ? (n/1e9).toFixed(1)+"B" : n >= 1e6 ? Math.round(n/1e6)+"M" : fmt(n);

function useAPI(endpoint) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  useEffect(() => {
    fetch("/api/b2b/" + endpoint)
      .then(r => { if (!r.ok) throw new Error(r.statusText); return r.json(); })
      .then(d => { setData(d); setLoading(false); })
      .catch(e => { setError(e.message); setLoading(false); });
  }, [endpoint]);
  return { data, loading, error };
}

function B2BDashboard() {
  const [tab, setTab] = useState("overview");
  const [aiQ, setAiQ] = useState("");
  const [aiR, setAiR] = useState(null);
  const [aiLoading, setAiLoading] = useState(false);

  const companyAPI = useAPI("company");
  const healthAPI = useAPI("health");
  const cashflowAPI = useAPI("cashflow");
  const receivablesAPI = useAPI("receivables");
  const payablesAPI = useAPI("payables");
  const recommendationsAPI = useAPI("recommendations");
  const discountsAPI = useAPI("discounts");
  const arAgingAPI = useAPI("ar-aging");
  const monthlyAPI = useAPI("monthly-revenue");

  // Derive company object from API (first row)
  const COMPANY = companyAPI.data && Array.isArray(companyAPI.data) && companyAPI.data.length > 0 ? companyAPI.data[0] : null;
  const HEALTH = healthAPI.data && Array.isArray(healthAPI.data) ? healthAPI.data : [];
  const latestHealth = HEALTH.length > 0 ? HEALTH[0] : {};

  // Map cashflow data for charts
  const CASHFLOW_DATA = (cashflowAPI.data && Array.isArray(cashflowAPI.data) ? cashflowAPI.data : []).map(row => ({
    week: row.period_label || row.period_start || "",
    inflow: Number(row.inflow || 0) / 1e6,
    outflow: Number(row.outflow || 0) / 1e6,
    net: (Number(row.inflow || 0) - Number(row.outflow || 0)) / 1e6,
    balance: Number(row.balance || 0) / 1e6,
    alert: Number(row.balance || 0) < 400000000,
  })).reverse();

  const RECEIVABLES = (receivablesAPI.data && Array.isArray(receivablesAPI.data) ? receivablesAPI.data : []).map(r => ({
    customer: r.customer_name || r.customer || r.counterparty || "",
    amount: Number(r.amount || 0),
    due: r.due_date || "",
    days_overdue: Math.max(0, Number(r.calc_days_overdue || r.days_overdue || 0)),
    status: r.status || "outstanding",
    probability: Number(r.collection_probability || r.probability || 80),
  }));

  const PAYABLES = (payablesAPI.data && Array.isArray(payablesAPI.data) ? payablesAPI.data : []).map(p => ({
    vendor: p.vendor_name || p.vendor || p.counterparty || "",
    amount: Number(p.amount || 0),
    due: p.due_date || "",
    early_disc: Number(p.early_discount_pct || p.early_disc || 0),
    early_deadline: p.early_discount_deadline || p.early_deadline || null,
  }));

  const PRODUCT_RECS = (recommendationsAPI.data && Array.isArray(recommendationsAPI.data) ? recommendationsAPI.data : []).map(p => ({
    product: p.product_name || p.product || "",
    type: p.product_type || p.type || "",
    amount: Number(p.estimated_amount || p.amount || 0),
    trigger: p.trigger_reason || p.trigger || "",
    priority: p.priority || "medium",
    status: p.status || "new",
  }));

  const DISCOUNT_ANALYSIS = (discountsAPI.data && Array.isArray(discountsAPI.data) ? discountsAPI.data : []).map(d => ({
    segment: d.segment_name || d.segment || "",
    customers: Number(d.customer_count || d.customers || 0),
    share: Number(d.revenue_share_pct || d.share || 0),
    margin: Number(d.margin_pct || d.margin || 0),
    suggestion: d.ai_suggestion || d.suggestion || "",
    impact: d.expected_impact || d.impact || "",
  }));

  const MONTHLY_METRICS = (monthlyAPI.data && Array.isArray(monthlyAPI.data) ? monthlyAPI.data : []).map(m => ({
    month: m.month || "",
    revenue: Number(m.revenue || 0) / 1e6,
    expense: Number(m.expense || 0) / 1e6,
    profit: (Number(m.revenue || 0) - Number(m.expense || 0)) / 1e6,
  })).reverse();

  // Computed totals
  const totalAR = RECEIVABLES.reduce((s,r) => s + r.amount, 0);
  const overdueAR = RECEIVABLES.filter(r => r.days_overdue > 0).reduce((s,r) => s + r.amount, 0);
  const totalAP = PAYABLES.reduce((s,p) => s + p.amount, 0);
  const discountSavings = PAYABLES.filter(p => p.early_disc > 0).reduce((s,p) => s + p.amount * p.early_disc / 100, 0);

  const arAging = useMemo(() => {
    if (arAgingAPI.data && Array.isArray(arAgingAPI.data) && arAgingAPI.data.length > 0) {
      return arAgingAPI.data.map(r => ({ name: r.bucket, value: Number(r.total || 0) }));
    }
    const b = { current: 0, "1-30d": 0, "31-60d": 0, ">60d": 0 };
    RECEIVABLES.forEach(r => {
      if (r.days_overdue === 0) b.current += r.amount;
      else if (r.days_overdue <= 30) b["1-30d"] += r.amount;
      else if (r.days_overdue <= 60) b["31-60d"] += r.amount;
      else b[">60d"] += r.amount;
    });
    return Object.entries(b).map(([name, value]) => ({ name, value }));
  }, [arAgingAPI.data, RECEIVABLES]);

  // Health breakdown from latest health metrics
  const HEALTH_BREAKDOWN = latestHealth ? [
    { metric: "Profitability", score: Number(latestHealth.profitability_score || latestHealth.profitability || 0), detail: latestHealth.profitability_detail || "" },
    { metric: "Liquidity", score: Number(latestHealth.liquidity_score || latestHealth.liquidity || 0), detail: latestHealth.liquidity_detail || "" },
    { metric: "Efficiency", score: Number(latestHealth.efficiency_score || latestHealth.efficiency || 0), detail: latestHealth.efficiency_detail || "" },
    { metric: "Growth", score: Number(latestHealth.growth_score || latestHealth.growth || 0), detail: latestHealth.growth_detail || "" },
    { metric: "Stability", score: Number(latestHealth.stability_score || latestHealth.stability || 0), detail: latestHealth.stability_detail || "" },
  ].filter(h => h.score > 0) : [];

  const healthScore = Number(latestHealth.overall_score || latestHealth.health_score || (COMPANY && COMPANY.health_score) || 0);
  const riskGrade = latestHealth.risk_grade || (COMPANY && COMPANY.risk_grade) || "N/A";

  const handleAi = async () => {
    if (!aiQ.trim()) return;
    setAiLoading(true); setAiR(null);
    await new Promise(r => setTimeout(r, 1200));
    const q = aiQ.toLowerCase();
    let r = "";
    if (q.includes("cashflow") || q.includes("dong tien") || q.includes("tuan")) {
      r = "CASHFLOW FORECAST\n\nBased on your current data, review the Cashflow tab for a detailed weekly breakdown of inflows vs outflows.\n\nKey actions:\n1. Follow up on overdue receivables\n2. Negotiate payment terms with vendors\n3. Consider Shinhan Revolving Credit for short-term gaps";
    } else if (q.includes("discount") || q.includes("giam gia") || q.includes("khuyen mai")) {
      r = "DISCOUNT STRATEGY\n\nReview the Strategy tab for AI-recommended discount tiers by customer segment.\n\nKey insight: Focus discounts on mid-size restaurants for retention while protecting margins on small retailers.";
    } else if (q.includes("vay") || q.includes("loan") || q.includes("mo rong")) {
      r = "LOAN ASSESSMENT\n\nHealth Score: " + healthScore + "/100 (Grade " + riskGrade + ")\n\nCheck the Shinhan Products tab for tailored recommendations based on your financial data.";
    } else {
      r = "QUICK OVERVIEW\n\nRevenue trend: See Overview tab\nAR Outstanding: " + fM(totalAR) + " (Overdue: " + fM(overdueAR) + ")\nAP Due: " + fM(totalAP) + "\nHealth Score: " + healthScore + "\n\nAsk about: cashflow, discount strategy, loan assessment";
    }
    setAiR(r); setAiLoading(false);
  };

  const tabs = [
    { id: "overview", label: "Overview", icon: "\u{1F4CA}" },
    { id: "cashflow", label: "Cashflow", icon: "\u{1F4B0}" },
    { id: "arap", label: "AR / AP", icon: "\u{1F4CB}" },
    { id: "strategy", label: "Strategy", icon: "\u{1F3AF}" },
    { id: "products", label: "Shinhan Products", icon: "\u{1F3E6}" },
  ];

  const SH_BLUE = "#0f4c81";
  const SH_LIGHT = "#e8f0f8";

  const anyLoading = companyAPI.loading || healthAPI.loading || cashflowAPI.loading || receivablesAPI.loading || payablesAPI.loading;
  const anyError = companyAPI.error || receivablesAPI.error;

  if (anyError) {
    return (
      <div style={{ fontFamily: '"DM Sans", sans-serif', maxWidth: 600, margin: "80px auto", textAlign: "center" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>{"\u{26A0}\u{FE0F}"}</div>
        <h2 style={{ fontSize: 18, fontWeight: 600, color: "#991b1b", marginBottom: 8 }}>B2B Database Not Available</h2>
        <p style={{ fontSize: 13, color: "#6b7280", marginBottom: 20 }}>{anyError}</p>
        <p style={{ fontSize: 12, color: "#9ca3af" }}>Install the shinhan-b2b-coach skill first:</p>
        <code style={{ display: "inline-block", padding: "8px 16px", background: "#f3f4f6", borderRadius: 6, fontSize: 12, marginTop: 8 }}>clawkit install shinhan-b2b-coach</code>
        <div style={{ marginTop: 24 }}><a href="/" style={{ color: SH_BLUE, fontSize: 13 }}>{"\u2190"} Back to Clawkit Dashboard</a></div>
      </div>
    );
  }

  const companyName = COMPANY ? (COMPANY.name || COMPANY.company_name || "") : "Loading...";
  const companyIndustry = COMPANY ? (COMPANY.industry || "") : "";
  const companyEmployees = COMPANY ? (COMPANY.employees || COMPANY.employee_count || "") : "";
  const companyTaxCode = COMPANY ? (COMPANY.tax_code || COMPANY.tax_id || "") : "";
  const cashReserve = COMPANY ? Number(COMPANY.cash_reserve || COMPANY.cash_balance || 0) : 0;
  const monthlyRevAvg = COMPANY ? Number(COMPANY.monthly_revenue_avg || 0) : 0;

  const latestMetric = MONTHLY_METRICS.length > 0 ? MONTHLY_METRICS[MONTHLY_METRICS.length - 1] : null;
  const prevMetric = MONTHLY_METRICS.length > 1 ? MONTHLY_METRICS[MONTHLY_METRICS.length - 2] : null;
  const revenueChange = (latestMetric && prevMetric && prevMetric.revenue > 0) ? ((latestMetric.revenue - prevMetric.revenue) / prevMetric.revenue * 100).toFixed(1) : null;
  const netMargin = (latestMetric && latestMetric.revenue > 0) ? ((latestMetric.profit / latestMetric.revenue) * 100).toFixed(1) : "N/A";

  return (
    <div style={{ fontFamily: '"DM Sans", sans-serif', maxWidth: 1200, margin: "0 auto", padding: "0 12px" }}>

      {/* Back link */}
      <div style={{ padding: "8px 0", fontSize: 11 }}>
        <a href="/" style={{ color: SH_BLUE, textDecoration: "none" }}>{"\u2190"} Back to Clawkit Dashboard</a>
      </div>

      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "16px 0 12px", borderBottom: "2px solid " + SH_BLUE }}>
        <div>
          <div style={{ fontSize: 10, fontWeight: 600, letterSpacing: 2, color: SH_BLUE, textTransform: "uppercase" }}>Shinhan SOL {"\u00B7"} B2B Finance Coach</div>
          <div style={{ fontSize: 18, fontWeight: 700, color: "#0a0a0a", marginTop: 2 }}>{companyName}</div>
          <div style={{ fontSize: 11, color: "#6b7280", marginTop: 1 }}>{companyIndustry}{companyEmployees ? " \u00B7 " + companyEmployees + " employees" : ""}{companyTaxCode ? " \u00B7 MST: " + companyTaxCode : ""}</div>
        </div>
        <div style={{ textAlign: "right" }}>
          <div style={{ fontSize: 32, fontWeight: 700, color: healthScore >= 70 ? "#059669" : "#d97706" }}>{healthScore || (anyLoading ? "..." : "N/A")}</div>
          <div style={{ fontSize: 10, color: "#6b7280" }}>Health Score</div>
          {riskGrade !== "N/A" && <div style={{ fontSize: 11, fontWeight: 600, padding: "2px 8px", borderRadius: 4, background: "#059669", color: "#fff", display: "inline-block", marginTop: 2 }}>Grade {riskGrade}</div>}
        </div>
      </div>

      {/* AI Chat Bar */}
      <div style={{ margin: "12px 0", padding: 12, borderRadius: 10, background: SH_LIGHT, border: "1px solid #bfdbfe" }}>
        <div style={{ display: "flex", gap: 8 }}>
          <span style={{ fontSize: 18, lineHeight: "36px" }}>{"\u{1F916}"}</span>
          <input value={aiQ} onChange={e => setAiQ(e.target.value)} onKeyDown={e => e.key === "Enter" && handleAi()}
            placeholder="Ask AI: cashflow forecast, discount strategy, loan assessment..."
            style={{ flex: 1, padding: "8px 12px", borderRadius: 8, border: "1px solid #bfdbfe", fontSize: 12, outline: "none", fontFamily: "inherit" }} />
          <button onClick={handleAi} disabled={aiLoading}
            style={{ padding: "8px 16px", borderRadius: 8, background: SH_BLUE, color: "#fff", border: "none", fontSize: 12, fontWeight: 600, cursor: "pointer", opacity: aiLoading ? 0.6 : 1, whiteSpace: "nowrap" }}>
            {aiLoading ? "..." : "Ask AI"}
          </button>
        </div>
        {aiR && <div style={{ marginTop: 10, padding: 12, borderRadius: 8, background: "#fff", border: "1px solid #dbeafe", whiteSpace: "pre-wrap", fontSize: 11, lineHeight: 1.7, color: "#1e293b" }}>{aiR}</div>}
      </div>

      {/* Tabs */}
      <div style={{ display: "flex", gap: 0, borderBottom: "1px solid #e5e7eb" }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)}
            style={{ padding: "8px 14px", fontSize: 12, fontWeight: tab === t.id ? 600 : 400, color: tab === t.id ? SH_BLUE : "#6b7280", background: "none", border: "none", borderBottom: tab === t.id ? "2px solid " + SH_BLUE : "2px solid transparent", cursor: "pointer" }}>
            {t.icon} {t.label}
          </button>
        ))}
      </div>

      <div style={{ marginTop: 14 }}>

        {/* ==================== OVERVIEW ==================== */}
        {tab === "overview" && (<div>
          {anyLoading ? <div style={{textAlign:"center",padding:40,color:"#6b7280"}}>Loading data...</div> : (<>
          {/* KPIs */}
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))", gap: 8, marginBottom: 14 }}>
            {[
              { label: "Monthly Revenue", value: latestMetric ? fM(latestMetric.revenue * 1e6) : "N/A", sub: revenueChange ? (revenueChange > 0 ? "+" : "") + revenueChange + "% vs last month" : "", color: "#059669" },
              { label: "Net Margin", value: netMargin + "%", sub: "From transactions data", color: SH_BLUE },
              { label: "Cash Reserve", value: fM(cashReserve), sub: monthlyRevAvg > 0 ? "~" + Math.round(cashReserve / (monthlyRevAvg / 30)) + " days runway" : "", color: "#d97706" },
              { label: "AR Outstanding", value: fM(totalAR), sub: fM(overdueAR) + " overdue", color: "#dc2626" },
              { label: "AP Due", value: fM(totalAP), sub: fM(discountSavings) + " disc. available", color: "#7c3aed" },
            ].map((k, i) => (
              <div key={i} style={{ padding: 12, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
                <div style={{ fontSize: 10, color: "#6b7280" }}>{k.label}</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: k.color, marginTop: 2 }}>{k.value}</div>
                <div style={{ fontSize: 10, color: "#9ca3af" }}>{k.sub}</div>
              </div>
            ))}
          </div>

          {/* Revenue Trend + Health */}
          <div style={{ display: "grid", gridTemplateColumns: HEALTH_BREAKDOWN.length > 0 ? "2fr 1fr" : "1fr", gap: 12, marginBottom: 14 }}>
            <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
              <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Revenue & Profit (millions VND)</div>
              {MONTHLY_METRICS.length > 0 ? (
              <ResponsiveContainer width="100%" height={180}>
                <BarChart data={MONTHLY_METRICS} barGap={2}>
                  <XAxis dataKey="month" tick={{ fontSize: 10 }} axisLine={false} tickLine={false} />
                  <YAxis tick={{ fontSize: 9 }} axisLine={false} tickLine={false} />
                  <Tooltip formatter={(v) => Math.round(v) + "M VND"} />
                  <Bar dataKey="revenue" fill={SH_BLUE} radius={[3,3,0,0]} name="Revenue" />
                  <Bar dataKey="profit" fill="#059669" radius={[3,3,0,0]} name="Profit" />
                </BarChart>
              </ResponsiveContainer>
              ) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No transaction data yet</div>}
            </div>
            {HEALTH_BREAKDOWN.length > 0 && (
            <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
              <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Health breakdown</div>
              {HEALTH_BREAKDOWN.map((h, i) => (
                <div key={i} style={{ marginBottom: 8 }}>
                  <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, marginBottom: 2 }}>
                    <span style={{ fontWeight: 500, color: "#374151" }}>{h.metric}</span>
                    <span style={{ fontWeight: 600, color: h.score >= 75 ? "#059669" : h.score >= 60 ? "#d97706" : "#dc2626" }}>{h.score}</span>
                  </div>
                  <div style={{ height: 5, background: "#f3f4f6", borderRadius: 3 }}>
                    <div style={{ height: "100%", width: h.score + "%", borderRadius: 3, background: h.score >= 75 ? "#059669" : h.score >= 60 ? "#d97706" : "#dc2626" }}></div>
                  </div>
                  {h.detail && <div style={{ fontSize: 9, color: "#9ca3af", marginTop: 1 }}>{h.detail}</div>}
                </div>
              ))}
            </div>
            )}
          </div>
          </>)}
        </div>)}

        {/* ==================== CASHFLOW ==================== */}
        {tab === "cashflow" && (<div>
          <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff", marginBottom: 14 }}>
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Cashflow forecast (millions VND)</div>
            {CASHFLOW_DATA.length > 0 ? (
            <ResponsiveContainer width="100%" height={240}>
              <AreaChart data={CASHFLOW_DATA}>
                <defs>
                  <linearGradient id="gIn" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#059669" stopOpacity={0.2} /><stop offset="100%" stopColor="#059669" stopOpacity={0} /></linearGradient>
                  <linearGradient id="gOut" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#dc2626" stopOpacity={0.2} /><stop offset="100%" stopColor="#dc2626" stopOpacity={0} /></linearGradient>
                </defs>
                <XAxis dataKey="week" tick={{ fontSize: 9 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 9 }} axisLine={false} tickLine={false} />
                <Tooltip formatter={v => Math.round(v) + "M"} />
                <ReferenceLine y={400} stroke="#dc2626" strokeDasharray="4 4" label={{ value: "Min safe", fontSize: 9, fill: "#dc2626" }} />
                <Area type="monotone" dataKey="inflow" stroke="#059669" fill="url(#gIn)" strokeWidth={2} name="Inflow" />
                <Area type="monotone" dataKey="outflow" stroke="#dc2626" fill="url(#gOut)" strokeWidth={2} name="Outflow" />
                <Line type="monotone" dataKey="balance" stroke={SH_BLUE} strokeWidth={2.5} dot={{ r: 3 }} name="Balance" />
              </AreaChart>
            </ResponsiveContainer>
            ) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No cashflow forecast data yet</div>}
          </div>

          {/* Weekly detail from payables/receivables */}
          {PAYABLES.length > 0 && (
          <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Upcoming Payables</div>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 11 }}>
              <thead><tr style={{ borderBottom: "2px solid #e5e7eb" }}>
                {["Vendor", "Amount", "Due Date", "Early Discount"].map(h => (
                  <th key={h} style={{ textAlign: "left", padding: "6px 4px", color: "#6b7280", fontWeight: 500, fontSize: 10 }}>{h}</th>
                ))}
              </tr></thead>
              <tbody>
                {PAYABLES.map((p, i) => (
                  <tr key={i} style={{ borderBottom: "1px solid #f3f4f6" }}>
                    <td style={{ padding: "6px 4px", fontWeight: 500 }}>{p.vendor}</td>
                    <td style={{ padding: "6px 4px" }}>{fM(p.amount)}</td>
                    <td style={{ padding: "6px 4px" }}>{p.due}</td>
                    <td style={{ padding: "6px 4px" }}>{p.early_disc > 0 ? <span style={{color:"#059669",fontWeight:600}}>{p.early_disc}% by {p.early_deadline}</span> : <span style={{color:"#9ca3af"}}>None</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          )}
        </div>)}

        {/* ==================== AR/AP ==================== */}
        {tab === "arap" && (<div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 14 }}>
            {/* AR Aging */}
            <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
              <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Accounts Receivable Aging</div>
              {arAging.some(a => a.value > 0) ? (
              <ResponsiveContainer width="100%" height={160}>
                <BarChart data={arAging}>
                  <XAxis dataKey="name" tick={{ fontSize: 10 }} axisLine={false} tickLine={false} />
                  <YAxis tick={{ fontSize: 9 }} axisLine={false} tickLine={false} tickFormatter={v => fM(v)} />
                  <Tooltip formatter={v => fmt(v) + " VND"} />
                  <Bar dataKey="value" radius={[4,4,0,0]}>
                    {arAging.map((_, i) => <Cell key={i} fill={["#059669","#d97706","#f97316","#dc2626"][i]} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
              ) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No aging data</div>}
            </div>
            {/* AP with discount opportunities */}
            <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
              <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Payables -- Early-Pay Discount Opportunities</div>
              {PAYABLES.filter(p => p.early_disc > 0).length > 0 ? PAYABLES.filter(p => p.early_disc > 0).map((p, i) => (
                <div key={i} style={{ padding: 8, borderRadius: 6, background: "#f0fdf4", border: "1px solid #bbf7d0", marginBottom: 6 }}>
                  <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11 }}>
                    <span style={{ fontWeight: 500 }}>{p.vendor}</span>
                    <span style={{ fontWeight: 600, color: "#059669" }}>Save {fM(p.amount * p.early_disc / 100)}</span>
                  </div>
                  <div style={{ fontSize: 10, color: "#6b7280", marginTop: 2 }}>
                    {fM(p.amount)} {"\u00B7"} {p.early_disc}% discount {"\u00B7"} Pay by {p.early_deadline} (vs. {p.due})
                  </div>
                </div>
              )) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No discount opportunities</div>}
              {discountSavings > 0 && (
              <div style={{ padding: 6, background: "#ecfdf5", borderRadius: 4, fontSize: 10, fontWeight: 600, color: "#059669", textAlign: "center", marginTop: 4 }}>
                Total savings: {fM(discountSavings)}/month {"\u00B7"} {fM(discountSavings * 12)}/year
              </div>
              )}
            </div>
          </div>

          {/* AR Detail Table */}
          {RECEIVABLES.length > 0 && (
          <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Receivables Detail</div>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 11 }}>
              <thead><tr style={{ borderBottom: "2px solid #e5e7eb" }}>
                {["Customer", "Amount", "Due Date", "Overdue", "Probability"].map(h => (
                  <th key={h} style={{ textAlign: "left", padding: "6px 4px", color: "#6b7280", fontWeight: 500, fontSize: 10 }}>{h}</th>
                ))}
              </tr></thead>
              <tbody>
                {RECEIVABLES.sort((a,b) => b.days_overdue - a.days_overdue).map((r, i) => (
                  <tr key={i} style={{ borderBottom: "1px solid #f3f4f6" }}>
                    <td style={{ padding: "6px 4px", fontWeight: 500 }}>{r.customer}</td>
                    <td style={{ padding: "6px 4px" }}>{fM(r.amount)}</td>
                    <td style={{ padding: "6px 4px" }}>{r.due}</td>
                    <td style={{ padding: "6px 4px" }}>
                      {r.days_overdue > 0 ? <span style={{ padding: "1px 6px", borderRadius: 3, fontSize: 10, fontWeight: 600, background: r.days_overdue > 30 ? "#fef2f2" : "#fffbeb", color: r.days_overdue > 30 ? "#dc2626" : "#d97706" }}>{r.days_overdue}d</span> : <span style={{ color: "#059669", fontSize: 10 }}>On time</span>}
                    </td>
                    <td style={{ padding: "6px 4px" }}>
                      <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                        <div style={{ width: 40, height: 4, background: "#f3f4f6", borderRadius: 2 }}>
                          <div style={{ height: "100%", width: r.probability + "%", borderRadius: 2, background: r.probability >= 80 ? "#059669" : r.probability >= 60 ? "#d97706" : "#dc2626" }}></div>
                        </div>
                        <span style={{ fontSize: 10, color: "#6b7280" }}>{r.probability}%</span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          )}
        </div>)}

        {/* ==================== STRATEGY ==================== */}
        {tab === "strategy" && (<div>
          {DISCOUNT_ANALYSIS.length > 0 ? (
          <div style={{ padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff", marginBottom: 14 }}>
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>Customer Discount Strategy -- AI Recommended</div>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 11 }}>
              <thead><tr style={{ borderBottom: "2px solid #e5e7eb" }}>
                {["Segment", "Customers", "Revenue %", "Margin", "AI Suggestion", "Expected Impact"].map(h => (
                  <th key={h} style={{ textAlign: "left", padding: "6px 4px", color: "#6b7280", fontWeight: 500, fontSize: 10 }}>{h}</th>
                ))}
              </tr></thead>
              <tbody>
                {DISCOUNT_ANALYSIS.map((d, i) => (
                  <tr key={i} style={{ borderBottom: "1px solid #f3f4f6" }}>
                    <td style={{ padding: "8px 4px", fontWeight: 500 }}>{d.segment}</td>
                    <td style={{ padding: "8px 4px" }}>{d.customers}</td>
                    <td style={{ padding: "8px 4px" }}>{d.share}%</td>
                    <td style={{ padding: "8px 4px" }}>{d.margin}%</td>
                    <td style={{ padding: "8px 4px" }}><span style={{ padding: "2px 6px", borderRadius: 4, fontSize: 10, background: SH_LIGHT, color: SH_BLUE }}>{d.suggestion}</span></td>
                    <td style={{ padding: "8px 4px", fontWeight: 500, color: "#059669" }}>{d.impact}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          ) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No discount strategy data yet</div>}
        </div>)}

        {/* ==================== SHINHAN PRODUCTS ==================== */}
        {tab === "products" && (<div>
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 10 }}>AI-Generated Product Recommendations</div>
          {PRODUCT_RECS.length > 0 ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            {PRODUCT_RECS.map((p, i) => (
              <div key={i} style={{ padding: 14, borderRadius: 8, border: "1px solid " + (p.priority === "high" ? "#fca5a5" : p.priority === "medium" ? "#fde68a" : "#e5e7eb"), background: "#fff" }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start" }}>
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 600, color: SH_BLUE }}>{p.product}</div>
                    <div style={{ fontSize: 11, color: "#6b7280", marginTop: 2 }}>Estimated: {fM(p.amount)} VND</div>
                  </div>
                  <div style={{ display: "flex", gap: 4 }}>
                    <span style={{ fontSize: 9, padding: "2px 8px", borderRadius: 4, fontWeight: 600, background: p.priority === "high" ? "#fef2f2" : p.priority === "medium" ? "#fffbeb" : "#f9fafb", color: p.priority === "high" ? "#dc2626" : p.priority === "medium" ? "#d97706" : "#6b7280" }}>{p.priority}</span>
                    <span style={{ fontSize: 9, padding: "2px 8px", borderRadius: 4, fontWeight: 500, background: p.status === "new" ? "#eff6ff" : "#f0fdf4", color: p.status === "new" ? "#2563eb" : "#059669" }}>{p.status}</span>
                  </div>
                </div>
                <div style={{ fontSize: 11, color: "#374151", marginTop: 8, padding: 8, borderRadius: 6, background: "#f9fafb" }}>
                  <span style={{ fontWeight: 500 }}>Trigger:</span> {p.trigger}
                </div>
              </div>
            ))}
          </div>
          ) : <div style={{textAlign:"center",padding:40,color:"#9ca3af",fontSize:12}}>No product recommendations yet</div>}

          {/* Product Pipeline Summary */}
          {PRODUCT_RECS.length > 0 && (
          <div style={{ marginTop: 14, padding: 14, borderRadius: 8, border: "1px solid #e5e7eb", background: "#fff" }}>
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8 }}>Pipeline value by priority</div>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              {["high", "medium", "low"].map(pri => {
                const items = PRODUCT_RECS.filter(p => p.priority === pri);
                const total = items.reduce((s, p) => s + p.amount, 0);
                if (items.length === 0) return null;
                const color = pri === "high" ? "#dc2626" : pri === "medium" ? "#d97706" : "#6b7280";
                return (
                  <div key={pri} style={{ flex: 1, minWidth: 120, padding: 10, borderRadius: 6, border: "1px solid " + color + "20", background: color + "08" }}>
                    <div style={{ fontSize: 10, color: color, fontWeight: 600, textTransform: "capitalize" }}>{pri} Priority</div>
                    <div style={{ fontSize: 18, fontWeight: 700, color: color }}>{fM(total)}</div>
                    <div style={{ fontSize: 9, color: "#9ca3af" }}>{items.length} {items.length === 1 ? "opportunity" : "opportunities"}</div>
                  </div>
                );
              })}
            </div>
            <div style={{ marginTop: 10, padding: 8, borderRadius: 6, background: SH_LIGHT, textAlign: "center" }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: SH_BLUE }}>Total pipeline: {fM(PRODUCT_RECS.reduce((s, p) => s + p.amount, 0))} VND</span>
            </div>
          </div>
          )}
        </div>)}

      </div>

      <div style={{ textAlign: "center", padding: "16px 0", fontSize: 10, color: "#9ca3af", borderTop: "1px solid #e5e7eb", marginTop: 20 }}>
        Shinhan SOL B2B Finance Coach {"\u00B7"} AI-powered financial advisory for SMEs {"\u00B7"} Data from shinhan-b2b-coach skill
      </div>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<B2BDashboard />);
</script>
</body>
</html>` + ""
