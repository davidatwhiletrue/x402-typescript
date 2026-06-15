package facilitator

const (
	ErrUnsupportedScheme   = "invalid_exact_casper_unsupported_scheme"
	ErrNetworkMismatch     = "invalid_exact_casper_network_mismatch"
	ErrMalformedPayload    = "invalid_exact_casper_malformed_payload"
	ErrPayToMismatch       = "invalid_exact_casper_payto_mismatch"
	ErrAmountMismatch      = "invalid_exact_casper_amount_mismatch"
	ErrInvalidPayTo        = "invalid_exact_casper_invalid_payto"
	ErrInvalidAmount       = "invalid_exact_casper_invalid_amount"
	ErrInvalidAsset        = "invalid_exact_casper_invalid_asset"
	ErrPayloadExpired      = "invalid_exact_casper_payload_expired"
	ErrNotYetValid         = "invalid_exact_casper_not_yet_valid"
	ErrInsufficientTime    = "invalid_exact_casper_insufficient_time_to_settle"
	ErrMissingTokenName    = "invalid_exact_casper_missing_token_name"
	ErrMissingTokenVersion = "invalid_exact_casper_missing_token_version"
	ErrFailedToHash        = "invalid_exact_casper_failed_to_hash"
	ErrInvalidSignature    = "invalid_exact_casper_invalid_signature"

	ErrVerificationFailed = "invalid_exact_casper_verification_failed"
	ErrBuildDeployFailed  = "invalid_exact_casper_build_deploy_failed"
	ErrSignDeployFailed   = "invalid_exact_casper_sign_deploy_failed"
	ErrPutDeployFailed    = "invalid_exact_casper_put_deploy_failed"
	ErrWaitDeployFailed   = "invalid_exact_casper_wait_deploy_failed"
)
