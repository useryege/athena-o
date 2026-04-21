/**
 * Patch axios defaults to use https-proxy-agent when HTTPS_PROXY is set.
 * Import via --import tsx/esm before running tests.
 */
import axios from 'axios';
import { HttpsProxyAgent } from 'https-proxy-agent';

const proxyUrl = process.env.HTTPS_PROXY ?? process.env.https_proxy;
if (proxyUrl) {
  const agent = new HttpsProxyAgent(proxyUrl);
  axios.defaults.httpsAgent = agent;
}
