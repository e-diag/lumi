package domain

import "fmt"

// SubscriptionPriceKopeks — стоимость периода в копейках (пропорция от «цены за 30 дней»).
func SubscriptionPriceKopeks(tier SubscriptionTier, days int) (int64, error) {
	price30, ok := map[SubscriptionTier]int64{
		TierBasic:   14900,
		TierPremium: 29900,
	}[tier]
	if !ok {
		return 0, fmt.Errorf("unknown tier: %s", tier)
	}
	if days <= 0 {
		return 0, fmt.Errorf("invalid days: %d", days)
	}
	amount := price30 * int64(days)
	kopeks := amount / 30
	rem := amount % 30
	if rem*2 >= 30 {
		kopeks++
	}
	return kopeks, nil
}
