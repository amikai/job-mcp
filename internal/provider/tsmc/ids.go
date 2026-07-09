package tsmc

// These opaque filter IDs were observed from TSMC's live en_US form on
// 2026-07-03 and cannot be derived from their labels.

// LocationIDs maps a "Location" filter label to its field-1277 value.
var LocationIDs = map[string]string{
	"Taiwan":               LocTaiwan,
	"Canada":               LocCanada,
	"China":                LocChina,
	"Germany-Dresden":      LocGermanyDresden,
	"Germany-Munich":       LocGermanyMunich,
	"Japan-Yokohama":       LocJapanYokohama,
	"Japan-Osaka":          LocJapanOsaka,
	"Japan-Tsukuba":        LocJapanTsukuba,
	"Japan-Kumamoto":       LocJapanKumamoto,
	"Korea":                LocKorea,
	"Netherlands":          LocNetherlands,
	"USA-Arizona":          LocUSAArizona,
	"USA-California":       LocUSACalifornia,
	"USA-Massachusetts":    LocUSAMassachusetts,
	"USA-Texas":            LocUSATexas,
	"USA-Washington":       LocUSAWashington,
	"USA-Washington, D.C.": LocUSAWashingtonDC,
}

// CategoryIDs maps a "Job Category" filter label to its field-558 value.
var CategoryIDs = map[string]string{
	"R&D":                  CatRD,
	"Specialty Technology": CatSpecialtyTechnology,
	"IC Design Technology": CatICDesignTechnology,
	"Manufacturing (fabs)": CatManufacturing,
	"Facility & Industrial Safety / Environmental Protection": CatFacilityAndSafety,
	"Product Development":                           CatProductDevelopment,
	"R&D Advanced Packaging Technology Development": CatICPackagingTechnology,
	"Testing Development and Technology":            CatTestingDevelopment,
	"Quality and Reliability":                       CatQualityAndReliability,
	"Information Technology":                        CatIT,
	"Internal Audit":                                CatInternalAudit,
	"Business Development":                          CatBusinessDevelopment,
	"Customer Service":                              CatCustomerService,
	"Corporate Planning":                            CatCorporatePlanning,
	"Finance / Accounting / Risk Management":        CatFinance,
	"Human Resources":                               CatHumanResources,
	"Legal":                                         CatLegal,
	"Materials Management":                          CatMaterialsManagement,
	"Corporate Sustainability (ESG)":                CatCorporateSustainability,
	"Administration":                                CatAdministration,
	"Accessibility Inclusion":                       CatAccessibilityInclusion,
}

// JobTypeIDs maps a job-level label to its field-147 value.
var JobTypeIDs = map[string]string{
	"Technician":                 JobTypeTechnician,
	"Associate Engineer / Admin": JobTypeAssociateEngineer,
	"Engineer / Admin":           JobTypeEngineer,
	"Manager / Executive":        JobTypeManager,
	"Others":                     JobTypeOthers,
}

// EmploymentTypeIDs maps an employment-type label to its field-542 value.
var EmploymentTypeIDs = map[string]string{
	"Regular":        EmployRegular,
	"Temporary":      EmployTemporary,
	"Intern":         EmployIntern,
	"Apprenticeship": EmployApprenticeship,
}
