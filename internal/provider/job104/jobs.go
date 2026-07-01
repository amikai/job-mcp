package job104

type Job struct {
	JobNo    string `json:"jobNo"`
	JobName  string `json:"jobName"`
	CustName string `json:"custName"`
	CustNo   string `json:"custNo"`
	Link     struct {
		Job  string `json:"job"`
		Cust string `json:"cust"`
	} `json:"link"`
	SalaryHigh     int    `json:"salaryHigh"`
	SalaryLow      int    `json:"salaryLow"`
	JobAddrNoDesc  string `json:"jobAddrNoDesc"`
	AppearDate     string `json:"appearDate"`
	ApplyCnt       int    `json:"applyCnt"`
	RemoteWorkType int    `json:"remoteWorkType"`
}

type JobDetail struct {
	Header struct {
		JobName    string `json:"jobName"`
		CustName   string `json:"custName"`
		CustURL    string `json:"custUrl"`
		AppearDate string `json:"appearDate"`
		IsSaved    bool   `json:"isSaved"`
		IsApplied  bool   `json:"isApplied"`
	} `json:"header"`
	Contact struct {
		HRName string `json:"hrName"`
		Email  string `json:"email"`
		Reply  string `json:"reply"`
	} `json:"contact"`
	Condition struct {
		WorkExp   string   `json:"workExp"`
		Edu       string   `json:"edu"`
		Major     []string `json:"major"`
		Specialty []struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"specialty"`
	} `json:"condition"`
	Welfare struct {
		Welfare string `json:"welfare"`
	} `json:"welfare"`
	JobDetail struct {
		JobDescription string `json:"jobDescription"`
		JobCategory    []struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"jobCategory"`
		Salary        string `json:"salary"`
		SalaryMin     int    `json:"salaryMin"`
		SalaryMax     int    `json:"salaryMax"`
		JobType       int    `json:"jobType"`
		AddressRegion string `json:"addressRegion"`
		AddressDetail string `json:"addressDetail"`
		ManageResp    string `json:"manageResp"`
		NeedEmp       string `json:"needEmp"`
		RemoteWork    string `json:"remoteWork"`
	} `json:"jobDetail"`
	Industry  string `json:"industry"`
	Employees string `json:"employees"`
	CustNo    string `json:"custNo"`
}

// Codes confirmed live via the site's own area filter UI (not sequential/
// guessable from the value): the naive assumption that area codes increment
// in some obvious order (city population, alphabetical, etc.) is wrong.
const (
	AreaTaipei    = "6001001000"
	AreaNewTaipei = "6001002000"
	AreaTaoyuan   = "6001005000"
	AreaTaichung  = "6001008000"
	AreaTainan    = "6001014000"
	AreaKaohsiung = "6001016000"
)

// Confirmed live via the site's own 上班型態 filter tabs (全職/兼職). ro=0
// is accepted but a no-op (same result as omitting the param entirely).
const (
	ROFullTime = 1
	ROPartTime = 2
)

const (
	OrderRelevance = 1
	OrderNewest    = 15
)

// remoteWork only accepts these two values server-side (confirmed live: 0 and
// 3 both 400 with "Items in remoteWork is invalid: validation.in"). There is
// no explicit "no remote" value — omit RemoteWork entirely for that.
// Confirmed via the site's own filter UI (not guessable from the value
// alone): 1 = 完全遠端 (fully remote), 2 = 部分遠端 (partial/hybrid remote).
const (
	RemoteWorkFull    = 1
	RemoteWorkPartial = 2
)

type JobsRequest struct {
	Keyword    string
	Area       string
	RO         *int // one of RO*
	Order      *int // one of Order*
	Page       *int
	Edu        string
	RemoteWork *int   // one of RemoteWork*
	S9         string // experience codes, comma-separated
}

type JobsResponse struct {
	Data     []Job `json:"data"`
	Metadata struct {
		Pagination struct {
			CurrentPage int `json:"currentPage"`
			LastPage    int `json:"lastPage"`
			Total       int `json:"total"`
		} `json:"pagination"`
	} `json:"metadata"`
}

type JobDetailResponse struct {
	Data JobDetail `json:"data"`
}
