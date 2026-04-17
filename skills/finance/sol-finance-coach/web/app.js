(function () {
  const CUSTOMERS = [
    { user_id: "U001", name: "Nguyễn Văn Minh", income: 35000000, monthly_fixed: 8000000, goal: "mua nhà", risk_level: "moderate", knowledge_level: "intermediate", credit_cards: "Techcombank Visa Platinum", onboarded: "true", segment: "Young Professional", age: 28, city: "HCMC" },
    { user_id: "U002", name: "Trần Thị Lan", income: 18000000, monthly_fixed: 5000000, goal: "tiết kiệm", risk_level: "conservative", knowledge_level: "beginner", credit_cards: "", onboarded: "true", segment: "Budget Conscious", age: 24, city: "Hanoi" },
    { user_id: "U003", name: "Lê Hoàng Nam", income: 65000000, monthly_fixed: 20000000, goal: "đầu tư", risk_level: "aggressive", knowledge_level: "advanced", credit_cards: "VPBank Visa Signature,ACB JCB Gold", onboarded: "true", segment: "High Earner", age: 38, city: "HCMC" },
    { user_id: "U004", name: "Phạm Thuỳ Dung", income: 25000000, monthly_fixed: 7000000, goal: "du lịch", risk_level: "moderate", knowledge_level: "beginner", credit_cards: "TPBank EVO", onboarded: "true", segment: "Young Professional", age: 26, city: "Danang" },
    { user_id: "U005", name: "Võ Đình Khoa", income: 45000000, monthly_fixed: 15000000, goal: "nghỉ hưu sớm", risk_level: "moderate", knowledge_level: "advanced", credit_cards: "Shinhan Visa Platinum", onboarded: "true", segment: "High Earner", age: 42, city: "HCMC" },
    { user_id: "U006", name: "Đặng Minh Tú", income: 12000000, monthly_fixed: 4000000, goal: "trả nợ", risk_level: "conservative", knowledge_level: "beginner", credit_cards: "", onboarded: "true", segment: "Budget Conscious", age: 22, city: "Hanoi" },
    { user_id: "U007", name: "Hoàng Gia Bảo", income: 85000000, monthly_fixed: 25000000, goal: "đầu tư BĐS", risk_level: "aggressive", knowledge_level: "advanced", credit_cards: "Shinhan JCB Platinum,Techcombank Visa Infinite", onboarded: "true", segment: "Premium", age: 45, city: "HCMC" },
    { user_id: "U008", name: "Ngô Thanh Hà", income: 22000000, monthly_fixed: 6000000, goal: "mua xe", risk_level: "moderate", knowledge_level: "intermediate", credit_cards: "Shinhan Visa Classic", onboarded: "true", segment: "Young Professional", age: 30, city: "HCMC" },
    { user_id: "U009", name: "Bùi Quang Huy", income: 55000000, monthly_fixed: 18000000, goal: "giáo dục con", risk_level: "conservative", knowledge_level: "intermediate", credit_cards: "Shinhan Visa Gold", onboarded: "true", segment: "Family Planner", age: 36, city: "Hanoi" },
    { user_id: "U010", name: "Lý Thị Mai", income: 15000000, monthly_fixed: 5500000, goal: "tiết kiệm", risk_level: "conservative", knowledge_level: "beginner", credit_cards: "", onboarded: "false", segment: "Budget Conscious", age: 23, city: "Cantho" },
  ];

  const TRANSACTIONS = (() => {
    const cats = ["ăn uống", "di chuyển", "mua sắm", "hoá đơn", "giải trí", "giáo dục", "y tế", "tiết kiệm"];
    const data = [];
    CUSTOMERS.forEach((customer) => {
      const monthly = customer.income - customer.monthly_fixed;
      for (let month = 1; month <= 6; month++) {
        cats.forEach((category) => {
          const base = category === "ăn uống" ? 0.25 : category === "hoá đơn" ? 0.15 : category === "di chuyển" ? 0.12 : category === "mua sắm" ? 0.18 : category === "tiết kiệm" ? 0.15 : 0.05;
          const amount = Math.round(monthly * base * (0.7 + Math.random() * 0.6));
          if (amount > 0) {
            data.push({
              user_id: customer.user_id,
              month,
              category,
              amount,
              date: `2026-${String(month).padStart(2, "0")}-${String(Math.floor(Math.random() * 28) + 1).padStart(2, "0")}`,
            });
          }
        });
      }
    });
    return data;
  })();

  const COACH_QUESTIONS = [
    { user_id: "U001", q: "Nên đầu tư quỹ mở hay gửi tiết kiệm?", topic: "investment", ts: "2026-04-10" },
    { user_id: "U001", q: "Thẻ tín dụng nào cashback cao nhất cho ăn uống?", topic: "credit_card", ts: "2026-04-12" },
    { user_id: "U002", q: "Làm sao tiết kiệm được 5 triệu/tháng?", topic: "savings", ts: "2026-04-08" },
    { user_id: "U002", q: "Quy tắc 50/30/20 là gì?", topic: "education", ts: "2026-04-11" },
    { user_id: "U003", q: "So sánh ETF VN30 với chứng chỉ quỹ trái phiếu", topic: "investment", ts: "2026-04-09" },
    { user_id: "U003", q: "Bảo hiểm nhân thọ nào tốt nhất 2026?", topic: "insurance", ts: "2026-04-14" },
    { user_id: "U004", q: "Tích dặm bay thẻ nào nhanh nhất?", topic: "loyalty", ts: "2026-04-07" },
    { user_id: "U005", q: "Cần bao nhiêu tiền để nghỉ hưu lúc 50?", topic: "retirement", ts: "2026-04-13" },
    { user_id: "U005", q: "Nên mua vàng hay gửi tiết kiệm dài hạn?", topic: "investment", ts: "2026-04-15" },
    { user_id: "U006", q: "Trả nợ tín dụng hay tiết kiệm trước?", topic: "debt", ts: "2026-04-06" },
    { user_id: "U007", q: "Thuế BĐS cho căn hộ thứ 2 tính sao?", topic: "tax", ts: "2026-04-10" },
    { user_id: "U007", q: "Lãi suất margin cho chứng khoán bao nhiêu?", topic: "investment", ts: "2026-04-14" },
    { user_id: "U008", q: "Vay mua xe lãi suất thấp nhất ở đâu?", topic: "loan", ts: "2026-04-11" },
    { user_id: "U009", q: "Bảo hiểm giáo dục nào cho con tốt?", topic: "insurance", ts: "2026-04-12" },
    { user_id: "U009", q: "Nên gửi tiết kiệm bao lâu cho quỹ giáo dục?", topic: "savings", ts: "2026-04-15" },
    { user_id: "U010", q: "Làm sao mở tài khoản tiết kiệm online?", topic: "education", ts: "2026-04-09" },
  ];

  const SEGMENTS = {
    "Young Professional": { color: "#2563eb", desc: "25-32 tuổi, thu nhập 20-40tr, mục tiêu tích lũy", products: ["Shinhan Visa Cashback", "Gửi tiết kiệm online", "Bảo hiểm tai nạn"] },
    "Budget Conscious": { color: "#059669", desc: "22-28 tuổi, thu nhập <20tr, cần hỗ trợ quản lý chi", products: ["Thẻ ghi nợ miễn phí", "Tiết kiệm vi mô", "Bảo hiểm sức khỏe cơ bản"] },
    "High Earner": { color: "#d97706", desc: "35-45 tuổi, thu nhập 45-85tr, quan tâm đầu tư", products: ["Shinhan Visa Platinum", "Quỹ đầu tư", "Bảo hiểm nhân thọ cao cấp"] },
    "Family Planner": { color: "#7c3aed", desc: "33-40 tuổi, có gia đình, ưu tiên giáo dục và bảo hiểm", products: ["Shinhan Family Card", "Bảo hiểm giáo dục", "Tiết kiệm định kỳ"] },
    "Premium": { color: "#dc2626", desc: "40+ tuổi, thu nhập >80tr, cần quản lý tài sản cao cấp", products: ["Shinhan Visa Infinite", "Ngân hàng riêng", "BĐS đầu tư", "Quỹ tín thác"] },
  };

  const SEGMENT_LABELS = {
    "Young Professional": "Chuyên viên trẻ",
    "Budget Conscious": "Tối ưu ngân sách",
    "High Earner": "Thu nhập cao",
    "Family Planner": "Kế hoạch gia đình",
    Premium: "Khách hàng ưu tiên",
  };

  const RISK_LABELS = {
    aggressive: "Mạo hiểm",
    moderate: "Trung bình",
    conservative: "Thận trọng",
  };

  const TOPIC_LABELS = {
    investment: "Đầu tư",
    credit_card: "Thẻ tín dụng",
    savings: "Tiết kiệm",
    education: "Giáo dục tài chính",
    insurance: "Bảo hiểm",
    loyalty: "Tích điểm",
    retirement: "Hưu trí",
    debt: "Quản lý nợ",
    tax: "Thuế",
    loan: "Vay vốn",
  };

  const CITY_LABELS = {
    HCMC: "TP.HCM",
    Hanoi: "Hà Nội",
    Danang: "Đà Nẵng",
    Cantho: "Cần Thơ",
  };

  const AI_INSIGHTS = [
    { type: "opportunity", icon: "💳", title: "68% khách chưa có thẻ tín dụng Shinhan", detail: "Trong nhóm Young Professional, 68% đang dùng thẻ ngân hàng khác hoặc chưa có thẻ. Gợi ý: campaign Shinhan Visa Cashback với ưu đãi hoàn tiền F&B 5%.", priority: "high", segment: "Young Professional" },
    { type: "risk", icon: "⚠️", title: "3 khách có chi tiêu vượt 90% thu nhập", detail: "U002, U006, U010 chi tiêu gần hết thu nhập hàng tháng. Rủi ro churn cao. Gợi ý: kích hoạt tính năng Budget Alert trên app.", priority: "high", segment: "Budget Conscious" },
    { type: "upsell", icon: "📈", title: "High Earner quan tâm đầu tư nhưng chưa có sản phẩm", detail: "U003, U005 hỏi nhiều về đầu tư trên Finance Coach nhưng chưa mua sản phẩm đầu tư nào của Shinhan. Gợi ý: mời tham gia Investment Webinar.", priority: "medium", segment: "High Earner" },
    { type: "engagement", icon: "🔔", title: "U010 onboarding chưa hoàn tất", detail: "Lý Thị Mai đăng ký nhưng chưa hoàn thành onboarding. Đã 7 ngày. Gợi ý: gửi Zalo nhắc với incentive (tặng voucher 50K).", priority: "medium", segment: "Budget Conscious" },
    { type: "cross_sell", icon: "🛡️", title: "Family Planner cần bảo hiểm giáo dục", detail: "U009 hỏi về bảo hiểm giáo dục 2 lần trong tuần. Chưa có sản phẩm bảo hiểm nào. Gợi ý: RM liên hệ trực tiếp, giới thiệu Shinhan Education Plan.", priority: "high", segment: "Family Planner" },
  ];

  const fmt = (n) => new Intl.NumberFormat("vi-VN").format(n);
  const fmtM = (n) => (n >= 1e9 ? `${(n / 1e9).toFixed(1)}B` : n >= 1e6 ? `${(n / 1e6).toFixed(0)}M` : fmt(n));
  const escapeHtml = (value) => String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");

  const displaySegment = (name) => SEGMENT_LABELS[name] || name;
  const displayRisk = (value) => RISK_LABELS[value] || value;
  const displayTopic = (value) => TOPIC_LABELS[value] || value;
  const displayCity = (value) => CITY_LABELS[value] || value;

  const state = {
    tab: "overview",
    selectedSegment: null,
    aiQuery: "",
    aiResponse: null,
    aiLoading: false,
  };

  const root = document.getElementById("root");

  function segmentData() {
    const groups = {};
    CUSTOMERS.forEach((customer) => {
      if (!groups[customer.segment]) groups[customer.segment] = { name: customer.segment, count: 0, totalIncome: 0 };
      groups[customer.segment].count += 1;
      groups[customer.segment].totalIncome += customer.income;
    });
    return Object.values(groups).map((item) => ({ ...item, avgIncome: Math.round(item.totalIncome / item.count) }));
  }

  function monthlySpending() {
    const months = {};
    TRANSACTIONS.forEach((txn) => {
      if (!months[txn.month]) months[txn.month] = { month: `T${txn.month}`, total: 0 };
      months[txn.month].total += txn.amount;
    });
    return Object.values(months).sort((a, b) => a.month.localeCompare(b.month));
  }

  function categoryBreakdown() {
    const categories = {};
    TRANSACTIONS.forEach((txn) => {
      if (!categories[txn.category]) categories[txn.category] = { name: txn.category, value: 0 };
      categories[txn.category].value += txn.amount;
    });
    return Object.values(categories).sort((a, b) => b.value - a.value);
  }

  function topicBreakdown() {
    const topics = {};
    COACH_QUESTIONS.forEach((question) => {
      if (!topics[question.topic]) topics[question.topic] = { name: displayTopic(question.topic), count: 0 };
      topics[question.topic].count += 1;
    });
    return Object.values(topics).sort((a, b) => b.count - a.count);
  }

  function averageMonthlySpendForSegment(segmentName) {
    const users = CUSTOMERS.filter((customer) => customer.segment === segmentName);
    if (!users.length) return 0;
    const userIds = new Set(users.map((user) => user.user_id));
    const txns = TRANSACTIONS.filter((txn) => userIds.has(txn.user_id));
    return txns.length > 0 ? Math.round(txns.reduce((sum, txn) => sum + txn.amount, 0) / users.length / 6) : 0;
  }

  function topCategoriesForSegment(segmentName) {
    const userIds = new Set(CUSTOMERS.filter((customer) => customer.segment === segmentName).map((user) => user.user_id));
    const txns = TRANSACTIONS.filter((txn) => userIds.has(txn.user_id));
    const byCategory = {};
    txns.forEach((txn) => {
      byCategory[txn.category] = (byCategory[txn.category] || 0) + txn.amount;
    });
    return Object.entries(byCategory).sort((a, b) => b[1] - a[1]).slice(0, 3).map((entry) => entry[0]);
  }

  function aiResponseFor(query) {
    const totalCustomers = CUSTOMERS.length;
    const onboarded = CUSTOMERS.filter((customer) => customer.onboarded === "true").length;
    const avgIncome = Math.round(CUSTOMERS.reduce((sum, customer) => sum + customer.income, 0) / totalCustomers);
    const questionsAsked = COACH_QUESTIONS.length;
    const noCardCustomers = CUSTOMERS.filter((customer) => !customer.credit_cards).length;

    const q = query.toLowerCase();
    if (q.includes("thẻ") || q.includes("card")) {
      return `📊 Phân tích thẻ tín dụng:\n\n• ${noCardCustomers}/${totalCustomers} khách (${Math.round(noCardCustomers / totalCustomers * 100)}%) chưa có thẻ tín dụng Shinhan\n• Nhóm Chuyên viên trẻ: 2/3 đang dùng thẻ ngân hàng khác → cơ hội chuyển đổi cao\n• Top chi tiêu theo thẻ: Ăn uống (25%), Mua sắm (18%), Di chuyển (12%)\n\n💡 Gợi ý: Triển khai chiến dịch "Switch to Shinhan" với ưu đãi hoàn tiền 10% trong tháng đầu, nhắm đến nhóm Chuyên viên trẻ có thu nhập >25tr.`;
    }
    if (q.includes("segment") || q.includes("phân khúc") || q.includes("nhóm")) {
      return `📊 Phân khúc khách hàng:\n\n• Chuyên viên trẻ: 3 KH (30%) — nhu cầu thẻ hoàn tiền + tiết kiệm\n• Tối ưu ngân sách: 3 KH (30%) — cần tiết kiệm vi mô + giáo dục tài chính\n• Thu nhập cao: 2 KH (20%) — quan tâm đầu tư + quản lý tài sản\n• Kế hoạch gia đình: 1 KH (10%) — bảo hiểm + tiết kiệm giáo dục\n• Khách hàng ưu tiên: 1 KH (10%) — ngân hàng riêng + BĐS\n\n💡 Phân khúc mang lại ROI cao nhất: nhắm đến nhóm Thu nhập cao với sản phẩm đầu tư (doanh thu/KH cao nhất).`;
    }
    if (q.includes("risk") || q.includes("rủi ro") || q.includes("churn")) {
      return `⚠️ Phân tích rủi ro rời bỏ:\n\n• 3 KH chi tiêu >90% thu nhập: U002 (Trần Thị Lan), U006 (Đặng Minh Tú), U010 (Lý Thị Mai)\n• 1 KH chưa hoàn tất onboarding sau 7 ngày: U010\n• 2 KH không đăng nhập Finance Coach >14 ngày\n\n💡 Hành động: Gửi thông báo "Cảnh báo ngân sách" cho 3 KH rủi ro. RM gọi điện cho U010.`;
    }
    return `📊 Tổng quan nhanh:\n\n• ${totalCustomers} khách hàng, ${onboarded} đã onboard\n• Thu nhập BQ: ${fmtM(avgIncome)} VND/tháng\n• ${questionsAsked} câu hỏi Finance Coach (tuần này)\n• Chủ đề nổi bật: đầu tư (4), bảo hiểm (2), tiết kiệm (2)\n• ${noCardCustomers} KH chưa có thẻ Shinhan → cơ hội bán chéo\n\n💡 Hỏi thêm: "phân tích thẻ tín dụng", "rủi ro rời bỏ", "phân khúc khách hàng"`;
  }

  function renderDonut(data) {
    const colors = data.map((item) => SEGMENTS[item.name]?.color || "#94a3b8");
    const total = data.reduce((sum, item) => sum + item.count, 0);
    let start = 0;
    const stops = data.map((item, index) => {
      const pct = (item.count / total) * 100;
      const end = start + pct;
      const stop = `${colors[index]} ${start.toFixed(2)}% ${end.toFixed(2)}%`;
      start = end;
      return stop;
    }).join(", ");
    return `<div class="donut" style="background: conic-gradient(${stops});"></div>`;
  }

  function renderBars(items, maxValue, valueKey, labelKey, colorFn, formatter) {
    return items.map((item) => {
      const value = item[valueKey];
      const pct = maxValue > 0 ? Math.max(6, (value / maxValue) * 100) : 0;
      return `
        <div class="bar-row">
          <div class="bar-row-head">
            <span class="bar-label">${item[labelKey]}</span>
            <span class="bar-value">${formatter ? formatter(value) : value}</span>
          </div>
          <div class="bar-track"><div class="bar-fill" style="width:${pct}%; background:${colorFn(item)}"></div></div>
        </div>
      `;
    }).join("");
  }

  function renderOverview() {
    const totalCustomers = CUSTOMERS.length;
    const onboarded = CUSTOMERS.filter((customer) => customer.onboarded === "true").length;
    const avgIncome = Math.round(CUSTOMERS.reduce((sum, customer) => sum + customer.income, 0) / totalCustomers);
    const totalTxVolume = TRANSACTIONS.reduce((sum, txn) => sum + txn.amount, 0);
    const questionsAsked = COACH_QUESTIONS.length;
    const noCardCustomers = CUSTOMERS.filter((customer) => !customer.credit_cards).length;
    const segments = segmentData();
    const monthly = monthlySpending();

    const maxMonth = Math.max(...monthly.map((item) => item.total), 1);
    const totalSegments = segments.reduce((sum, item) => sum + item.count, 0);

    return `
      <section class="section">
        <div class="grid-5">
          ${[
            { label: "Tổng khách hàng", value: totalCustomers, sub: `${onboarded} đã onboard`, color: "#0f4c81" },
            { label: "Thu nhập trung bình", value: fmtM(avgIncome), sub: "VND/tháng", color: "#059669" },
            { label: "Khối lượng giao dịch", value: fmtM(totalTxVolume), sub: "6 tháng", color: "#d97706" },
            { label: "Câu hỏi Coach", value: questionsAsked, sub: "tuần này", color: "#7c3aed" },
            { label: "Chưa có thẻ Shinhan", value: noCardCustomers, sub: `${Math.round(noCardCustomers / totalCustomers * 100)}% cơ hội`, color: "#dc2626" },
          ].map((kpi) => `
            <div class="kpi">
              <div class="kpi-label">${kpi.label}</div>
              <div class="kpi-value" style="color:${kpi.color}">${kpi.value}</div>
              <div class="kpi-sub">${kpi.sub}</div>
            </div>
          `).join("")}
        </div>

        <div class="grid-2" style="margin-top:16px; margin-bottom:20px;">
          <div class="card card-pad">
            <div class="card-title">Khối lượng giao dịch theo tháng</div>
            <div class="chart-shell">
              ${monthly.map((item) => {
                const pct = (item.total / maxMonth) * 100;
                return `
                  <div class="month-bar">
                    <div class="month-bar-track"><div class="month-bar-fill" style="height:${pct}%;"></div></div>
                    <div class="month-value">${fmtM(item.total)}</div>
                    <div class="month-label">${item.month}</div>
                  </div>
                `;
              }).join("")}
            </div>
          </div>

          <div class="card card-pad">
            <div class="card-title">Phân khúc khách hàng</div>
            <div class="donut-wrap">${renderDonut(segments)}</div>
            <div class="legend">
              ${segments.map((item) => `
                <div class="legend-item">
                  <span class="legend-swatch" style="background:${SEGMENTS[item.name]?.color || "#94a3b8"}"></span>
                  <span>${displaySegment(item.name)} (${item.count})</span>
                </div>
              `).join("")}
            </div>
          </div>
        </div>

        <div class="card card-pad insight-accent">
          <div class="card-title" style="color:#92400e;">Insight AI - Cần hành động</div>
          <div style="display:flex; flex-direction:column; gap:8px;">
            ${AI_INSIGHTS.filter((insight) => insight.priority === "high").map((insight) => `
              <div class="insight-item">
                <span style="font-size:20px">${insight.icon}</span>
                <div>
                  <div style="font-size:12px; font-weight:700;">${insight.title}</div>
                  <div class="muted" style="font-size:11px; margin-top:2px;">${insight.detail.substring(0, 120)}...</div>
                </div>
                <div class="pill" style="margin-left:auto; align-self:flex-start; white-space:nowrap; background:#fef3c7; color:#92400e;">${displaySegment(insight.segment)}</div>
              </div>
            `).join("")}
          </div>
        </div>
      </section>
    `;
  }

  function renderSegments() {
    const customers = state.selectedSegment ? CUSTOMERS.filter((customer) => customer.segment === state.selectedSegment) : CUSTOMERS;
    return `
      <section class="section">
        <div class="grid-fit-220" style="margin-bottom:20px;">
          ${Object.entries(SEGMENTS).map(([name, segment]) => {
            const count = CUSTOMERS.filter((customer) => customer.segment === name).length;
            const isSelected = state.selectedSegment === name;
            return `
              <div class="seg-card ${isSelected ? "selected" : ""}" data-segment-card="${name}" style="border-color:${isSelected ? segment.color : "var(--line)"}; background:${isSelected ? `${segment.color}10` : "rgba(255,255,255,0.9)"};">
                <div class="seg-title-row">
                  <div class="seg-dot" style="background:${segment.color}"></div>
                    <div style="font-size:14px; font-weight:700;">${displaySegment(name)}</div>
                </div>
                  <div style="font-size:11px; color:var(--muted); margin-top:6px;">${segment.desc}</div>
                <div class="seg-count" style="color:${segment.color}">${count}</div>
                  <div class="seg-meta">khách hàng</div>
                <div class="subtle-divider">
                    <div style="font-size:10px; font-weight:700; color:var(--muted); margin-bottom:4px;">Sản phẩm đề xuất:</div>
                  ${segment.products.map((product) => `<span class="pill" style="display:inline-block; margin-right:4px; margin-bottom:4px; background:${segment.color}15; color:${segment.color};">${product}</span>`).join("")}
                </div>
              </div>
            `;
          }).join("")}
        </div>

        <div class="card card-pad">
          <div class="card-title">${state.selectedSegment ? `${displaySegment(state.selectedSegment)} khách hàng` : "Tất cả khách hàng"} (${customers.length})</div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>${["Tên", "Tuổi", "Thành phố", "Thu nhập", "Mục tiêu", "Rủi ro", "Thẻ", "Phân khúc"].map((header) => `<th>${header}</th>`).join("")}</tr>
              </thead>
              <tbody>
                ${customers.map((customer) => `
                  <tr>
                    <td style="font-weight:700;">${customer.name}</td>
                    <td>${customer.age}</td>
                    <td>${displayCity(customer.city)}</td>
                    <td>${fmtM(customer.income)}</td>
                    <td>${customer.goal}</td>
                    <td><span class="pill" style="background:${customer.risk_level === "aggressive" ? "#fef2f2" : customer.risk_level === "moderate" ? "#fffbeb" : "#f0fdf4"}; color:${customer.risk_level === "aggressive" ? "#dc2626" : customer.risk_level === "moderate" ? "#d97706" : "#059669"};">${displayRisk(customer.risk_level)}</span></td>
                    <td style="font-size:10px;">${customer.credit_cards || "—"}</td>
                    <td><span class="pill" style="background:${SEGMENTS[customer.segment]?.color}15; color:${SEGMENTS[customer.segment]?.color};">${displaySegment(customer.segment)}</span></td>
                  </tr>
                `).join("")}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    `;
  }

  function renderTransactions() {
    const categories = categoryBreakdown();
    const maxValue = Math.max(...categories.map((item) => item.value), 1);

    return `
      <section class="section">
        <div class="grid-2" style="margin-bottom:20px;">
          <div class="card card-pad">
            <div class="card-title">Chi tiêu theo danh mục (tất cả khách hàng)</div>
            ${renderBars(categories, maxValue, "value", "name", (_, item) => "#0f4c81", (value) => fmtM(value))}
          </div>

          <div class="card card-pad">
            <div class="card-title">Chi tiêu trung bình mỗi tháng theo phân khúc</div>
            <div style="display:flex; flex-direction:column; gap:10px;">
              ${Object.keys(SEGMENTS).map((segmentName) => {
                const avgMonthly = averageMonthlySpendForSegment(segmentName);
                const pct = Math.min(100, (avgMonthly / 25000000) * 100);
                return `
                  <div>
                    <div style="display:flex; justify-content:space-between; font-size:11px; margin-bottom:2px;">
                      <span style="color:#374151; font-weight:700;">${displaySegment(segmentName)}</span>
                      <span class="muted">${fmtM(avgMonthly)}/tháng</span>
                    </div>
                    <div class="bar-track"><div class="bar-fill" style="width:${pct}%; background:${SEGMENTS[segmentName].color};"></div></div>
                  </div>
                `;
              }).join("")}
            </div>

            <div class="subtle-divider">
              <div style="font-size:11px; font-weight:700; color:#374151; margin-bottom:6px;">Danh mục ưa thích theo phân khúc:</div>
              ${Object.keys(SEGMENTS).slice(0, 3).map((segmentName) => `<div style="font-size:10px; color:#6b7280; margin-bottom:3px;"><span style="color:${SEGMENTS[segmentName].color}; font-weight:700;">${displaySegment(segmentName)}:</span> ${topCategoriesForSegment(segmentName).join(", ")}</div>`).join("")}
            </div>
          </div>
        </div>
      </section>
    `;
  }

  function renderCoach() {
    const topics = topicBreakdown();
    const maxTopics = Math.max(...topics.map((item) => item.count), 1);
    const recent = COACH_QUESTIONS.slice().sort((a, b) => b.ts.localeCompare(a.ts));

    return `
      <section class="section">
        <div class="grid-2" style="margin-bottom:20px;">
          <div class="card card-pad">
            <div class="card-title">Chủ đề câu hỏi (tuần này)</div>
            ${renderBars(topics, maxTopics, "count", "name", () => "#0f4c81")}
          </div>

          <div class="card card-pad">
            <div class="card-title">Câu hỏi gần đây của khách hàng</div>
            <div style="display:flex; flex-direction:column; gap:6px; max-height:260px; overflow-y:auto;">
              ${recent.map((question) => {
                const customer = CUSTOMERS.find((item) => item.user_id === question.user_id);
                return `
                  <div style="padding:8px; border-radius:10px; background:#f9fafb; border:1px solid #f3f4f6;">
                    <div style="display:flex; justify-content:space-between; margin-bottom:4px;">
                      <span style="font-size:11px; font-weight:700; color:#0f4c81;">${customer?.name}</span>
                      <span style="font-size:10px; color:#9ca3af;">${question.ts}</span>
                    </div>
                    <div style="font-size:12px; color:#374151;">${question.q}</div>
                    <div style="display:flex; gap:4px; margin-top:4px; flex-wrap:wrap;">
                      <span class="pill" style="background:#eff6ff; color:#2563eb;">${displayTopic(question.topic)}</span>
                      <span class="pill" style="background:${SEGMENTS[customer?.segment]?.color}15; color:${SEGMENTS[customer?.segment]?.color};">${displaySegment(customer?.segment)}</span>
                    </div>
                  </div>
                `;
              }).join("")}
            </div>
          </div>
        </div>

        <div class="card card-pad">
          <div class="card-title">Ý định khách hàng → cơ hội sản phẩm</div>
          <div class="grid-fit-200">
            ${[
              { intent: "Investment questions", count: 4, product: "Shinhan Investment Fund", color: "#2563eb" },
              { intent: "Insurance interest", count: 2, product: "Shinhan Life Insurance", color: "#059669" },
              { intent: "Savings optimization", count: 2, product: "Smart Savings Account", color: "#d97706" },
              { intent: "Credit card inquiries", count: 1, product: "Shinhan Visa Cashback", color: "#dc2626" },
              { intent: "Loan interest", count: 1, product: "Shinhan Auto Loan", color: "#7c3aed" },
              { intent: "Debt management", count: 1, product: "Balance Transfer Card", color: "#0891b2" },
            ].map((item) => `
              <div style="padding:12px; border-radius:14px; border:1px solid ${item.color}20; background:${item.color}05;">
                <div style="font-size:11px; font-weight:700; color:${item.color};">${item.intent}</div>
                <div style="font-size:20px; font-weight:700; color:${item.color}; margin:4px 0;">${item.count}</div>
                <div style="font-size:10px; color:#6b7280;">khách trong tuần này</div>
                <div style="margin-top:6px; font-size:10px; padding:3px 6px; border-radius:6px; background:${item.color}15; color:${item.color}; display:inline-block; font-weight:700;">→ ${item.product}</div>
              </div>
            `).join("")}
          </div>
        </div>
      </section>
    `;
  }

  function renderAI() {
    return `
      <section class="section">
        <div class="chat-box" style="margin-bottom:20px;">
          <div style="font-size:13px; font-weight:700; color:#0f4c81; margin-bottom:10px;">Trí tuệ khách hàng từ AI</div>
          <div class="chat-controls">
            <input id="ai-query" value="${escapeHtml(state.aiQuery)}" placeholder="Hỏi AI: phân tích thẻ tín dụng, rủi ro rời bỏ, phân khúc khách hàng..." class="chat-input" />
            <button id="ai-submit" class="chat-button" style="opacity:${state.aiLoading ? 0.7 : 1};">${state.aiLoading ? "Đang phân tích..." : "Hỏi AI"}</button>
          </div>
          ${state.aiResponse ? `<div class="response">${escapeHtml(state.aiResponse)}</div>` : ""}
        </div>

        <div class="card card-pad">
          <div class="card-title">Tất cả insight do AI tạo</div>
          <div style="display:flex; flex-direction:column; gap:10px;">
            ${AI_INSIGHTS.map((insight) => `
              <div style="display:flex; gap:12px; padding:14px; border-radius:14px; border:1px solid var(--line); background:#fff;">
                <span style="font-size:24px;">${insight.icon}</span>
                <div style="flex:1;">
                  <div style="display:flex; justify-content:space-between; align-items:start; gap:10px;">
                    <div style="font-size:13px; font-weight:700;">${insight.title}</div>
                    <div style="display:flex; gap:6px; flex-wrap:wrap; justify-content:flex-end;">
                      <span class="pill" style="background:${SEGMENTS[insight.segment]?.color}15; color:${SEGMENTS[insight.segment]?.color};">${displaySegment(insight.segment)}</span>
                      <span class="pill" style="background:${insight.priority === "high" ? "#fef2f2" : "#fffbeb"}; color:${insight.priority === "high" ? "#dc2626" : "#d97706"};">${insight.priority}</span>
                    </div>
                  </div>
                  <div class="muted" style="font-size:12px; margin-top:6px; line-height:1.6;">${insight.detail}</div>
                  <div style="margin-top:8px;"><button class="action-button">Thực hiện</button></div>
                </div>
              </div>
            `).join("")}
          </div>
        </div>
      </section>
    `;
  }

  function renderApp() {
    const totalCustomers = CUSTOMERS.length;
    const onboarded = CUSTOMERS.filter((customer) => customer.onboarded === "true").length;
    root.innerHTML = `
      <div class="app-shell">
        <div class="hero">
          <div>
            <div class="eyebrow">Ngân hàng Shinhan Việt Nam</div>
            <div class="title">Bảng phân tích Finance Coach</div>
            <div class="subtitle">Bảng điều khiển độc lập được tạo từ dữ liệu mẫu hiện tại. Bố cục đã sẵn sàng để sau này chuyển sang các file JSON trong data/ mà không cần đổi cấu trúc giao diện.</div>
          </div>
          <div class="status-chip"><span class="status-dot"></span>Đang hoạt động · ${totalCustomers} khách hàng</div>
        </div>

        <div class="tabs" role="tablist">
          ${[
            ["overview", "Tổng quan"],
            ["segments", "Phân khúc"],
            ["transactions", "Giao dịch"],
            ["coach", "Phân tích Coach"],
            ["ai", "Insight AI"],
          ].map(([id, label]) => `<button class="tab-button ${state.tab === id ? "active" : ""}" data-tab="${id}">${label}</button>`).join("")}
        </div>

        <div class="content">
          ${state.tab === "overview" ? renderOverview() : ""}
          ${state.tab === "segments" ? renderSegments() : ""}
          ${state.tab === "transactions" ? renderTransactions() : ""}
          ${state.tab === "coach" ? renderCoach() : ""}
          ${state.tab === "ai" ? renderAI() : ""}
        </div>
      </div>
    `;

    document.querySelectorAll(".tab-button").forEach((button) => {
      button.addEventListener("click", () => {
        state.tab = button.dataset.tab;
        renderApp();
      });
    });

    document.querySelectorAll("[data-segment-card]").forEach((card) => {
      card.addEventListener("click", () => {
        const segmentName = card.dataset.segmentCard;
        state.selectedSegment = state.selectedSegment === segmentName ? null : segmentName;
        renderApp();
      });
    });

    const aiQuery = document.getElementById("ai-query");
    const aiSubmit = document.getElementById("ai-submit");
    if (aiQuery) {
      aiQuery.addEventListener("input", (event) => {
        state.aiQuery = event.target.value;
      });
      aiQuery.addEventListener("keydown", (event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          submitAIQuery();
        }
      });
    }
    if (aiSubmit) {
      aiSubmit.addEventListener("click", submitAIQuery);
    }
  }

  function submitAIQuery() {
    if (!state.aiQuery.trim() || state.aiLoading) return;
    state.aiLoading = true;
    state.aiResponse = null;
    renderApp();

    window.setTimeout(() => {
      state.aiResponse = aiResponseFor(state.aiQuery);
      state.aiLoading = false;
      renderApp();
    }, 900);
  }

  document.addEventListener("DOMContentLoaded", () => {
    renderApp();
  });
})();
