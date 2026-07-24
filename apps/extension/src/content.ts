chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== "DOCSNAP_SCRAPE") {
    return false;
  }

  const selectors = ["script", "style", "noscript", "svg", "canvas"];
  const clone = document.body.cloneNode(true) as HTMLElement;
  for (const selector of selectors) {
    clone.querySelectorAll(selector).forEach((node) => node.remove());
  }

  const scrapedText = clone.innerText.replace(/\s+/g, " ").trim().slice(0, 16000);
  sendResponse({
    title: document.title,
    url: location.href,
    scrapedText
  });

  return true;
});

