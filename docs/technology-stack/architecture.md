# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->

Athena is an automated trading system designed for blockchain transaction scenarios. It continuously monitors on-chain data, maintains project data, and automatically determines buy and sell timing based on user-configured strategies. When strategy conditions are met, the system creates transactions through the Swap Server and, when necessary, accelerates confirmation through the Tx Speed Up Server. At the same time, the Protect Server continuously monitors user assets and order status, triggering protective sell actions when risks are detected to reduce potential losses.

![Athena Architecture](assets/athena-architecture.png)
