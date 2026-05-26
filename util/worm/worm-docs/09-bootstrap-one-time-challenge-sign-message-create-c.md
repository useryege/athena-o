# --- Bootstrap (one-time): challenge → sign message → create credential ---

keypair = load_keypair(os.environ["WORM_PRIVATE_KEY"])
wallet_address = str(keypair.pubkey())

challenge = unwrap(
    requests.post(
        f"{BASE}/auth/keys/challenge/",
        json={"wallet_address": wallet_address},
        timeout=30,
    )
)
nonce = challenge["nonce"]
message = challenge["message"]

signature = keypair.sign_message(message.encode("utf-8"))
signature_hex = signature.to_bytes().hex()

creds = unwrap(
    requests.post(
        f"{BASE}/auth/keys/create/",
        json={
            "wallet_address": wallet_address,
            "message": message,
            "signature": signature_hex,
            "nonce": nonce,
        },
        timeout=30,
    )
)
API_KEY = creds["api_key"]
API_SECRET = creds["secret"]