import streamlit as st
import requests
import pandas as pd
import json
import time
import os
import ssl
from datetime import datetime
from dotenv import load_dotenv
from requests.adapters import HTTPAdapter
from urllib3.poolmanager import PoolManager
import urllib3

load_dotenv()

# 全局兼容性补丁：很多 Windows 环境（杀毒、VPN、运营商中间件）在 OpenSSL 3 + 默认 SECLEVEL=2 下
# 会导致 UNEXPECTED_EOF_WHILE_READING 和 DECRYPTION_FAILED_OR_BAD_RECORD_MAC。
# 降低安全等级 + 更宽松的 cipher 是社区验证有效的缓解手段。
try:
    urllib3.util.ssl_.DEFAULT_CIPHERS = "DEFAULT@SECLEVEL=1"
except Exception:
    pass

st.set_page_config(page_title="Polymarket Dispute 看板", layout="wide")

# ================== 标题 + 推广 ==================
col_title, col_promo = st.columns([4, 1.2])
with col_title:
    st.title("🚨 Polymarket 正在 Dispute 的市场")

with col_promo:
    st.markdown(
        """
        <div style="text-align: right; margin-top: 20px;">
            <a href="https://x.com/poly_make_money" target="_blank" 
               style="text-decoration: none;">
                <span style="font-size: 28px; font-weight: bold; color: #1DA1F2;">
                    💰 更多赚钱策略
                </span><br>
                <span style="font-size: 24px; font-weight: bold; color: #1DA1F2;">
                    关注 @poly_make_money
                </span>
            </a>
        </div>
        """,
        unsafe_allow_html=True
    )

st.caption("使用 .env 配置 • 默认 Telegram 推送 • 标题可点击")

# ================== 配置 ==================
TELEGRAM_TOKEN = os.getenv("TELEGRAM_TOKEN")
TELEGRAM_CHAT_ID = os.getenv("TELEGRAM_CHAT_ID")
TOPIC_THREAD_ID = os.getenv("TOPIC_THREAD_ID")

if not TELEGRAM_TOKEN or not TELEGRAM_CHAT_ID:
    st.error("❌ 未在 .env 文件中找到 TELEGRAM_TOKEN 或 TELEGRAM_CHAT_ID")
    st.stop()

SEEN_FILE = "seen_disputes.json"

with st.sidebar:
    st.header("⚙️ 设置")
    refresh_interval = st.slider("自动刷新间隔 (秒)", 60, 600, 300)
    min_volume = st.number_input("最小 24h 成交量过滤 ($)", value=2000, step=5000)
    auto_refresh = st.checkbox("开启自动刷新", value=True)

# ================== 辅助函数 ==================
def get_robust_session():
    """
    返回一个“极度不复用连接”的 requests.Session。
    针对 Windows 上常见的 SSL 中间件问题（杀毒软件、VPN、DPI）做了多层防御：
    - 强制 TLS 1.2（避开 TLS 1.3 记录解密问题）
    - SECLEVEL=1（全局已打补丁，这里再显式设置）
    - OP_LEGACY_SERVER_CONNECT（帮助某些老中间件）
    - 连接池大小强制为 1（基本不复用 TCP/TLS 连接）
    - 默认带 Connection: close
    """
    session = requests.Session()
    session.headers.update({"Connection": "close"})

    class HardenedTLSAdapter(HTTPAdapter):
        def __init__(self, *args, **kwargs):
            # 强制极小连接池：每次请求尽量走全新 TCP + TLS 握手
            kwargs.setdefault("pool_connections", 1)
            kwargs.setdefault("pool_maxsize", 1)
            super().__init__(*args, **kwargs)

        def init_poolmanager(self, connections, maxsize, block=False, **pool_kwargs):
            ctx = ssl.create_default_context()
            # 1. 强制 TLS 1.2
            ctx.minimum_version = ssl.TLSVersion.TLSv1_2
            ctx.maximum_version = ssl.TLSVersion.TLSv1_2
            # 2. 降低安全等级（很多安全软件会在 level 2 搞坏记录）
            ctx.set_ciphers("DEFAULT@SECLEVEL=1")
            # 3. 允许 legacy server connect（对某些 DPI/杀毒有帮助）
            ctx.options |= ssl.OP_LEGACY_SERVER_CONNECT  # 0x4
            pool_kwargs["ssl_context"] = ctx
            # 再次确保池很小
            self.poolmanager = PoolManager(
                num_pools=1, maxsize=1, block=block, **pool_kwargs
            )

    adapter = HardenedTLSAdapter()
    session.mount("https://", adapter)
    return session


def load_seen():
    if os.path.exists(SEEN_FILE):
        try:
            with open(SEEN_FILE, "r", encoding="utf-8") as f:
                data = json.load(f)
                if isinstance(data, list):
                    return {slug: "" for slug in data}   # 兼容旧版
                return data
        except:
            return {}
    return {}

def save_seen(seen):
    with open(SEEN_FILE, "w", encoding="utf-8") as f:
        json.dump(seen, f, ensure_ascii=False)

seen_disputes = load_seen()

# ================== Telegram 发送（标题可点击）==================
def send_telegram_message(market):
    try:
        base_link = market['链接']
        referral_link = base_link + "/?r=disputebot"
        
        msg = f"""🚨 **新 Dispute 检测！**

**标题**：[{market['标题']}]({referral_link})
**24h成交**：{market['24h成交量']}
**状态**：{market['状态']}
**结束日期**：{market['结束日期']}

⏰ {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}
"""
        payload = {
            "chat_id": TELEGRAM_CHAT_ID,
            "text": msg,
            "parse_mode": "Markdown"
        }
        if TOPIC_THREAD_ID and TOPIC_THREAD_ID.strip():
            payload["message_thread_id"] = int(TOPIC_THREAD_ID)
        
        s = get_robust_session()
        s.post(
            f"https://api.telegram.org/bot{TELEGRAM_TOKEN}/sendMessage",
            json=payload,
            timeout=15,
            headers={"Connection": "close"},
        )
        
    except Exception as e:
        st.warning(f"Telegram 发送失败: {e}")

# ================== 抓取函数（已修复显示逻辑）==================
@st.cache_data(ttl=refresh_interval)
def fetch_disputed_markets():
    url = "https://gamma-api.polymarket.com/markets/keyset"
    all_markets = []
    cursor = None
    total = 0
    new_disputes = 0
    
    progress = st.progress(0, "正在拉取...")
    status = st.empty()

    def _fetch_page_with_retry(p, prm, max_attempts=5):
        """
        针对 SSL 不稳定环境的重试封装。
        专门处理：
        - DECRYPTION_FAILED_OR_BAD_RECORD_MAC
        - UNEXPECTED_EOF_WHILE_READING / SSLEOFError
        - Max retries exceeded (底层连接被中间件干掉)
        每次重试都用全新的 session + 强制 Connection: close + 极小连接池。
        """
        last_err = None
        for attempt in range(1, max_attempts + 1):
            try:
                sess = get_robust_session()
                # 显式带 Connection: close，防止任何 keep-alive 被污染的连接复用
                resp = sess.get(
                    url,
                    params=prm,
                    timeout=30,
                    headers={"Connection": "close"}
                )
                if resp.status_code == 500:
                    time.sleep(1.5)
                    last_err = Exception("HTTP 500 from server")
                    continue
                resp.raise_for_status()
                return resp.json()
            except Exception as ex:
                last_err = ex
                estr = str(ex)
                is_ssl_err = any(
                    x in estr
                    for x in [
                        "DECRYPTION_FAILED_OR_BAD_RECORD_MAC",
                        "UNEXPECTED_EOF_WHILE_READING",
                        "SSLEOFError",
                        "EOF occurred in violation of protocol",
                        "SSLError",
                        "ssl.SSLError",
                        "Bad record mac",
                        "Max retries exceeded",
                    ]
                )
                if is_ssl_err and attempt < max_attempts:
                    st.warning(f"SSL 连接不稳定 (第 {attempt}/{max_attempts} 次)，正在用全新连接重试...")
                    # 递增退避，给中间件/网络一点恢复时间
                    time.sleep(1.5 + attempt * 0.6)
                    continue
                if attempt < max_attempts:
                    time.sleep(0.8)
                    continue
        raise last_err if last_err else Exception("Unknown fetch error")

    for page in range(500):
        params = {
            "active": "true",
            "closed": "false",
            "limit": 100,
            "order": "volume_24hr",
            "ascending": "false",
            "end_date_gte": "2026-05-01",
            "volume_24hr_gte": str(min_volume)
        }
        if cursor:
            params["after_cursor"] = cursor

        try:
            data = _fetch_page_with_retry(page, params)

            markets = data.get("markets", []) if isinstance(data, dict) else data
            page_size = len(markets)
            total += page_size

            progress.progress(min(0.98, total / 20000), f"已拉取 {total:,} 条...")
            status.text(f"第 {page+1} 页 | 本页 {page_size} | 总 {total:,}")
            
            for m in markets:
                vol = float(m.get("volume24hr") or 0)
                if vol < min_volume:
                    continue
                
                statuses = m.get("umaResolutionStatuses") or []
                status_str = json.dumps(statuses).lower()
                
                if any(k in status_str for k in ["dispute", "disputed", "challenged"]):
                    market_slug = m.get("slug", "")
                    if not market_slug:
                        continue
                    
                    current_status = str(statuses)
                    
                    event_slug = None
                    events = m.get("events", [])
                    if events and len(events) > 0:
                        event_slug = events[0].get("slug")
                    if not event_slug:
                        event_slug = m.get("eventSlug") or m.get("parentEventSlug")
                    
                    link = f"https://polymarket.com/event/{event_slug}/{market_slug}" if event_slug and event_slug != market_slug else f"https://polymarket.com/event/{market_slug}"
                    
                    market_info = {
                        "标题": m.get("question") or "N/A",
                        "24h成交量": f"${int(vol):,}",
                        "状态": current_status,
                        "结束日期": m.get("endDate", ""),
                        "链接": link,
                        "slug": market_slug
                    }
                    
                    # 关键修复：所有 Dispute 都显示
                    all_markets.append(market_info)
                    
                    # 只对新增或状态变化的推送 Telegram
                    if market_slug not in seen_disputes or seen_disputes[market_slug] != current_status:
                        send_telegram_message(market_info)
                        seen_disputes[market_slug] = current_status
                        new_disputes += 1
                        save_seen(seen_disputes)
            
            cursor = data.get("next_cursor") if isinstance(data, dict) else None
            if not cursor or page_size < 90:
                progress.progress(1.0, "✅ 完成")
                break
                
            time.sleep(0.08)
            
        except Exception as e:
            st.error(f"请求错误: {e}")
            time.sleep(2)
            cursor = None
            continue
    
    st.sidebar.success(f"本次新发现 {new_disputes} 个 | 当前显示 {len(all_markets)} 个 Dispute")
    return pd.DataFrame(all_markets)

# ===================== 主界面 =====================
placeholder = st.empty()

def main_loop():
    with placeholder.container():
        df = fetch_disputed_markets()
        
        col1, col2 = st.columns([3, 1])
        with col1:
            if df.empty:
                st.info("🎉 当前没有 Dispute 市场")
            else:
                st.success(f"🔥 当前共有 **{len(df)}** 个 Dispute 市场")
        
        with col2:
            if st.button("🔄 手动刷新", type="primary"):
                st.cache_data.clear()
                st.rerun()
        
        search = st.text_input("🔍 搜索标题", "", key="search_input")
        filtered_df = df[df["标题"].str.contains(search, case=False, na=False)] if search and not df.empty else df
        
        if not filtered_df.empty:
            st.dataframe(
                filtered_df[["标题", "24h成交量", "状态", "结束日期", "链接"]],
                use_container_width=True,
                hide_index=True,
                height=700,
                column_config={"链接": st.column_config.LinkColumn("打开", display_text="🔗 查看")}
            )

main_loop()

if auto_refresh:
    time.sleep(refresh_interval)
    st.rerun()