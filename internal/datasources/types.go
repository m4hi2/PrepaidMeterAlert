package datasources

type Identifier struct {
	AccountNumber string
	MeterNumber   string
}

type Balance struct {
	Identifier
	Balance float64
}

type AccountDetails struct {
	Identifier
	CustomerName    string
	CustomerContact string
	SanctionedLoad  string
	Address         string
}
