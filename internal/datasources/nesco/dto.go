package nesco

const (
	AccountNumber  = "Consumer No."
	MeterNumber    = "Meter No."
	Balance        = "Remaining Balance (Tk.)"
	ReadingTimeLbl = "Last Reading Date"
)

const (
	panelPath      = "/pre/panel"
	languageEn     = "/language/en"
	submitRecharge = "Recharge History"
	paramCustNo    = "cust_no"
	paramToken     = "_token"
	paramSubmit    = "submit"
)

type NescoBalanceResp struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
	Data struct {
		AccountNo   string `json:"accountNo"`
		MeterNo     string `json:"meterNo"`
		Balance     string `json:"balance"`
		ReadingTime string `json:"readingTime"`
	} `json:"data"`
}
