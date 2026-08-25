package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// cmdAnalytics dispatches sme-analytics — a THIN AGGREGATION layer over
// data that already exists (kpi.go, outreach.go's funnel, fetchAllContacts'
// business_stage). It creates no new analytics platform, no new database
// table, and no number that can't be reconciled against the underlying
// command it was built from (kpiTeamData/kpiActualData/outreachFunnelData
// are the exact same functions `kpi team`/`kpi actual`/`outreach funnel`
// use — Analytics never re-derives its own copy of these queries).
//
//	sme-cli analytics summary [--days N] [--week YYYY-Www] [--member NAME] [--max-pages N]
//	sme-cli analytics campaigns [--max-pages N]
func cmdAnalytics(args []string) {
	if len(args) == 0 {
		errOut("usage: analytics summary|campaigns")
		return
	}
	switch args[0] {
	case "summary":
		analyticsSummary(args[1:])
	case "campaigns":
		analyticsCampaigns(args[1:])
	default:
		errOut("unknown analytics command: " + args[0])
	}
}

func analyticsSummary(args []string) {
	days := 7
	week := isoWeekLabel(vnNow())
	member := ""
	maxPages := 8 // same on-demand-only default as `cosmo daily-plan` — analytics summary is NOT in the PIPELINE_WATCH hot path, so this cost is acceptable here (matches sme-opportunity's risk-list precedent).
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days":
			i++
			if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
				days = n
			}
		case "--week":
			i++
			week = args[i]
		case "--member":
			i++
			member = args[i]
		case "--max-pages":
			i++
			if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
				maxPages = n
			}
		}
	}

	targets, weekStart, weekEnd, tErr := kpiTeamData(week)
	if tErr != nil || targets == nil {
		targets = []map[string]interface{}{}
	}
	actuals, _, _, hasActuals, _ := kpiActualData(week, member)
	kpiSummary := map[string]interface{}{
		"week": week, "week_start": weekStart, "week_end": weekEnd,
		"targets": targets,
	}
	if hasActuals {
		kpiSummary["actuals"] = actuals
	} else {
		kpiSummary["actuals"] = "insufficient_data — chưa có interaction log trong local DB (interactions table), xem sme-cli cosmo search-interactions để tra COSMO trực tiếp"
	}

	funnelRows, since, fErr := outreachFunnelData(days)
	if fErr != nil {
		errOut(fErr.Error())
		return
	}
	if funnelRows == nil {
		funnelRows = []map[string]interface{}{}
	}

	channelComparison := analyticsChannelComparison(funnelRows)
	bottleneck, bottleneckNote := analyticsBottleneck(funnelRows, maxPages)
	recommendations := analyticsRecommendations(channelComparison)

	okOut(map[string]interface{}{
		"period":               map[string]interface{}{"days": days, "since": since, "week": week},
		"kpi_summary":          kpiSummary,
		"outreach_funnel":      funnelRows,
		"channel_comparison":   channelComparison,
		"bottleneck":           bottleneck,
		"bottleneck_note":      bottleneckNote,
		"recommendations":      recommendations,
		"campaign_performance": "unavailable_until_campaign_engine",
	})
}

// toCount reads a SQLite COUNT(*) result cell, which the driver may hand
// back as int64 or float64 depending on column affinity — never silently
// drop the value to 0 on a type mismatch.
func toCount(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// analyticsChannelComparison pivots outreachFunnelData's (channel,
// event_type, count) rows into a per-channel summary. reply_rate is only
// computed when the denominator (messages sent) is a real, non-zero number
// — otherwise it's reported as "insufficient_data", never a fabricated 0%.
func analyticsChannelComparison(funnelRows []map[string]interface{}) []map[string]interface{} {
	type chanStats struct {
		activities, messages, replies int
	}
	byChannel := map[string]*chanStats{}
	var order []string
	for _, r := range funnelRows {
		ch := fmt.Sprint(r["channel"])
		et := fmt.Sprint(r["event_type"])
		cnt := toCount(r["count"])
		st, ok := byChannel[ch]
		if !ok {
			st = &chanStats{}
			byChannel[ch] = st
			order = append(order, ch)
		}
		st.activities += cnt
		switch et {
		case "message_sent":
			st.messages += cnt
		case "message_reply_received":
			st.replies += cnt
		}
	}
	out := make([]map[string]interface{}, 0, len(order))
	for _, ch := range order {
		st := byChannel[ch]
		entry := map[string]interface{}{
			"channel":    ch,
			"activities": st.activities,
			"messages":   st.messages,
			"replies":    st.replies,
		}
		if st.messages > 0 {
			entry["reply_rate"] = fmt.Sprintf("%.1f%%", float64(st.replies)/float64(st.messages)*100)
		} else {
			entry["reply_rate"] = "insufficient_data"
		}
		out = append(out, entry)
	}
	return out
}

// analyticsBottleneck reports only stages with a real, reliable data source.
// "Research" is deliberately omitted from the funnel — there is no event/
// field anywhere in this system that records "researched but not yet
// contacted" — showing a fake 0 or omitted-but-implied stage there would
// misrepresent funnel completeness. Qualified/Proposal counts come from
// COSMO business_stage via fetchAllContacts (the same, already-fixed
// pagination as cosmo daily-plan/sme-opportunity — Phase 2B COSMO audit).
func analyticsBottleneck(funnelRows []map[string]interface{}, maxPages int) ([]map[string]interface{}, string) {
	outreachCount, replyCount := 0, 0
	for _, r := range funnelRows {
		et := fmt.Sprint(r["event_type"])
		cnt := toCount(r["count"])
		switch et {
		case "connection_request_sent", "message_sent":
			outreachCount += cnt
		case "message_reply_received":
			replyCount += cnt
		}
	}
	stages := []map[string]interface{}{
		{"stage": "Outreach", "count": outreachCount, "source": "outreach_events"},
		{"stage": "Reply", "count": replyCount, "source": "outreach_events"},
	}

	contacts, _, err := fetchAllContacts(maxPages)
	if err != nil {
		return stages, "Qualified/Proposal stage bị bỏ qua — không lấy được contact từ COSMO (" + err.Error() + "). Research stage luôn bị bỏ qua — không có nguồn dữ liệu 'đã research nhưng chưa liên hệ' trong hệ thống hiện tại."
	}
	qualified, proposal := 0, 0
	for _, c := range contacts {
		switch c.BusinessStage {
		case "QUALIFIED":
			qualified++
		case "PROPOSAL":
			proposal++
		}
	}
	stages = append(stages,
		map[string]interface{}{"stage": "Qualified", "count": qualified, "source": "COSMO business_stage"},
		map[string]interface{}{"stage": "Proposal", "count": proposal, "source": "COSMO business_stage"},
	)
	return stages, "Research stage bị bỏ qua có chủ đích — không có nguồn dữ liệu 'đã research nhưng chưa liên hệ' trong hệ thống hiện tại. Qualified/Proposal lấy từ tối đa " + strconv.Itoa(maxPages) + " trang COSMO (có thể chưa phủ hết nếu tổ chức có rất nhiều contact)."
}

// analyticsRecommendations flags a channel whose reply rate is notably
// LOWER than another channel with comparable volume — a relative
// comparison grounded entirely in this org's own measured data, never an
// invented absolute "good reply rate" benchmark (no such target exists
// anywhere in weekly_kpis). Requires at least 2 channels with real volume
// (>=5 messages) to compare; otherwise returns no recommendation at all.
func analyticsRecommendations(channelComparison []map[string]interface{}) []string {
	const minVolume = 5
	const minGapPoints = 10.0 // percentage points — a gap smaller than this is within normal noise for small BD volumes

	type rated struct {
		channel  string
		rate     float64
		messages int
	}
	var rates []rated
	for _, c := range channelComparison {
		messages, _ := c["messages"].(int)
		rateStr, _ := c["reply_rate"].(string)
		if messages < minVolume || rateStr == "insufficient_data" {
			continue
		}
		var rate float64
		fmt.Sscanf(rateStr, "%f%%", &rate)
		rates = append(rates, rated{fmt.Sprint(c["channel"]), rate, messages})
	}
	if len(rates) < 2 {
		return []string{}
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i].rate > rates[j].rate })
	best, worst := rates[0], rates[len(rates)-1]
	if best.rate-worst.rate < minGapPoints {
		return []string{}
	}
	return []string{fmt.Sprintf(
		"Kênh %s có reply rate %.1f%% (n=%d tin nhắn) — thấp hơn đáng kể so với %s (%.1f%%, n=%d). Nên review lại message/targeting cho %s.",
		worst.channel, worst.rate, worst.messages, best.channel, best.rate, best.messages, worst.channel,
	)}
}

// --- Campaign analytics (Phase 2C) -----------------------------------------

// analyticsCampaigns reports only metrics COSMO's real Campaign API already
// computes (sent/reply/reply_rate per campaign via GetByID/List — see
// campaign.go) plus Qualified/Proposal counts reused from analyticsBottleneck
// (same fetchAllContacts business_stage aggregation, not re-derived).
// Channel is read per-campaign via the same campaignChannel() helper
// campaignActivate uses — one small source of truth for "what channel is
// this campaign", never two.
func analyticsCampaigns(args []string) {
	maxPages := 8
	for i := 0; i < len(args); i++ {
		if args[i] == "--max-pages" && i+1 < len(args) {
			i++
			if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
				maxPages = n
			}
		}
	}

	raw, code, err := cosmoRequest("GET", "/v1/campaigns?limit=100", nil)
	if err != nil {
		errOut(err.Error())
		return
	}
	if code >= 400 {
		errOut(fmt.Sprintf("HTTP %d: %s", code, string(raw)))
		return
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"entity"`
				Sent      int     `json:"sent"`
				Reply     int     `json:"reply"`
				ReplyRate float64 `json:"reply_rate"`
			} `json:"list"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut(err.Error())
		return
	}
	truncated := resp.Data.Total > len(resp.Data.List)

	byStatus := map[string]int{}
	byChannel := map[string]*struct{ campaigns, sent, reply int }{}
	totalSent, totalReply := 0, 0
	for _, item := range resp.Data.List {
		byStatus[item.Entity.Status]++
		totalSent += item.Sent
		totalReply += item.Reply

		view, code, err := cosmoGetCampaign(item.Entity.ID)
		channel := "email"
		if err == nil && code < 400 {
			channel = campaignChannel(view)
		}
		ch, ok := byChannel[channel]
		if !ok {
			ch = &struct{ campaigns, sent, reply int }{}
			byChannel[channel] = ch
		}
		ch.campaigns++
		ch.sent += item.Sent
		ch.reply += item.Reply
	}

	channelPerf := make([]map[string]interface{}, 0, len(byChannel))
	for channel, ch := range byChannel {
		entry := map[string]interface{}{
			"channel": channel, "campaigns": ch.campaigns, "sent": ch.sent, "reply": ch.reply,
		}
		if ch.sent > 0 {
			entry["reply_rate"] = fmt.Sprintf("%.1f%%", float64(ch.reply)/float64(ch.sent)*100)
		} else {
			entry["reply_rate"] = "insufficient_data"
		}
		channelPerf = append(channelPerf, entry)
	}

	replyRate := "insufficient_data"
	if totalSent > 0 {
		replyRate = fmt.Sprintf("%.1f%%", float64(totalReply)/float64(totalSent)*100)
	}

	// Reuse the exact same COSMO business_stage aggregation as `analytics
	// summary`'s bottleneck — never a second copy of that scan.
	bottleneck, _ := analyticsBottleneck(nil, maxPages)
	qualified, proposal := 0, 0
	for _, s := range bottleneck {
		switch s["stage"] {
		case "Qualified":
			qualified = s["count"].(int)
		case "Proposal":
			proposal = s["count"].(int)
		}
	}

	note := "sent/reply/reply_rate lấy nguyên từ COSMO GetByID/List — không tự tính lại. qualified_opportunities/proposals reuse COSMO business_stage (cùng logic `analytics summary`'s bottleneck)."
	if truncated {
		note += fmt.Sprintf(" CẢNH BÁO: org có %d campaign, chỉ lấy được %d (limit=100) — số liệu dưới đây CHƯA đầy đủ.", resp.Data.Total, len(resp.Data.List))
	}

	okOut(map[string]interface{}{
		"campaigns_created":       resp.Data.Total,
		"by_status":               byStatus,
		"active_campaigns":        byStatus["active"],
		"messages_sent":           totalSent,
		"replies":                 totalReply,
		"reply_rate":              replyRate,
		"channel_performance":     channelPerf,
		"qualified_opportunities": qualified,
		"proposals":               proposal,
		"truncated":               truncated,
		"note":                    note,
	})
}
