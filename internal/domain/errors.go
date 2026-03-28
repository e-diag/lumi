package domain

import "errors"

// Sentinel-ошибки для domain-слоя.
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrNodeNotFound         = errors.New("node not found")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrRoutingRuleNotFound  = errors.New("routing rule not found")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrSubscriptionExpired  = errors.New("subscription expired")
	ErrAlreadyExists        = errors.New("already exists")
	ErrTrialAbuseIP         = errors.New("trial quota exceeded for this network")
	ErrReferralAbuseSameIP  = errors.New("referral from same network as inviter")
	ErrReferralMonthlyCap   = errors.New("referral bonus monthly limit reached")
)
