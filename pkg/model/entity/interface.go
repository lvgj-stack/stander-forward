package entity

type PlanPeriod int32

const (
	Month PlanPeriod = iota
	Quarter
	HalfYear
	Year
)
