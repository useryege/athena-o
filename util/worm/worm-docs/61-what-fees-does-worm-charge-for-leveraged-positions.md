# What fees does Worm charge for leveraged positions?

Source: https://docs.worm.wtf/faq/hot-questions/leveraged-fees



Worm charges a closing fee in two cases:

* **You win**: the fee is taken from your profit at resolution.
* **You are liquidated**: the fee is taken from your remaining capital when the position is force-closed.

There is no fee if you lose without being liquidated. For example, if the market resolves against you before the price reaches your liquidation level, no fee is charged.

At 1x (no leverage), the fee applies only if you win. There is no fee on a standard losing trade.