package billing

type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
	PlanTeam Plan = "team"
)

type PlanLimits struct {
	MaxEndpoints    int
	MaxRequestsPerEndpoint int
	RetentionDays   int
}

var Limits = map[Plan]PlanLimits{
	PlanFree: {MaxEndpoints: 1, MaxRequestsPerEndpoint: 100, RetentionDays: 1},
	PlanPro:  {MaxEndpoints: 10, MaxRequestsPerEndpoint: 10000, RetentionDays: 30},
	PlanTeam: {MaxEndpoints: 50, MaxRequestsPerEndpoint: 50000, RetentionDays: 90},
}

func GetLimits(plan Plan) PlanLimits {
	if l, ok := Limits[plan]; ok {
		return l
	}
	return Limits[PlanFree]
}
