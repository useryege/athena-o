# Etherscan Configuration

Athena runtime services centralize Etherscan access through `athena-ethereum-api`.

- `ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS`

`athena-ethereum-api` uses an Etherscan Gateway manager and requires a comma,
space, or newline separated key list plus explicit gateway gRPC addresses:

```bash
ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS='key1,key2,key3'
ATHENA_ETHEREUM_API_ETHERSCAN_GATEWAY_ADDRS='47.245.183.140:6776 47.245.166.57:6776 47.245.161.139:6776 47.245.181.189:6776 47.254.154.128:6776'
ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN='gateway-bearer-token'
```

Each ethereum-api cache refresh picks the next API key and next gateway address
in round-robin order. A failed gateway request is returned immediately; the
manager does not retry with another key or gateway.

The `athena-token-collector --data-type contract_code_source` process does not
read a standalone Etherscan API key. It calls `athena-ethereum-api` over gRPC:

```bash
ATHENA_TOKEN_ETHEREUM_API_SERVER_ADDRESS='localhost:8100'
```

`athena-server` also reads the gateway IP list, gateway bearer token, and
`ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS` for the `/etherscan-gateways` UI live
probe. The UI accepts an interval in milliseconds and a request count per API
key, then runs the same `ListNormalTransactions` success-rate probe through the
configured gateway fleet. It stores only the latest run in `athena-server`
memory and returns short API key fingerprints rather than full keys. The probe
uses `ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS` when set; otherwise it uses
`0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045`.

The key pool below is committed in the repository documentation and retained in Git history.

```json
{
  "etherscan_api_key": [
    "454GKQPGFRIYNEKESBHS2E8PVJESNN9B5I",
    "ZZKWKRXINQD41IMPZVU6YDPD4TRKNHQ68Z",
    "1UDTDRNSYQTF6XZY1WN7DIMBYTAN9ZSTUJ",
    "HTHBUE6Y6T14CZRCUW5BAGIWJEHATYZ33W",
    "9EBNJTR26SFQ65AWN8Y7VRB1EY1Y5MDC81",
    "CMTF945AJNQ1AA81URCFXFT4E2KDD6JNBZ",
    "AEZY7P4MDZCGHP7CYHG2M8GPWVS6MG6I5H",
    "7E53PRKPA98VUN6S4YY346XS24D3HJC5V7",
    "RD6FY5ZGG9116UKR9ZDP5BWJRGDIXW5IBZ",
    "BMPWCNDB5CR61X1UX9BBXXW2UGRB6Q73RN",
    "TVNQ7ISZNH8V7X888T2AGEK9EYBF9U68SN",
    "JHJ9CTCYZRDDP6Q2TDRBFPSJCIMP4BK315",
    "88JT7J32XWT47964HMHI9BSCP3HMEQP7CX",
    "RB6UES3KKPAQ1Z87AAGXWQ5VMCCRXQ2ZT3",
    "F4NXG95NHH2MDYXG5SRWIQV6D9VG5YYYJ5",
    "WKUUQNWCM1AF2Q3MN59Y8GCWADXRAMT2HQ",
    "WNV5A2TJ8R5XTUCPFMXNKWP7W441Q4G2X5",
    "KNGNJZG3MHCYS9XGZM1DGH1NU1Y56I98AB",
    "HC9TXUJHHCR6CRVYZ3E4A6B6RSW2V56SP5",
    "ANDI26XPDF1KGPKV4D27JFZVC5CF7SD5S8",
    "84FHCZQFXCAPM4JWB547BAMEEZW6P6FGWK",
    "I9H4RSK9UWT7MZ8SV18CZE1YX97JETVHK5",
    "YYFVC7EBH5SVFQENYBHE6YZN7KTMF32H91",
    "SXU3XXRDBA1TGJE7KJ7MNQWM7TNVC7FDE1",
    "V2AY1D1QF5CBP232ZWUA7YMRM3A4RQ2FM6",
    "UNQ61Q7HKWK9H29CC1GB7IT6FJCPJRQ3YI",
    "XNS5DA7DVFIV2PY8HFGH81R579D6KF4A86",
    "EZU6QQWAQC9SUJJQYPH9WKQH8QTI479G5J",
    "H82415G62BA2IWCRV83994FCU8QYJ41KU9",
    "QI9YU1UI559MFKH7FX95PMNSS7SXCVIVXK",
    "54INDE61DX2HGDGJB5QAYUR1BTKG27IAD5",
    "W1BKSZZDHMY3QNRNAXVW4FWBC293V6HQH1",
    "TITM9BPCQW46C7YIGGIYY3ZW33EXWXJVKS",
    "S28KKSSWMMS4JBJWE9Q7WZMGN9P2JVB5U2",
    "MAWJUND59MMCQG72YQDVUI6NVUUTBZ8VYY",
    "5HBQDQGSTV1CA5VC6RM4QRTSDG1HGIGXPC",
    "4ZTRI78VB4I9XF53VGN71TPS68GWTZZNTD",
    "8GRIC5JAAC98QUG93FQPMKBKCEMJA5877X",
    "1EWWNJFETYJ6HTEVBHX392Z3RYNV82K5BV",
    "5TMAGBWKV8JB27MHJE2VWM4J5FC6CZJ1SV",
    "E7393UK68554GXK7XUUXSF5YITVNH4YF62",
    "MRCW3WYDQJZS2BXYK8SXIT76V1Q5M52YCV",
    "CHUVFC2R5YWJ7ZJAYHNRYACEVVJGAFP9Q6",
    "DDTUA9SCN9QWG566WFEVGSQ2GVIGZMC4RU",
    "SVB877PZZ4FIZR87MK45Q85VV6DEBAP1Q5",
    "V1GWKHCDN3A18WREMNY46CGIFG8VZID89Y",
    "R3RUF3Q23QH47B5J7W1HZGX6ZUDTGSIKK6",
    "2FN17YCA5D24XRBC4YNTDBC3HKEMGW3MSX",
    "PN26I3HNQ2ZU1GSW3TWB44QDSURUBVY6YF",
    "64ZXJFNKQG1GYVS59J6Q4CESZ21XSNE5ES",
    "8B1AVH8X12ZC8794JDG6Z2SWQ5GHDYCARK",
    "YNTB7B24SDC4BCW2EUSA2VWC71YI3BTWJF",
    "BGHCJ1MQNABVJ99XKJUM6TQETBN69GTPVT",
    "V8Y3RVW8G8TJ9MXZZJ39FNHT3MU4GG3IMC",
    "E3PB1DH4RJW4646FP8XKGN7AYXERD442GP",
    "E9D3RI1QDRAH9BMGK31JZ8DB8DVGVSYICQ",
    "8UUHNMM9REWM5YCVD4IACRIYH9U2H5RG8B",
    "TVIPUARG8Y1FTYXTJ81WE9VX4E2891C545",
    "VH53HQEW8XEVZ9U8IDTZKPFBEFJHRW449M",
    "RNC5ZQ1SJC4DVXQQ1EIR9H3WPXQSG4PYC1",
    "HH47PUYSSZX327F8ZIAYXDF8G9J69TF5HH",
    "P2YUFQX2WYA2DMWQHJI7UXKP18KHGR8V4T",
    "T1YFG2BXABFSMIWUCYNBHISS72F6D55MQJ"
  ]
}
```

## Webshare Proxy Pool

!!! warning "Contains proxy credentials"
    This proxy list includes Webshare usernames and passwords in plain text by
    request. If this repository becomes public or the credentials are exposed,
    rotate the Webshare proxy password immediately.

These proxies are intended only for the manual live Etherscan proxy probe. They
are not Athena service runtime configuration.

The original Webshare format is `ip:port:username:password`. The live probe
expects HTTP/HTTPS proxy URLs in `http://username:password@ip:port` format via
`ATHENA_E2E_ETHERSCAN_PROXY_URLS`.

`ATHENA_E2E_ETHERSCAN_PROXY_URLS` accepts comma or newline-separated values. The
following example uses newlines for readability:

```bash
export ATHENA_E2E_ETHERSCAN_PROXY_URLS='http://gwtrixbs:ih956sib66x7@31.59.20.176:6754
http://gwtrixbs:ih956sib66x7@31.56.127.193:7684
http://gwtrixbs:ih956sib66x7@45.38.107.97:6014
http://gwtrixbs:ih956sib66x7@198.105.121.200:6462
http://gwtrixbs:ih956sib66x7@64.137.96.74:6641
http://gwtrixbs:ih956sib66x7@198.23.243.226:6361
http://gwtrixbs:ih956sib66x7@2.57.21.2:7239
http://gwtrixbs:ih956sib66x7@38.154.185.97:6370
http://gwtrixbs:ih956sib66x7@142.111.67.146:5611
http://gwtrixbs:ih956sib66x7@191.96.254.138:6185'
```

Run the probe with the exported proxy list:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_STAGGERED_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
make e2e-live-etherscan-proxy-multi-key-staggered-rate-limit
```
