const COSTON2_CHAIN_ID = "0x72";
type Eip1193Provider = {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
};
function getProvider(): Eip1193Provider {
  const eth = (
    window as unknown as {
      ethereum?: Eip1193Provider;
    }
  ).ethereum;
  if (!eth)
    throw new Error(
      "No wallet found — install MetaMask or another injected wallet.",
    );
  return eth;
}
async function ensureCoston2(eth: Eip1193Provider): Promise<void> {
  const chainId = await eth.request({ method: "eth_chainId" });
  if (chainId === COSTON2_CHAIN_ID) return;
  try {
    await eth.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: COSTON2_CHAIN_ID }],
    });
  } catch (err) {
    if (
      err &&
      typeof err === "object" &&
      "code" in err &&
      (
        err as {
          code: number;
        }
      ).code === 4902
    ) {
      await eth.request({
        method: "wallet_addEthereumChain",
        params: [
          {
            chainId: COSTON2_CHAIN_ID,
            chainName: "Flare Testnet Coston2",
            nativeCurrency: {
              name: "Coston2 Flare",
              symbol: "C2FLR",
              decimals: 18,
            },
            rpcUrls: ["https://coston2-api.flare.network/ext/C/rpc"],
            blockExplorerUrls: ["https://coston2-explorer.flare.network"],
          },
        ],
      });
    } else {
      throw err;
    }
  }
}
export async function connectWallet(): Promise<string> {
  const eth = getProvider();
  const accounts = (await eth.request({
    method: "eth_requestAccounts",
  })) as string[];
  if (!accounts?.[0]) throw new Error("No account returned by wallet");
  await ensureCoston2(eth);
  return accounts[0];
}
export async function sendAnchorTx(
  from: string,
  to: string,
  data: string,
): Promise<string> {
  const eth = getProvider();
  const txHash = (await eth.request({
    method: "eth_sendTransaction",
    params: [{ from, to, data }],
  })) as string;
  return txHash;
}
