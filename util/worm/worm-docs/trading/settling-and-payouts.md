> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Settling & Payouts

> When markets resolve, how long payouts take, and what 'under review' means.

## How payouts work

When a market resolves, settlement happens automatically on-chain. The winning side is worth \$1 per share and the losing side \$0. There is no separate claim step: funds are distributed directly to your wallet as part of the resolution transaction.

## When will my market resolve?

Markets resolve **at or after their deadline**, never before, even if the outcome is already clear. How long after the deadline depends on the market.

For **sports match markets**, Worm monitors events using live data feeds. These markets typically resolve within a few minutes of the event ending, with no manual review required.

For **all other markets**, the process is:

1. The deadline passes
2. Worm's AI attempts to resolve the market using its data sources
3. A member of the Worm team reviews and approves the resolution
4. Funds are settled on-chain

The review step means resolution time varies. Simpler markets with clear, verifiable outcomes resolve faster. More complex markets may take longer. If a market can't be resolved at the first attempt, it will be retried automatically after a waiting period.

<Note>
  **Waiting for your payout?** Resolution and payout happen together. Once a market resolves, funds reach your wallet within about a minute as the on-chain transaction processes.
</Note>

## What does "Under Review" mean?

<Frame>
  <img src="https://mintcdn.com/worm/GppqAf3UEX2nw1bR/images/market_under_review.png?fit=max&auto=format&n=GppqAf3UEX2nw1bR&q=85&s=c6bc9389b1b737f069d3d5f37d02cc6b" alt="Market under review: results coming soon" width="2940" height="1680" data-path="images/market_under_review.png" />
</Frame>

When a market shows **Under Review**, it means the event has concluded, Worm's AI has proposed a resolution, and a team member is reviewing it before it's finalized. Trading is paused during this period. Once the review is approved, the market resolves and payouts process automatically.

Worm doesn't yet have an open dispute mechanism, so review is handled internally rather than through a public process.

## Cashing out early

As long as a market is active and not under review, you can exit a position at any time by selling your shares back into the market. Selling on Worm is not a withdrawal: it's a trade. Your shares are matched against buyers in the order book, and your PnL is realized at the price you sell.

After the trade executes, funds typically appear in your wallet within about a minute as the on-chain transaction settles. If you've waited longer than that and still don't see anything, [open a support ticket](https://discord.gg/jQgQssQHT9).
