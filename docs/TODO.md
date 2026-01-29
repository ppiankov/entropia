## Proxy support (required for corporate intraweb audits)

Entropia must support proxy routing to enable safe intraweb scrutiny via a controlled boundary.
This is required for both `--llm` and `--no-llm` modes: sensitive leakage can occur via reports
and exports even without LLM usage.

- [ ] Support standard env vars: HTTP_PROXY, HTTPS_PROXY, NO_PROXY
- [ ] Add explicit CLI flags (optional): --http-proxy, --https-proxy, --no-proxy
- [ ] Ensure proxy settings propagate to all HTTP clients used by Entropia
- [ ] Add tests for proxy routing
- [ ] Document proxy usage
