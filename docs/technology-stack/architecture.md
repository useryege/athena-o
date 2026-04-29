# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->

Athena is an automated trading system designed for blockchain transaction scenarios. It continuously monitors on-chain data, maintains project data, and automatically determines buy and sell timing based on user-configured strategies. When strategy conditions are met, the system creates transactions through the Swap Server and, when necessary, accelerates confirmation through the Tx Speed Up Server. At the same time, the Protect Server continuously monitors user assets and order status, triggering protective sell actions when risks are detected to reduce potential losses.

![Athena Architecture](assets/athena-architecture.png)

# Athena Overall Workflow Overview

Athena is an automated trading system designed for blockchain transaction scenarios. It continuously monitors on-chain data, maintains project data, and automatically determines buy and sell timing based on user-configured strategies. When strategy conditions are met, the system creates transactions through the Swap Server and, when necessary, accelerates confirmation through the Tx Speed Up Server. At the same time, the Protect Server continuously monitors user assets and order status, triggering protective sell actions when risks are detected to reduce potential losses.

The system can be divided into five core workflows:

1. Project Data Synchronization Workflow
2. Automated Buy Workflow
3. Automated Sell Workflow
4. Manual Trading Workflow
5. Risk Control and Protection Workflow

Athena first uses the Block Sniffer to monitor real-time blockchain transactions and log events. 
When target events are detected, the Block Sniffer emits a sync event to the Project Controller. 
The Project Controller fetches the lastest project data from the blockchain or external sources and maintains it in memory. 
The Projects module stores token and project information used by the strategy engines.

The Buy Strategy Engine subscribes to project data and decides when to buy based on the user's buy strategy. 
The Sell Strategy Engine also subscribes to project data and decides when to sell based on the user's sell strategy. 
When a buy or sell condition is met, the strategy engine sends a trade signal to the Swap Server. 
The Swap Server creates the swap transaction and calls the Tx Speed Up Server when transaction acceleration is required.

The API Server acts as the entry point of the system. 
It connects the UI with backend services, allows users to start or stop the Block Sniffer, manually trigger trades, query project and order status, and open or close the Protect Server.

The Order Controller manages the full lifecycle of orders and synchronizes order data into Orders. 
Orders store the user's wallet token information, balances, prices, transaction history, and other data needed for trading decisions.

The Protect Server monitors order and asset status. 
When it detects potential risks, such as a sharp price drop or an unsafe token state, it triggers an automatic sell through the Swap Server and may use the Tx Speed Up Server to accelerate the protective transaction.

---

## Project Data Synchronization Workflow

This is the foundational workflow of the entire system.

The **Block Sniffer** listens to the lastest block height and monitors on-chain transactions and log events in real time. When it detects transactions or log events that the system is interested in, it sends a sync event to the **Project Controller**.

The flow is:

Block Sniffer
  -> Emit Sync Event
Project Controller
  -> Sync
Projects

Specifically:

* Block Sniffer subscribes to the lastest block height.
* Block Sniffer monitors real-time on-chain transactions and log events.
* When target events are detected, it triggers a sync event.
* Project Controller receives the sync event.
* Project Controller fetches the lastest project data from the blockchain or external data sources.
* Project Controller updates project data in memory.
* Projects stores project information used by strategy engines for decision-making.

Here, **Projects** can be understood as the system's internal project data collection, including token information, on-chain status, liquidity, prices, transaction history, and external data. The subsequent buy and sell strategies both rely on this dataset.

---

## Automated Buy Workflow

The automated buy workflow is handled by the **Buy Strategy Engine**.

It subscribes to project data and evaluates whether a token should be bought based on user-configured buy strategies. If buy conditions are met, it sends a buy signal to the **Swap Server**, which then creates the buy transaction.

The flow is:

Projects
  -> Subscribe
Buy Strategy Engine
  -> Emit Order Created Signal
  -> Emit Buy Signal
  -> Emit Order Status Changed Signal(Status: Waiting to Create the Buy Transaction)
Swap Server
  -> Emit Order Status Changed Signal(Status: Started to Create the Buy Transaction)
  -> Create Buy Transaction
  -> Emit Order Status Changed Signal(Status: Create the Buy Transaction Success and Waiting to Speed Up the Transaction)
  -> Emit Speed Up Transaction Signal
Tx Speed Up Server
  -> Emit Order Status Changed Signal(Status: Started to Speed Up the Transaction)
  -> Speed Up the Transaction
Order Controller
  -> Wait for the Transaction to be confirmed
  -> Emit Order Status Changed Signal(Status: Transaction Confirmed)

Specifically:

* The Buy Strategy Engine subscribes to updates from Projects.
* When buy conditions are met, the Buy Strategy Engine emits an `Order Created` signal.
* The Buy Strategy Engine emits a `Buy` signal.
* The Buy Strategy Engine emits an order status update: `Waiting to Create the Buy Transaction`.
* The Swap Server emits an order status update: `Started to Create the Buy Transaction`.
* The Swap Server creates the buy transaction.
* The Swap Server emits an order status update: `Create the Buy Transaction Success and Waiting to Speed Up the Transaction`.
* The Swap Server emits a `Speed Up Transaction` signal.
* The Tx Speed Up Server emits an order status update: `Started to Speed Up the Transaction`.
* The Tx Speed Up Server speeds up the transaction.
* The Order Controller waits for transaction confirmation.
* The Order Controller emits an order status update: `Transaction Confirmed`.

---

## Automated Sell Workflow

The automated sell workflow is handled by the **Sell Strategy Engine**.

It also subscribes to project data, but focuses on whether a token that has already been bought or held has reached the sell timing. It will decide whether to trigger a sell based on user-configured sell strategies, such as profit-taking, price changes, time windows, project risk changes, and other conditions.

The flow is:

Projects
  -> Subscribe
Sell Strategy Engine
  -> Emit Order Created Signal
  -> Emit Sell Signal
  -> Emit Order Status Changed Signal(Status: Waiting to Create the Sell Transaction)
Swap Server
  -> Emit Order Status Changed Signal(Status: Started to Create the Sell Transaction)
  -> Create Sell Transaction
  -> Emit Order Status Changed Signal(Status: Create the Sell Transaction Success and Waiting to Speed Up the Transaction)
  -> Emit Speed Up Transaction Signal
Tx Speed Up Server
  -> Emit Order Status Changed Signal(Status: Started to Speed Up the Transaction)
  -> Speed Up the Transaction
Order Controller
  -> Wait for the Transaction to be confirmed
  -> Emit Order Status Changed Signal(Status: Transaction Confirmed)


Specifically:

* The Sell Strategy Engine subscribes to updates from Projects.
* When sell conditions are met, the Sell Strategy Engine emits an `Order Created` signal.
* The Sell Strategy Engine emits a `Sell` signal.
* The Sell Strategy Engine emits an order status update: `Waiting to Create the Sell Transaction`.
* The Swap Server emits an order status update: `Started to Create the Sell Transaction`.
* The Swap Server creates the sell transaction.
* The Swap Server emits an order status update: `Create the Sell Transaction Success and Waiting to Speed Up the Transaction`.
* The Swap Server emits a `Speed Up Transaction` signal.
* The Tx Speed Up Server emits an order status update: `Started to Speed Up the Transaction`.
* The Tx Speed Up Server speeds up the transaction.
* The Order Controller waits for transaction confirmation.
* The Order Controller emits an order status update: `Transaction Confirmed`.

---

## Core Components

### Block Sniffer

Subscribe the lastest block height and use it to monitor the blockchain real-time tx and logs events.When the tx and the logs events that we interested in are detected, it will emit a sync event to the **Project Controller**.

Maintain Target Pair Address
  -> Maintain the lastest target Pair Address in the momory. Fetch the target Pair data from the **Project Controller**

Sniffer For Swap Events(Here Must be as Fast as Possible)
  -> Subscribe the lastest block height form evm node.
  -> Filter the logs by the target Pair Address and the swap events.
  -> When the target logs are detected, it will emit a sync event to the **Project Controller**.

Sniffer For Create Token Tx
  -> Subscribe the lastest block height form evm node.
  -> Filter the tx by the block height.
  -> Tx:
    -> Create the ERC20 New Token
---

### Project Controller

Receive the sync event from the **Block Sniffer** and fetch the lastest project data from the blockchain or external sources. maintain the project data in the memory. refresh the project data periodically to keep the project data up to date.

Sync Project Data
  -> Fetch the lastest project data from the blockchain or external sources.
  -> Maintain the project data in the memory.
  -> Refresh the project data periodically to keep the project data up to date.

Sync Target Pair Address
  -> Filter the target pair address from the project data.

---


## Project

A **Project** represents the important token, pool and wallet data used by strategy engines which is used to decide whether a trade is worth executing.

Token data includes:

Pool data includes:

Wallet data includes:


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

### Swap Server

It is used to create the swap transaction.It will call the **Tx Speed Up Server** to speed up the transaction.

---

### Protect Server

It is used to protect the user's assets from the loss.It will detect the token price is too low and call the **Swap Server** to sell the token.Predicting impending dangers and take action to protect the user's assets.It will also call the **Tx Speed Up Server** to speed up the transaction.

---

### Tx Speed Up Server

It is a low-level supporting service. It does not make strategy decisions and does not create transactions; it is only responsible for accelerating transaction confirmation.

It can be called by:

Swap Server -> Tx Speed Up Server
Protect Server -> Tx Speed Up Server

Its main use cases include:

* increasing gas fees
* replacing pending transactions
* rebroadcasting transactions
* accelerating buy transactions
* accelerating sell transactions
* speeding up emergency exits in protection scenarios

From a responsibility boundary perspective, the cleanest design is:

Swap Server handles all transaction creation
Tx Speed Up Server handles all transaction acceleration

Protect Server should normally notify Swap Server to execute protective sells instead of directly handling transaction logic. Only in highly urgent scenarios should Protect Server call Tx Speed Up Server directly.

---

## Order

An **Order** represents a user's position and execution state for a token. It is the core record used by strategy engines, manual operations, and risk-control modules to make follow-up decisions.

Key fields include:

* token balance
* token price
* transaction history
* buy price
* sell price
* profit / loss
* order status
* risk status
* additional decision data used for future sell judgments

Order lifecycle management is primarily handled by the **Order Controller**, which continuously synchronizes events from strategy engines, API-triggered manual actions, Swap Server, and Tx Speed Up Server.

A typical lifecycle includes:

* Order Created
* Waiting to Create Transaction
* Started to Create Transaction
* Transaction Created
* Waiting to Speed Up Transaction (if needed)
* Started to Speed Up Transaction
* Waiting for Transaction Confirmation
* Transaction Confirmed / Transaction Failed
* Position Updated (holding, partial sell, full sell, closed)

---
