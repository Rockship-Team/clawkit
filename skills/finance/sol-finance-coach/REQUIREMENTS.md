# TỔNG QUAN

**Bot là gì:** Một trợ lý tài chính cá nhân 24/7 chạy trên nền OpenClaw. Bot không truy cập tài khoản ngân hàng — bot là một **financial knowledge coach** giúp người dùng hiểu về đầu tư, tiết kiệm thông minh, tận dụng ưu đãi thẻ tín dụng, và tối ưu chi tiêu hàng ngày.

---

## Module 1: `financial-knowledge-base`
**Mô tả:** RAG (Retrieval-Augmented Generation) trên kho kiến thức tài chính cá nhân Việt Nam.

**Chi tiết:**
- Chuẩn bị knowledge base dạng markdown/JSON chứa: kiến thức đầu tư cơ bản (cổ phiếu, trái phiếu, chứng chỉ quỹ, vàng, tiết kiệm gửi ngân hàng), thuật ngữ tài chính phổ thông (lãi suất kép, lạm phát, diversification, P/E, ROI), quy tắc tài chính cá nhân (50/30/20, quỹ dự phòng 6 tháng, trả nợ lãi cao trước), so sánh các kênh đầu tư phù hợp theo mức thu nhập và rủi ro
- Bot sử dụng RAG: khi user hỏi → tìm chunk liên quan trong knowledge base → LLM tổng hợp câu trả lời tự nhiên bằng tiếng Việt
- Tone: thân thiện, dễ hiểu, không dùng jargon. Như một người bạn giỏi tài chính, không phải chuyên gia ngân hàng.

**Input:** Câu hỏi tiếng Việt tự nhiên
**Output:** Câu trả lời có nguồn, dễ hiểu, có ví dụ thực tế

**Ví dụ tương tác:**
```
User: "Mình có 50 triệu muốn đầu tư, nên bỏ vào đâu?"
Bot: "Với 50 triệu, bạn có vài lựa chọn phù hợp:

1. Nếu chưa có quỹ dự phòng: gửi tiết kiệm online 3-6 tháng
   (lãi ~5-6%/năm, rút được khi cần)
2. Nếu đã có dự phòng và chấp nhận rủi ro thấp:
   chứng chỉ quỹ trái phiếu (TCBF, VCBF-FI) — lãi ~7-8%/năm
3. Nếu chấp nhận rủi ro trung bình, đầu tư dài hạn 3-5 năm:
   chứng chỉ quỹ cổ phiếu (VFMVN30) — tiềm năng 12-15%/năm

Quy tắc vàng: không bỏ toàn bộ vào 1 kênh. Bạn muốn mình phân
tích kỹ hơn cho trường hợp cụ thể của bạn không?"
```

**Effort:** 4-5 giờ (chuẩn bị content + setup RAG + test)

---

## Module 2: `savings-tips-engine`
**Mô tả:** Gợi ý mẹo tiết kiệm thực tế trong cuộc sống hàng ngày tại Việt Nam.

**Chi tiết:**
- Database 200+ mẹo tiết kiệm, phân loại theo: ăn uống (nấu ăn vs. đặt app, meal prep, mua sỉ), di chuyển (xe buýt/metro vs. Grab, carpooling), mua sắm (flash sale timing, cashback apps, so sánh giá), hoá đơn (tiết kiệm điện nước, gói cước phù hợp), giải trí (free events, subscription sharing)
- Bot gợi ý dựa trên context: nếu user nói "tháng này tiêu nhiều ăn uống" → gợi ý mẹo F&B cụ thể
- Daily tips: mỗi sáng gửi 1 mẹo tiết kiệm ngắn gọn (cron job OpenClaw)
- Seasonal: mẹo tiết kiệm theo mùa (Tết, back to school, Black Friday)

**Ví dụ:**
```
Bot (daily tip 8h sáng): "💡 Mẹo hôm nay: Đặt Grab/Be sau
21h thường rẻ hơn 20-30% nhờ ít người đặt. Nếu không gấp,
đợi tối muộn để tiết kiệm tiền di chuyển!"
```

**Effort:** 3-4 giờ (content creation + skill logic + cron setup)

---

## Module 3: `user-profile-memory`
**Mô tả:** Nhớ thông tin cá nhân người dùng xuyên suốt các phiên chat.

**Chi tiết:**
- Khi user chia sẻ thông tin, bot lưu vào OpenClaw memory: thu nhập ước tính, tình trạng tài chính (có nợ không, có quỹ dự phòng không), mục tiêu (mua nhà, mua xe, du lịch, nghỉ hưu sớm), sở thích chi tiêu, thẻ tín dụng đang dùng (ngân hàng nào, loại gì), mức độ hiểu biết tài chính (beginner/intermediate/advanced)
- Bot sử dụng profile này để cá nhân hoá MỌI câu trả lời sau đó
- User có thể yêu cầu: "xem profile của tôi", "quên thông tin của tôi"

**Ví dụ:**
```
User: "Lương mình 25 triệu, đang trả góp xe 5 triệu/tháng"
Bot: "Đã ghi nhận! Với thu nhập 25tr và trả góp 5tr, bạn có
khoảng 20tr cho chi tiêu và tiết kiệm. Theo quy tắc 50/30/20:
- Nhu cầu thiết yếu: ~10tr (50%)
- Muốn có: ~6tr (30%)
- Tiết kiệm/đầu tư: ~4tr (20%)
Mình sẽ dựa trên thông tin này để gợi ý phù hợp hơn cho bạn."
```

**Effort:** 2-3 giờ (OpenClaw memory API + profile schema)

---

## Module 4: `onboarding-flow`
**Mô tả:** Flow chào mừng khi user lần đầu chat với bot.

**Chi tiết:**
- Giới thiệu bot là ai, làm được gì
- Hỏi 5 câu nhanh để build profile: "Thu nhập hàng tháng khoảng bao nhiêu?", "Bạn đang có mục tiêu tài chính gì?", "Bạn đã từng đầu tư chưa?", "Bạn đang dùng thẻ tín dụng ngân hàng nào?", "Bạn muốn mình gợi ý mẹo tiết kiệm hàng ngày không?"
- Lưu vào user-profile-memory
- Gửi "welcome gift": 1 bài tổng quan tài chính cá nhân dựa trên profile vừa tạo

**Effort:** 2-3 giờ

---

## Module 5: `credit-card-optimizer`
**Mô tả:** So sánh và gợi ý thẻ tín dụng phù hợp. Tối ưu cách dùng thẻ hiện tại.

**Chi tiết:**
- Database thẻ tín dụng các ngân hàng VN (cập nhật thủ công ban đầu, sau này có thể scrape): tên thẻ, phí thường niên, cashback/rewards rate, ưu đãi đặc biệt (miễn phí năm đầu, hoàn tiền F&B, tích dặm bay), điều kiện mở (thu nhập tối thiểu), lãi suất nếu trả chậm
- Gợi ý thẻ mới: dựa trên profile user (chi tiêu gì nhiều, thu nhập bao nhiêu)
- Tối ưu thẻ hiện tại: "Bạn đang dùng Visa Platinum Techcombank? Mỗi thứ 3 chi tiêu online được hoàn 5%. Đặt lịch mua sắm online vào thứ 3 nhé!"
- So sánh: "So sánh thẻ cashback Techcombank vs. VPBank vs. TPBank cho mình"

**Ví dụ:**
```
User: "Mình hay đi ăn ngoài và đặt Grab, nên mở thẻ gì?"
Bot: "Với chi tiêu chính là F&B và di chuyển, top 3 thẻ cho bạn:

🥇 VPBank Shopee Platinum: Hoàn 6% Shopee + 1% mọi chi tiêu
   Phí: miễn năm đầu. Thu nhập tối thiểu: 6tr

🥈 Techcombank Visa: Hoàn 5% chi tiêu online thứ 3
   + 1% các ngày khác. Phí: 299K/năm

🥉 TPBank EVO: Hoàn 1% mọi giao dịch, không giới hạn
   Phí: miễn phí vĩnh viễn. Thu nhập tối thiểu: 5tr

Với thu nhập 25tr của bạn, cả 3 đều mở được.
Bạn muốn mình phân tích chi tiết thẻ nào?"
```

**Effort:** 5-6 giờ (data collection + comparison logic + testing)

---

## Module 6: `loyalty-program-tracker`
**Mô tả:** Theo dõi và tối ưu các chương trình loyalty mà user tham gia.

**Chi tiết:**
- User khai báo loyalty programs đang dùng: điểm thưởng ngân hàng (SOL points, VPBank Dream), tích dặm bay (Vietnam Airlines Lotusmiles, Bamboo Club), loyalty app (Shopee Coins, GrabRewards, MoMo), membership (The Coffee House, Highland, CGV)
- Bot nhắc: "Bạn có 12,000 Lotusmiles sắp hết hạn tháng 6. Đổi vé nội địa HN-SGN được đó!"
- Gợi ý cách tích điểm nhanh: "Chi tiêu qua thẻ Techcombank Visa tích dặm gấp đôi đến hết tháng 5"
- Combo stacking: "Thanh toán GrabFood bằng thẻ VPBank Shopee → được cả Shopee Coins + cashback thẻ + GrabRewards"

**Ví dụ:**
```
User: "Mình có tích dặm Lotusmiles, làm sao tích nhanh?"
Bot: "Vài cách tích Lotusmiles nhanh:

✈️ Bay Vietnam Airlines (dĩ nhiên rồi!)
💳 Chi tiêu thẻ tín dụng đồng thương hiệu VNA
   (Techcombank/Sacombank — 25K VND = 1 dặm)
🛒 Mua sắm qua Lotusmiles eStore (2-5x dặm)
🏨 Đặt khách sạn qua đối tác (Agoda, Booking)

Mẹo: Tập trung chi tiêu lớn (mua đồ điện tử, đặt tour)
qua thẻ tích dặm vào tháng có khuyến mãi x2.
Bạn đang có bao nhiêu dặm? Mình tính xem đổi được gì."
```

**Effort:** 4-5 giờ

---

## Module 7: `deal-hunter`
**Mô tả:** Thông báo ưu đãi hot từ các ngân hàng, thẻ tín dụng, ví điện tử, app mua sắm.

**Chi tiết:**
- Scrape hoặc cập nhật thủ công ưu đãi từ: ngân hàng (Techcombank, VPBank, TPBank, ACB...), ví điện tử (MoMo, ZaloPay, VNPay, ShopeePay), app đặt đồ ăn (Grab, ShopeeFood, Baemin), thương mại điện tử (Shopee, Lazada, Tiki — sale days)
- Match ưu đãi với profile user: "Bạn dùng thẻ Techcombank + GrabRewards → combo giảm 30% Grab + hoàn 5% thẻ hôm nay"
- Push khi có deal tốt (cron job hoặc webhook): "Flash deal: GrabFood giảm 50K cho đơn từ 100K khi thanh toán MoMo. Hôm nay 11h-13h thôi!"
- User có thể set preference: "chỉ báo deal F&B và du lịch"

**Effort:** 4-5 giờ (data source setup + matching logic + notification)

---

## Module 8: `spending-analyzer`
**Mô tả:** User tự nhập chi tiêu → bot phân tích và gợi ý cải thiện.

**Chi tiết:**
- Không kết nối ngân hàng — user tự track bằng cách nhắn cho bot: "Hôm nay tiêu 150K ăn trưa, 50K Grab, 200K mua sắm Shopee"
- Bot phân loại tự động và tích luỹ theo tuần/tháng
- Cuối tuần: gửi summary: "Tuần này bạn tiêu 2.3 triệu. Top 3: Ăn uống (900K, 39%), Mua sắm (700K, 30%), Di chuyển (400K, 17%). So với tuần trước: ăn uống tăng 200K."
- Gợi ý cụ thể: "Ăn uống chiếm 39% — cao hơn trung bình. Thử meal prep Chủ nhật: nấu sẵn 5 hộp cơm trưa, tiết kiệm ~300K/tuần."
- Cuối tháng: báo cáo tổng hợp + trend

**Ví dụ nhập liệu:**
```
User: "ăn trưa 80k, grab 45k, cafe 55k"
Bot: "✅ Đã ghi: Ăn uống 80K + 55K = 135K, Di chuyển 45K.
Tổng hôm nay: 180K. Tổng tháng này: 4.2 triệu (còn 15 ngày)."
```

**Effort:** 4-5 giờ (parser + aggregation + weekly report cron)

---

## Module 9: `daily-financial-digest`
**Mô tả:** Mỗi sáng gửi 1 bản tin tài chính ngắn gọn, cá nhân hoá.

**Chi tiết:**
- Cron job OpenClaw chạy 7h30 sáng mỗi ngày
- Nội dung: mẹo tiết kiệm hôm nay (từ savings-tips-engine), 1 deal ưu đãi hot nhất hôm nay (từ deal-hunter), 1 micro-lesson tài chính (từ knowledge-base), nhắc nếu có loyalty points sắp hết hạn
- Ngắn gọn — đọc trong 30 giây
- Cá nhân hoá theo profile: beginner nhận mẹo cơ bản, advanced nhận insight đầu tư

**Ví dụ:**
```
Bot (7h30 sáng):
"☀️ Chào buổi sáng! Bản tin tài chính hôm nay:

💡 Mẹo: Đặt auto-transfer 500K vào tài khoản tiết kiệm
mỗi ngày lương. Tiền bạn không thấy = tiền bạn không tiêu.

🔥 Deal: Techcombank hoàn 10% GrabFood hôm nay (tối đa 50K).
Dùng thẻ tín dụng Techcombank khi đặt đồ ăn trưa!

📚 Kiến thức: Lãi suất kép là gì? Nếu bạn gửi 5 triệu/tháng
với lãi 7%/năm, sau 10 năm bạn có ~865 triệu (gấp 1.44 lần
số tiền gửi vào). Thời gian là bạn!

Chúc bạn một ngày tiết kiệm thông minh! 🚀"
```

**Effort:** 2-3 giờ (cron + content assembly logic)

---

## Module 10: `financial-challenge-game`
**Mô tả:** Gamification — thử thách tiết kiệm và quiz tài chính.

**Chi tiết:**
- **Thử thách tiết kiệm:** "7 ngày không đặt trà sữa" — user check-in hàng ngày, bot cổ vũ, cuối tuần tính tiền tiết kiệm được. "30 ngày tiết kiệm 100K/ngày" — tích luỹ streak, bot nhắc nếu quên. "No Spend Weekend" — không chi tiêu ngoài thiết yếu cuối tuần
- **Quiz tài chính:** 5 câu hỏi mỗi tuần (trắc nghiệm): "Lãi suất 10%/năm trả hàng tháng thì lãi suất tháng là?", "Quỹ dự phòng nên bao nhiêu tháng chi tiêu?", "ETF là gì?" Có leaderboard nếu bot phục vụ nhiều user
- **Badges:** "Tiết kiệm Newbie" (hoàn thành thử thách đầu tiên), "Sát thủ deal" (dùng 10 deal), "Tài chính 101" (hoàn thành 20 quiz)
- **Streak tracking:** bot đếm số ngày user tương tác liên tục → encourage daily habit

**Effort:** 4-5 giờ

---

## Module 11: `investment-simulator`
**Mô tả:** Tính toán và mô phỏng kịch bản đầu tư/tiết kiệm.

**Chi tiết:**
- Compound interest calculator: "Nếu gửi X triệu/tháng, lãi Y%, sau Z năm được bao nhiêu?"
- Loan calculator: "Vay 1 tỷ mua nhà, lãi 10%/năm, 20 năm → trả bao nhiêu/tháng?"
- Retirement planner: "Muốn nghỉ hưu lúc 55 tuổi với 500 triệu/năm chi tiêu → cần bao nhiêu?"
- Goal planner: "Muốn mua xe 500 triệu trong 2 năm → cần tiết kiệm bao nhiêu/tháng?"
- So sánh kịch bản: "Gửi tiết kiệm 6% vs. Mua chứng chỉ quỹ 10% — sau 5 năm khác nhau bao nhiêu?"
- Output: bảng số liệu + giải thích đơn giản

**Ví dụ:**
```
User: "Mình muốn mua nhà 3 tỷ trong 5 năm, hiện có 500 triệu"
Bot: "Để có 3 tỷ trong 5 năm, bạn cần thêm 2.5 tỷ:

📊 Kịch bản 1: Gửi tiết kiệm (6%/năm)
   → Cần tiết kiệm ~36 triệu/tháng

📊 Kịch bản 2: Đầu tư quỹ trái phiếu (8%/năm)
   → Cần tiết kiệm ~33.5 triệu/tháng

📊 Kịch bản 3: Kết hợp (50% tiết kiệm + 50% quỹ cổ phiếu, ~10%/năm)
   → Cần tiết kiệm ~31 triệu/tháng

Với thu nhập 25tr của bạn, các kịch bản này đều vượt khả năng
hiện tại. Gợi ý: kết hợp tiết kiệm 8-10tr/tháng + vay mua nhà
(trả góp 15-18tr/tháng). Muốn mình tính kịch bản vay không?"
```

**Effort:** 3-4 giờ (math functions + scenario templates)

---

## Module 12: `feedback-and-referral`
**Mô tả:** Thu thập feedback và khuyến khích chia sẻ bot.

**Chi tiết:**
- Sau 1 tuần sử dụng: hỏi "Bot giúp ích cho bạn không? Cho mình đánh giá 1-5 sao nhé!"
- Thu thập gợi ý: "Bạn muốn mình thêm tính năng gì?"
- Referral: "Chia sẻ bot với bạn bè để cùng tiết kiệm thông minh! [link]"
- NPS tracking: đếm promoters vs. detractors

**Effort:** 1-2 giờ
