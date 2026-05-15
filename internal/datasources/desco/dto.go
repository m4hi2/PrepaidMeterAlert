package desco

/*
GetBalanceResp is the response from DESCO website for getting balance of an account
Endpoint: https://prepaid.desco.org.bd/api/unified/customer/getBalance?accountNo=#####&meterNo=#####

	{
	    "code": 200,
	    "desc": "OK",
	    "data": {
	        "accountNo": "########",
	        "meterNo": "############",
	        "balance": 3396.89,
	        "currentMonthConsumption": 3643.23191,
	        "readingTime": "2026-04-28"
	    }
	}
*/
type GetBalanceResp struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
	Data struct {
		AccountNo               string  `json:"accountNo"`
		MeterNo                 string  `json:"meterNo"`
		Balance                 float64 `json:"balance"`
		CurrentMonthConsumption float64 `json:"currentMonthConsumption"`
		ReadingTime             string  `json:"readingTime"`
	} `json:"data"`
}
