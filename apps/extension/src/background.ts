import {
  DEFAULT_COMPANY_PLACEHOLDER,
  WEB_APP_ORIGIN,
  addRecentInvestigation,
  capture,
  freshCaseId,
  getConnectionSettings,
  getOrCreateUserId,
  guessCompany,
  investigate,
} from "./api";
const VERIFY_SELECTION_ID = "docsnap-verify-selection";
const CREATE_PROOF_ID = "docsnap-create-proof";
chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: VERIFY_SELECTION_ID,
    title: "Verify with DocSnap",
    contexts: ["selection"],
  });
  chrome.contextMenus.create({
    id: CREATE_PROOF_ID,
    title: "Create Evidence Proof",
    contexts: ["page"],
  });
});
function setBadge(text: string, color = "#3c83f6") {
  chrome.action.setBadgeText({ text });
  chrome.action.setBadgeBackgroundColor({ color });
}
function clearBadgeSoon() {
  setTimeout(() => chrome.action.setBadgeText({ text: "" }), 4000);
}
// Both context-menu actions run entirely in the background service worker —
// there's no popup open to show status text in, so the badge dot is easy to
// miss entirely. A system notification is the one thing that reaches the
// user regardless of what they're looking at.
function notify(title: string, message: string) {
  chrome.notifications.create({
    type: "basic",
    iconUrl: chrome.runtime.getURL("icons/icon128.png"),
    title,
    message,
  });
}
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (!tab?.id || !tab.url) return;
  if (info.menuItemId === VERIFY_SELECTION_ID && info.selectionText) {
    setBadge("…");
    try {
      const settings = await getConnectionSettings();
      const userId = await getOrCreateUserId();
      const company = guessCompany(tab.url) || DEFAULT_COMPANY_PLACEHOLDER;
      const evidence = await capture(settings.apiUrl, settings.apiKey, {
        url: tab.url,
        title: tab.title || info.selectionText,
        company,
        caseId: freshCaseId(),
        userId,
        scrapedText: info.selectionText,
      });
      const claimId = evidence.claims?.[0]?.id;
      if (!claimId) throw new Error("no claim");
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
      setBadge("✓", "#10b981");
      notify("Verified", investigated.investigationStatus || "Investigation complete — opening result.");
      chrome.tabs.create({
        url: `${WEB_APP_ORIGIN}/investigations/${claimId}`,
      });
    } catch {
      setBadge("!", "#ef4444");
      notify("Verify failed", "Couldn't verify that selection — try again.");
    } finally {
      clearBadgeSoon();
    }
    return;
  }
  if (info.menuItemId === CREATE_PROOF_ID) {
    setBadge("…");
    try {
      const settings = await getConnectionSettings();
      const userId = await getOrCreateUserId();
      const company = guessCompany(tab.url) || DEFAULT_COMPANY_PLACEHOLDER;
      const [scrape] = await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        func: () => {
          const skip = new Set([
            "SCRIPT",
            "STYLE",
            "NOSCRIPT",
            "SVG",
            "CANVAS",
          ]);
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
      if (!scrape?.result) throw new Error("could not read page");
      const screenshotDataUrl = await chrome.tabs.captureVisibleTab(
        tab.windowId!,
        { format: "png" },
      );
      const evidence = await capture(settings.apiUrl, settings.apiKey, {
        url: scrape.result.url,
        title: scrape.result.title,
        company,
        caseId: freshCaseId(),
        userId,
        screenshotDataUrl,
        scrapedText: scrape.result.scrapedText,
      });
      setBadge("✓", "#10b981");
      notify(
        "Evidence captured",
        `${evidence.claims?.length ?? 0} claim(s) found — opening proof.`,
      );
      chrome.tabs.create({
        url: `${WEB_APP_ORIGIN}/proof/${evidence.id}`,
      });
    } catch {
      setBadge("!", "#ef4444");
      notify("Capture failed", "Couldn't capture this page — try again.");
    } finally {
      clearBadgeSoon();
    }
  }
});
