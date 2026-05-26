# Leveraged Markets

Source: https://docs.worm.wtf/trading/leveraged-markets

Amplify your upside on high-conviction predictions with 1x–3x leverage.

Prediction markets have a structural limitation: the higher the probability of an outcome, the lower the upside for trading on it. If a market is already at 80% Yes, the maximum return on a correct prediction is 25%, even if you're highly confident. Worm fixes this with leverage.

Worm's leveraged markets let you open positions with **1x to 3x exposure** (including fractional values like 1.5x or 2x) so your returns match your conviction.

<Warning>
  Leveraged positions can be liquidated if the market price moves against you. Only use leverage if you understand the risks involved.
</Warning>

## How it works

When you open a leveraged position, you provide the capital and Worm supplies the rest. For example, at 2x leverage with \$100:

* You put in **\$100**
* Worm adds **\$100**
* A **\$200** position is opened on your behalf

Worm sources liquidity by aggregating order books from major prediction market platforms, routing your trade for the best available price.

## Reading the trade panel

<img alt="Leverage trading panel" />

| Field                 | What it means                                                                           |
| --------------------- | --------------------------------------------------------------------------------------- |
| **Leverage**          | Your multiplier (1x–3x, fractional allowed). Slide to adjust.                           |
| **Amount**            | Your capital input in USDC.                                                             |
| **Avg. Price**        | The estimated average price at which your shares will fill, accounting for slippage.    |
| **Shares**            | The number of shares you'll receive for your chosen side.                               |
| **Liquidation Price** | If the price reaches this level, your position is force-closed to cover Worm's capital. |
| **Closing Fee**       | Charged when you win or when your position is liquidated.                               |
| **Total Return**      | Your payout if the position wins, after fees.                                           |

## Take Profit & Stop Loss

Worm provides TP/SL controls to help you manage risk automatically.

* **Take Profit**: Your position closes and profit is realized when the market price hits this level.
* **Stop Loss**: Your position closes to limit losses if the price falls to this level.

Both can be set in **cents** (e.g. 70¢) or as a **percentage** move from your entry.

## Fees

Worm charges a closing fee in two scenarios:

* ✅ **You win**: fee is taken from your profit at resolution.
* ⚠️ **You are liquidated**: fee is taken from your remaining capital.

At 1x leverage there is no liquidation risk, so the fee only applies if you win. A losing position at 1x simply returns nothing with no fee charged.

This structure aligns Worm's incentives with yours: the fee only applies when Worm's capital worked for you, or when Worm had to act to protect it.

## Liquidation

Liquidation happens when the market price falls to your **Liquidation Price**, the point at which Worm's potential loss equals your initial capital. At that point, Worm closes your position to recover the capital it supplied.

To reduce your liquidation risk, you can add funds to an existing position. Adding capital at the same leverage level moves your liquidation price further away.

## Risks to understand

Prediction markets can have thinner liquidity than traditional financial markets, and prices can move quickly around major events. A position that looks safe at 2x can reach its liquidation price faster than expected.

Some things to keep in mind:

* Leveraged markets currently support **market orders only**. Your trade fills at the best available price, which may include slippage on larger positions. Limit orders are in development.
* Rules and resolution are tied to the source markets. Worm waits for the underlying markets to resolve before settling your position.

## What's coming

Limit orders will allow you to set a specific entry price rather than filling at market. Liquidity pools are also planned, letting users deposit capital to earn yield as market makers.