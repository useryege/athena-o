import 'dotenv/config';

const API_KEY = process.env.API_KEY || "";
const RPC_URL = process.env.RPC_URL || "https://bsc-dataseed.binance.org/";
const PRIVATE_KEY = process.env.PRIVATE_KEY || "";
const MULTI_SIG_ADDRESS = process.env.MULTI_SIG_ADDRESS || "";

export { API_KEY, RPC_URL, PRIVATE_KEY, MULTI_SIG_ADDRESS };