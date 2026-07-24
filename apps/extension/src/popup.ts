import "./style.css";

const apiBase = "http://localhost:8080";

type ScrapeResponse = {
  title: string;
  url: string;
  scrapedText: string;
};

const button = document.querySelector<HTMLButtonElement>("#capture");
const status = document.querySelector<HTMLParagraphElement>("#status");
const company = document.querySelector<HTMLInputElement>("#company");
const caseId = document.querySelector<HTMLInputElement>("#caseId");
const userId = document.querySelector<HTMLInputElement>("#userId");

button?.addEventListener("click", async () => {
  if (!status || !button || !company || !caseId || !userId) return;
  button.disabled = true;
  status.textContent = "Capturing current page";

  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!tab.id || !tab.windowId) throw new Error("No active tab");

    const scrape = (await chrome.tabs.sendMessage(tab.id, { type: "DOCSNAP_SCRAPE" })) as ScrapeResponse;
    const screenshotDataUrl = await chrome.tabs.captureVisibleTab(tab.windowId, { format: "png" });

    status.textContent = "Submitting evidence";

    const response = await fetch(`${apiBase}/api/captures`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
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
