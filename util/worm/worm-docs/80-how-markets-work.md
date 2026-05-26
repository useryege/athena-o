# How Markets Work

Source: https://docs.worm.wtf/trading/how-markets-work

Order books, prices, liquidity, and how trades fill on Worm.

Worm is an **order-book aggregator**. This means prices aren't set by an algorithm: they emerge from real bids and asks placed by traders. Understanding this shapes how you trade effectively.

## The order book model

Every market on Worm has two sides: **Yes** and **No**. Each share is priced between 0 and 1 USDC, and the Yes price reflects the market's estimated probability that the event occurs.

When you place a trade, you're buying shares from another trader who's willing to sell at that price, or selling to one who's willing to buy. The current price is simply where buyers and sellers have most recently agreed.

<Note>
  Order types differ by market type. **User-created markets** support both market orders and **limit orders**. You can set a specific price you're willing to buy or sell at. **Leveraged markets** currently support market orders only. Limit orders for leveraged markets are in development.
</Note>

## Market formats

Worm supports two market formats:

**Binary** markets are the standard format: a single event either happens or it doesn't. You trade Yes or No.

**Categorical** markets have multiple named outcomes. Each outcome has its own Yes/No price and is traded independently. Sports matches (Team A / Draw / Team B) and multi-outcome questions ("Which of these will happen?") are both categorical markets.

## Two types of markets

### Leveraged markets

Leveraged markets are curated by Worm and backed by aggregating Worm's own order book with those of major prediction market platforms. This means deep liquidity, tight spreads, and reliable price discovery.

The rules and resolution for these markets are tied to their source markets. Resolution on Worm waits for the underlying markets to resolve, even if the outcome seems obvious earlier.

### Permissionless markets (created on Worm)

<Frame>
  <img alt="Permissionless market with empty order book" />
</Frame>

Markets created by Worm users have their own independent order books that start empty. Liquidity builds as traders discover and trade on the market. A brand-new market with no trades won't have a visible price yet. The first orders define it. A market stuck at 50% often means no one has traded yet, not that the event is truly a coin flip.

If you created a market and want to bootstrap liquidity, sharing it in relevant communities is the most effective way to get initial traders.

## How prices move

Prices change when new trades happen. If many people buy Yes, the Yes price rises. If new information makes the outcome seem less likely, sellers push the price down. Over time, markets with active participants tend to converge toward accurate probabilities.

## At resolution

When a market resolves, the correct side is worth **\$1 per share** and the incorrect side **\$0**. Traders who held the correct side are paid out automatically on-chain with no manual claim required.

<Card title="Settling & Payouts" icon="circle-check" href="/trading/settling-and-payouts">
  Learn about resolution timing, the "under review" state, and how cash-outs work.
</Card>