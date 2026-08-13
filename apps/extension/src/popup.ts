import "./style.css";

const DEFAULT_API_URL = "http://localhost:8080";

type ScrapeResponse = {
  title: string;
  url: string;
  scrapedText: string;
};

type StoredSettings = {
  apiUrl?: string;
  apiKey?: string;
};

const button = document.querySelector<HTMLButtonElement>("#capture");
const status = document.querySelector<HTMLParagraphElement>("#status");
const company = document.querySelector<HTMLInputElement>("#company");
const caseId = document.querySelector<HTMLInputElement>("#caseId");
const userId = document.querySelector<HTMLInputElement>("#userId");
const apiUrl = document.querySelector<HTMLInputElement>("#apiUrl");
const apiKey = document.querySelector<HTMLInputElement>("#apiKey");

chrome.storage.sync.get(["apiUrl", "apiKey"]).then((stored: StoredSettings) => {
  if (apiUrl && stored.apiUrl) apiUrl.value = stored.apiUrl;
  if (apiKey && stored.apiKey) apiKey.value = stored.apiKey;
});

button?.addEventListener("click", async () => {
  if (!status || !button || !company || !caseId || !userId || !apiUrl || !apiKey) return;
  button.disabled = true;
  status.textContent = "Capturing current page";

  try {
    const base = apiUrl.value.trim() || DEFAULT_API_URL;
    await chrome.storage.sync.set({ apiUrl: base, apiKey: apiKey.value });

    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!tab.id || !tab.windowId) throw new Error("No active tab");
    if (!tab.url || !/^https?:\/\//.test(tab.url)) {
      throw new Error("Open a normal webpage before capturing");
    }

    const scrape = await scrapeCurrentPage(tab.id);
    const screenshotDataUrl = await chrome.tabs.captureVisibleTab(tab.windowId, { format: "png" });

    status.textContent = "Submitting evidence";

    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (apiKey.value) headers.Authorization = `Bearer ${apiKey.value}`;

    const response = await fetch(`${base}/api/captures`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        url: scrape.url,
        title: scrape.title,
        company: company.value,
        caseId: caseId.value,
        userId: userId.value,
        screenshotDataUrl,
        scrapedText: scrape.scrapedText,
        capturedAt: new Date().toISOString()
      })
    });

    if (!response.ok) throw new Error("Capture failed");
    const data = await response.json();
    status.textContent = `Certified ${data.claims?.length ?? 0} claims`;
  } catch (error) {
    status.textContent = error instanceof Error ? error.message : "Capture failed";
  } finally {
    button.disabled = false;
  }
});

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
        if (node.nodeType !== Node.ELEMENT_NODE || skip.has((node as Element).tagName)) return;
        node.childNodes.forEach(walk);
      }

      walk(document.body);

      return {
        title: document.title,
        url: location.href,
        scrapedText: lines.join(" ").slice(0, 16000)
      };
    }
  });

  if (!result?.result) {
    throw new Error("Could not read the current page");
  }

  return result.result as ScrapeResponse;
}
