package application

type ProjectReport struct {
	IsPolicyEvaluated            bool
	IsBlacklistedCreatorWallet   bool
	IsBlacklistedGenesisWallet   bool
	IsBlacklistedBytecode        bool
	IsBlacklistedSourceCode      bool
	IsBlacklistedSourceCodeField bool
	HasMintRisk                  bool
}
