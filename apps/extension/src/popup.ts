import "./style.css";
import {
  DEFAULT_COMPANY_PLACEHOLDER,
  WEB_APP_ORIGIN,
  addRecentInvestigation,
  capture,
  freshCaseId,
  getConnectionSettings,
  getOrCreateUserId,
  getRecentInvestigations,
  guessCompany,
  investigate,
  type RecentInvestigation,
} from "./api";
type ScrapeResponse = {
  title: string;
  url: string;
  scrapedText: string;
};
const status = document.querySelector<HTMLParagraphElement>("#status");
const company = document.querySelector<HTMLInputElement>("#company");
const claimText = document.querySelector<HTMLTextAreaElement>("#claimText");
const verifyClaimBtn =
  document.querySelector<HTMLButtonElement>("#verifyClaim");
const analyzePageBtn =
  document.querySelector<HTMLButtonElement>("#analyzePage");
const captureOnlyBtn =
  document.querySelector<HTMLButtonElement>("#captureOnly");
const apiUrl = document.querySelector<HTMLInputElement>("#apiUrl");
const apiKey = document.querySelector<HTMLInputElement>("#apiKey");
const recentEl = document.querySelector<HTMLElement>("#recent");
let currentUserId = "";
async function init() {
  const settings = await getConnectionSettings();
  if (apiUrl) apiUrl.value = settings.apiUrl;
  if (apiKey) apiKey.value = settings.apiKey;
  currentUserId = await getOrCreateUserId();
  if (company && company.value === DEFAULT_COMPANY_PLACEHOLDER) {
    const [tab] = await chrome.tabs.query({
      active: true,
      currentWindow: true,
    });
    const guess = tab.url ? guessCompany(tab.url) : null;
    if (guess) company.value = guess;
  }
  await renderRecent();
}
void init();
async function renderRecent() {
  if (!recentEl) return;
  const recent = await getRecentInvestigations();
  if (recent.length === 0) {
    recentEl.innerHTML = "";
    return;
  }
  recentEl.innerHTML =
    `<div class="recent-heading">Recent investigations</div>` +
    recent
      .map(
        (r: RecentInvestigation) =>
          `<a class="recent-item" href="${WEB_APP_ORIGIN}/investigations/${r.id}" target="_blank" rel="noopener">${escapeHtml(truncate(r.text, 60))}</a>`,
      )
      .join("");
}
function escapeHtml(value: string): string {
  const div = document.createElement("div");
  div.textContent = value;
  return div.innerHTML;
}
function truncate(value: string, max: number): string {
  return value.length > max ? value.slice(0, max) + "…" : value;
}
async function withStatus(
  action: () => Promise<void>,
  buttons: (HTMLButtonElement | null)[],
) {
  buttons.forEach((b) => b && (b.disabled = true));
  try {
    await action();
  } catch (error) {
    if (status)
      status.textContent =
        error instanceof Error ? error.message : "Something went wrong";
  } finally {
    buttons.forEach((b) => b && (b.disabled = false));
  }
}
async function activeTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab.id || !tab.windowId) throw new Error("No active tab");
  if (!tab.url || !/^https?:\/\//.test(tab.url)) {
    throw new Error("Open a normal webpage first");
  }
  return tab as chrome.tabs.Tab & {
    id: number;
    windowId: number;
  };
}
verifyClaimBtn?.addEventListener("click", () =>
  withStatus(async () => {
    if (!status || !claimText || !apiUrl || !apiKey || !company) return;
    const text = claimText.value.trim();
    if (!text) throw new Error("Type or paste a claim first");
    const settings = {
      apiUrl: apiUrl.value.trim() || (await getConnectionSettings()).apiUrl,
      apiKey: apiKey.value,
    };
    await chrome.storage.sync.set({
      apiUrl: settings.apiUrl,
      apiKey: settings.apiKey,
    });
    const tab = await activeTab();
    status.textContent = "Analyzing claim…";
    const evidence = await capture(settings.apiUrl, settings.apiKey, {
      url: tab.url!,
      title: tab.title || text,
      company: company.value,
      caseId: freshCaseId(),
      userId: currentUserId,
      scrapedText: text,
    });
    const claimId = evidence.claims?.[0]?.id;
    if (!claimId) throw new Error("No claim extracted from that text");
    status.textContent = "Searching evidence…";
    const investigated = await investigate(
      settings.apiUrl,
      settings.apiKey,
      claimId,
    );
    await addRecentInvestigation({
      id: claimId,
      text: investigated.text,
      status: investigated.investigationStatus,
      at: new Date().toISOString(),
    });
    await renderRecent();
    status.textContent = `${investigated.investigationStatus.replace(/_/g, " ")} — ${Math.round((investigated.investigationConfidence ?? 0) * 100)}% confidence`;
    chrome.tabs.create({ url: `${WEB_APP_ORIGIN}/investigations/${claimId}` });
  }, [verifyClaimBtn, analyzePageBtn, captureOnlyBtn]),
);
analyzePageBtn?.addEventListener("click", () =>
  withStatus(async () => {
    if (!status || !apiUrl || !apiKey || !company) return;
    const settings = {
      apiUrl: apiUrl.value.trim() || (await getConnectionSettings()).apiUrl,
      apiKey: apiKey.value,
    };
    await chrome.storage.sync.set({
      apiUrl: settings.apiUrl,
      apiKey: settings.apiKey,
    });
    const tab = await activeTab();
    status.textContent = "Reading page…";
    const scrape = await scrapeCurrentPage(tab.id);
    const screenshotDataUrl = await chrome.tabs.captureVisibleTab(
      tab.windowId,
      { format: "png" },
    );
    status.textContent = "Extracting claims…";
    const evidence = await capture(settings.apiUrl, settings.apiKey, {
      url: scrape.url,
      title: scrape.title,
      company: company.value,
      caseId: freshCaseId(),
      userId: currentUserId,
      screenshotDataUrl,
      scrapedText: scrape.scrapedText,
    });
    const claimId = evidence.claims?.[0]?.id;
    if (!claimId) {
      status.textContent = "Captured — no claims to investigate";
      return;
    }
    status.textContent = "Searching evidence…";
    const investigated = await investigate(
      settings.apiUrl,
      settings.apiKey,
      claimId,
    );
    await addRecentInvestigation({
      id: claimId,
      text: investigated.text,
      status: investigated.investigationStatus,
      at: new Date().toISOString(),
    });
    await renderRecent();
    status.textContent = `Analyzed — ${evidence.claims.length} claim(s) found`;
    chrome.tabs.create({ url: `${WEB_APP_ORIGIN}/investigations/${claimId}` });
  }, [verifyClaimBtn, analyzePageBtn, captureOnlyBtn]),
);
captureOnlyBtn?.addEventListener("click", () =>
  withStatus(async () => {
    if (!status || !apiUrl || !apiKey || !company) return;
    const settings = {
      apiUrl: apiUrl.value.trim() || (await getConnectionSettings()).apiUrl,
      apiKey: apiKey.value,
    };
    await chrome.storage.sync.set({
      apiUrl: settings.apiUrl,
      apiKey: settings.apiKey,
    });
    const tab = await activeTab();
    status.textContent = "Capturing current page";
    const scrape = await scrapeCurrentPage(tab.id);
    const screenshotDataUrl = await chrome.tabs.captureVisibleTab(
      tab.windowId,
      { format: "png" },
    );
    status.textContent = "Submitting evidence";
    const evidence = await capture(settings.apiUrl, settings.apiKey, {
      url: scrape.url,
      title: scrape.title,
      company: company.value,
      caseId: freshCaseId(),
      userId: currentUserId,
      screenshotDataUrl,
      scrapedText: scrape.scrapedText,
    });
    status.textContent = `Certified ${evidence.claims?.length ?? 0} claims`;
  }, [verifyClaimBtn, analyzePageBtn, captureOnlyBtn]),
);
async function scrapeCurrentPage(tabId: number): Promise<ScrapeResponse> {
  const [result] = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => {
      const skip = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "SVG", "CANVAS"]);
      const lines: string[] = [];
      const seen = new Set<string>();
      function walk(node: Node) {
        if (node.nodeType === Node.TEXT_NODE) {
          const text = node.textContent?.replace(/\s+/g, " ").trim() ?? "";
          if (text && !seen.has(text)) {
            seen.add(text);
            lines.push(text);
          }
          return;
        }
        if (
          node.nodeType !== Node.ELEMENT_NODE ||
          skip.has((node as Element).tagName)
        )
          return;
        node.childNodes.forEach(walk);
      }
      walk(document.body);
      return {
        title: document.title,
        url: location.href,
        scrapedText: lines.join(" ").slice(0, 16000),
      };
    },
  });
  if (!result?.result) {
    throw new Error("Could not read the current page");
  }
  return result.result as ScrapeResponse;
}
