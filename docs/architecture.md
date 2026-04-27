# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->

Athena is a auto trading system for the blockchain.It can automatically buy and sell the token based on the project data and the user's strategy.It can also protect the user's assets from the loss.


![Athena Architecture](assets/athena-architecture.png)


## Core Components

### Block Sniffer

Subscribe the lastest block height and use it to monitor the blockchain real-time tx and logs events.

When the tx and the logs events that we interested in are detected, it will emit a sync event to the **Project Controller**.


---

### Project Controller

Receive the sync event from the **Block Sniffer** and fetch the latest project data from the blockchain or external sources. maintain the project data in the memory. refresh the project data periodically to keep the project data up to date.

---

### Projects

It is a collection of token information that can be fetched from the blockchain or external sources.The information can be used to judge the project is valuable for trading or not.

---

### Buy Strategy Engine

It subscribe the project data from the **Project Controller** and use the strategy to decide when to buy.If the project is valuable for trading, it will call the **Swap Server** to buy the token.

---

### Sell Strategy Engine

It subscribe the project data from the **Project Controller** and use the strategy to decide when to sell.It will call the **Swap Server** to sell the token.
---

### API Server

It is the entry of the system.It is used to connect the UI and the backend services.It is also used to control the system.

---

### Order Controller

It is used to manage the orders.It will keep the entire order lifecycle when **Buy Strategy Engine** or **Sell Strategy Engine** decide to buy or sell the token. user can also trigger the order manually.User can alse buy(start) or sell(end) the token manually.

---

### Order

It is basic information of a token in our wallet.It will keep the token balance and the token price. It will also keep the token transaction history. And many other important information of the token. User will use this information to make the decision to sell the token.

---

### Swap Server

It is used to create the swap transaction.It will call the **Tx Speed Up Server** to speed up the transaction.

---

### Protect Server

It is used to protect the user's assets from the loss.It will detect the token price is too low and call the **Swap Server** to sell the token.Predicting impending dangers and take action to protect the user's assets.It will also call the **Tx Speed Up Server** to speed up the transaction.

---

### Tx Speed Up Server

It is used to speed up the transaction.

---
