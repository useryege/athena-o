> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Liquidity Rewards

> Get paid daily for placing limit orders that tighten the spread on select markets.

Liquidity Rewards pay you for making markets better. On a handful of selected markets, Worm sets aside a **daily reward pool** and splits it between traders who keep good limit orders resting on the book. If your order sits near the current price and is large enough, you earn a slice of that pool every day, just for having the order open.

You don't have to get the outcome right, and you don't have to trade at all. The reward is for **providing liquidity**, not for winning the bet.

<Note>
  Rewards are for **limit orders that rest on the order book** (maker orders). Market orders fill instantly and never rest, so they don't qualify. Only markets that support limit orders can run rewards.
</Note>

## Which markets have rewards

Only a **hand-picked set of markets** are incentivized at any time, not every market. There are two easy ways to spot them:

* Open the **Rewards** tab in the top navigation to see every incentivized market in one place.
* Look for the **blue diamond** on a market card. It marks a market that's currently paying rewards.

<Frame>
  <img src="https://mintcdn.com/worm/4fjlaOo16AI9ksja/images/liquidity-rewards-nav.png?fit=max&auto=format&n=4fjlaOo16AI9ksja&q=85&s=0213c8a60b671ba1835374f9070cf3cf" alt="The Rewards tab in the top navigation and the blue diamond on a rewards market card" width="1344" height="994" data-path="images/liquidity-rewards-nav.png" />
</Frame>

On the market page itself, you'll see an **Earn Rewards** link under the title and a diamond on the **Order Book** header. Hover either one and a tooltip explains the market's reward terms.

## How to earn

Three things have to be true for an open order to earn rewards:

<Steps>
  <Step title="Place a limit order">
    Use a **limit** order (Buy or Sell), not a market order. Set your price and leave the order resting on the book. That's what makes you a liquidity provider.
  </Step>

  <Step title="Price it near the midpoint">
    Your order has to sit within the market's **Max Spread** of the current midpoint price. Quotes far away from the action don't help liquidity, so they don't earn. Orders closer to the midpoint generally earn more.
  </Step>

  <Step title="Meet the minimum size">
    Your order needs at least the market's **Min Shares** of remaining, unfilled shares. An order that's been mostly filled can drop below the minimum and stop earning. In the limit form, a quick-set button (e.g. **1000**) fills in the minimum for you.
  </Step>
</Steps>

<Note>
  Rewards only pay out when **both sides are quoted**: at least one buy **and** at least one sell must independently meet the Min Shares minimum near the midpoint. For as long as one side has only undersized orders, no one earns on that market, not even the side that qualifies.
</Note>

Each market publishes its own terms. Hover the diamond tooltip to see them:

<Frame>
  <img src="https://mintcdn.com/worm/TPPVaEtARURZQJWX/images/liquidity-rewards-tooltip.png?fit=max&auto=format&n=TPPVaEtARURZQJWX&q=85&s=99a081d45341b5c9610d6d654babbac8" alt="Reward terms tooltip showing Rewards, Max Spread and Min Shares" width="298" height="150" data-path="images/liquidity-rewards-tooltip.png" />
</Frame>

| Term           | What it means                                                             |
| -------------- | ------------------------------------------------------------------------- |
| **Rewards**    | The size of that market's daily reward pool.                              |
| **Max Spread** | How far from the midpoint your order can be and still qualify (e.g. ±2¢). |
| **Min Shares** | The minimum order size to be eligible (e.g. 1000 shares).                 |

<Info>
  The exact numbers (pool size, max spread, and minimum shares) are set per market and shown in the tooltip. The values above are examples, not fixed platform settings.
</Info>

## How the pool is shared

The daily pool is split among everyone whose orders qualify, in proportion to how much good liquidity they provide. Put simply: **the more you help (bigger orders, tighter to the midpoint, resting longer), the bigger your share.**

Your share is shown as a **percentage** on your profile. It's the slice of that market's pool your orders are earning *right now*, not a figure locked in for the day: it moves as other traders' orders arrive, fill, and cancel.

### Rewards accrue by the second

Eligibility isn't checked once a day, it's tracked **second by second**. Every second your order meets the terms, you earn your slice of the pool for that second. Post an order at noon and it starts earning from that moment; cancel it an hour later and you keep exactly the hour it earned. The same works in reverse: any stretch where your order drifts outside the Max Spread, drops below Min Shares, or the other side of the book goes unquoted is simply time you don't get paid for, and earning resumes the moment things qualify again.

### Payouts

Payouts run **once a day, at the end of the day (UTC)**, straight to your wallet. There's nothing to claim.

<Note>
  There is a **\$1 minimum payout**. If everything you accrued over the day comes to less than \$1, it's discarded rather than paid, and it does not roll over into the next day.
</Note>

### A quick example

Say a market has a **\$500 daily reward pool**, a **Max Spread of ±2¢**, and a **Min Shares of 1,000**. The midpoint is sitting at **60¢**, so to qualify, an order has to rest within 2¢ of the midpoint (between **58¢ and 62¢**) and be at least **1,000 shares**.

Two people are providing liquidity on this market today. Both leave their orders resting for the **full day**, and both post at the same distance from the midpoint: a two-sided quote **4¢ apart**, i.e. exactly **2¢ on each side of the 60¢ midpoint**:

* **You** post a limit **buy for 2,000 shares at 58¢** and a limit **sell for 2,000 shares at 62¢**, for **4,000** qualifying shares.
* **Bob** posts a limit **buy for 3,000 shares at 58¢** and a limit **sell for 3,000 shares at 62¢**, for **6,000** qualifying shares.

Because both quotes sit the same 2¢ from the midpoint, the only thing separating you is **size**. The pool splits in proportion to qualifying shares, and total qualifying liquidity is **10,000 shares**:

| Trader  | Orders                             | Qualifying shares | Share of pool | Payout    |
| ------- | ---------------------------------- | ----------------- | ------------- | --------- |
| **You** | 2,000 buy @ 58¢ + 2,000 sell @ 62¢ | 4,000             | 40%           | **\$200** |
| **Bob** | 3,000 buy @ 58¢ + 3,000 sell @ 62¢ | 6,000             | 60%           | **\$300** |

Your 4,000 shares are 40% of the 10,000 total, so you earn 40% of the \$500 pool = **\$200**; Bob's larger order earns him the other **\$300**.

<Note>
  We kept both quotes at the same distance, and both resting all day, to make the math clean. In practice, orders resting **closer to the midpoint** earn a bigger share for the same size, and an order that only rests for part of the day only earns for the seconds it was up.
</Note>

Leave your orders resting and you keep earning day after day. Cancel them, let them get fully filled, or let the price drift so they fall outside the spread, and they simply stop earning until they qualify again.

<Tip>
  Two-sided quotes (a buy **and** a sell near the midpoint) provide the most liquidity, which is exactly what the rewards are designed to encourage.
</Tip>

## Where to track your earnings

Your profile is your rewards dashboard:

<Frame>
  <img src="https://mintcdn.com/worm/4fjlaOo16AI9ksja/images/liquidity-rewards-profile.png?fit=max&auto=format&n=4fjlaOo16AI9ksja&q=85&s=7a5d77248e201ba98093ed52b141adf9" alt="Liquidity Rewards card on the profile showing total and today's earnings" width="1350" height="834" data-path="images/liquidity-rewards-profile.png" />
</Frame>

* The **Liquidity Rewards** card shows your total earnings and how much you've earned today.
* The **Rewards** tab lists the open orders that are earning right now, with your share for each.

On any incentivized market, the diamond on the **Order Book** header (and the **Earn Rewards** link under the title) opens the tooltip with that market's reward terms.

## Good to know

* Rewards are separate from any profit or loss on the position itself. You can earn rewards on an order whether the outcome ends up Yes or No.
* Only the **unfilled** portion of an order counts toward the minimum size.
* If a market is removed from the rewards program, its diamond disappears and orders stop earning, but the orders themselves stay on the book as normal.

### How eligibility is actually measured

A few details decide whether a resting order counts, beyond just being near the midpoint and above the minimum:

* **Both sides have to be covered.** A market only distributes rewards when there's a qualifying order on the buy side **and** the sell side at the same time. Say Min Shares is 20: if the best bid is only a 15-share order while the ask side has a healthy 50-share order, nobody earns for those seconds, not even the sell side, until a 20-share (or larger) buy shows up to balance the book. The two sides can even belong to the same account.
* **Orders at the same price add up.** Eligibility is checked per price level, combining all your orders resting at that exact price. Two separate **10-share buys at 60¢** count as a single **20-share** order and clear a 20-share minimum, exactly as one 20-share order would.
* **Each price level stands on its own.** An order is never topped up by your orders at other prices. If you rest a qualifying **20-share buy at 60¢** (earning) and then add a **5-share buy at 59¢**, that 5-share order is below the minimum at its own price, so it earns nothing, even though your 60¢ order keeps earning. It's judged by the size resting at 59¢, not by your total across the book.

<Card title="How Markets Work" icon="book-open" href="/trading/how-markets-work">
  New to limit orders and the order book? Start here.
</Card>
