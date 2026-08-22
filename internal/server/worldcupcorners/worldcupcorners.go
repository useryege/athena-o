package worldcupcorners

import (
	"context"

	worldcupcornerspkg "github.com/useryege/athena/pkg/apiclient/worldcupcorners"
)

type Server struct {
	worldcupcornerspkg.UnimplementedWorldCupCornersServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) GetWorldCupCornersDataset(_ context.Context, _ *worldcupcornerspkg.GetWorldCupCornersDatasetRequest) (*worldcupcornerspkg.GetWorldCupCornersDatasetResponse, error) {
	return &worldcupcornerspkg.GetWorldCupCornersDatasetResponse{
		Stages:  worldCupCornerStages,
		Matches: worldCupCornerMatches,
	}, nil
}

var worldCupCornerStages = []*worldcupcornerspkg.WorldCupCornerStage{
	{Key: "group-stage", Label: "Group Stage (32 → 16)", ShortLabel: "Group Stage", Knockout: false},
	{Key: "round-of-16", Label: "Round of 16", ShortLabel: "Round of 16", Knockout: true},
	{Key: "quarter-finals", Label: "Quarter-finals", ShortLabel: "Quarter-finals", Knockout: true},
	{Key: "semi-finals", Label: "Semi-finals", ShortLabel: "Semi-finals", Knockout: true},
	{Key: "third-place", Label: "Third-place Match", ShortLabel: "Third-place", Knockout: true},
	{Key: "final", Label: "Final", ShortLabel: "Final", Knockout: true},
}

var worldCupCornerMatches = []*worldcupcornerspkg.WorldCupCornerMatch{
	newMatch(1, "group-stage", "Qatar", "Ecuador", 0, 2, 1, 3),
	newMatch(2, "group-stage", "England", "Iran", 6, 2, 8, 0),
	newMatch(3, "group-stage", "Senegal", "Netherlands", 0, 2, 6, 7),
	newMatch(4, "group-stage", "United States", "Wales", 1, 1, 5, 3),
	newMatch(5, "group-stage", "Argentina", "Saudi Arabia", 1, 2, 9, 2),
	newMatch(6, "group-stage", "Denmark", "Tunisia", 0, 0, 11, 9),
	newMatch(7, "group-stage", "Mexico", "Poland", 0, 0, 6, 5),
	newMatch(8, "group-stage", "France", "Australia", 4, 1, 8, 1),
	newMatch(9, "group-stage", "Morocco", "Croatia", 0, 0, 0, 5),
	newMatch(10, "group-stage", "Germany", "Japan", 1, 2, 6, 6),
	newMatch(11, "group-stage", "Spain", "Costa Rica", 7, 0, 5, 0),
	newMatch(12, "group-stage", "Belgium", "Canada", 1, 0, 4, 4),
	newMatch(13, "group-stage", "Switzerland", "Cameroon", 1, 0, 11, 5),
	newMatch(14, "group-stage", "Uruguay", "South Korea", 0, 0, 4, 3),
	newMatch(15, "group-stage", "Portugal", "Ghana", 3, 2, 3, 3),
	newMatch(16, "group-stage", "Brazil", "Serbia", 2, 0, 5, 4),
	newMatch(17, "group-stage", "Wales", "Iran", 0, 2, 2, 7),
	newMatch(18, "group-stage", "Qatar", "Senegal", 1, 3, 6, 6),
	newMatch(19, "group-stage", "Netherlands", "Ecuador", 1, 1, 2, 5),
	newMatch(20, "group-stage", "England", "United States", 0, 0, 3, 7),
	newMatch(21, "group-stage", "Tunisia", "Australia", 0, 1, 5, 2),
	newMatch(22, "group-stage", "Poland", "Saudi Arabia", 2, 0, 4, 5),
	newMatch(23, "group-stage", "France", "Denmark", 2, 1, 6, 4),
	newMatch(24, "group-stage", "Argentina", "Mexico", 2, 0, 4, 2),
	newMatch(25, "group-stage", "Japan", "Costa Rica", 0, 1, 5, 0),
	newMatch(26, "group-stage", "Belgium", "Morocco", 0, 2, 9, 1),
	newMatch(27, "group-stage", "Croatia", "Canada", 4, 1, 5, 2),
	newMatch(28, "group-stage", "Spain", "Germany", 1, 1, 6, 5),
	newMatch(29, "group-stage", "Cameroon", "Serbia", 3, 3, 4, 3),
	newMatch(30, "group-stage", "South Korea", "Ghana", 2, 3, 12, 5),
	newMatch(31, "group-stage", "Brazil", "Switzerland", 1, 0, 8, 3),
	newMatch(32, "group-stage", "Portugal", "Uruguay", 2, 0, 6, 2),
	newMatch(33, "group-stage", "Netherlands", "Qatar", 2, 0, 4, 2),
	newMatch(34, "group-stage", "Ecuador", "Senegal", 1, 2, 3, 6),
	newMatch(35, "group-stage", "Iran", "United States", 0, 1, 1, 5),
	newMatch(36, "group-stage", "Wales", "England", 0, 3, 1, 6),
	newMatch(37, "group-stage", "Tunisia", "France", 1, 0, 7, 8),
	newMatch(38, "group-stage", "Australia", "Denmark", 1, 0, 2, 6),
	newMatch(39, "group-stage", "Saudi Arabia", "Mexico", 1, 2, 1, 8),
	newMatch(40, "group-stage", "Poland", "Argentina", 0, 2, 1, 8),
	newMatch(41, "group-stage", "Canada", "Morocco", 1, 2, 6, 2),
	newMatch(42, "group-stage", "Croatia", "Belgium", 0, 0, 2, 4),
	newMatch(43, "group-stage", "Japan", "Spain", 2, 1, 0, 2),
	newMatch(44, "group-stage", "Costa Rica", "Germany", 2, 4, 1, 14),
	newMatch(45, "group-stage", "Ghana", "Uruguay", 0, 2, 5, 2),
	newMatch(46, "group-stage", "South Korea", "Portugal", 2, 1, 5, 4),
	newMatch(47, "group-stage", "Serbia", "Switzerland", 2, 3, 2, 0),
	newMatch(48, "group-stage", "Cameroon", "Brazil", 1, 0, 3, 11),
	newMatch(49, "round-of-16", "Netherlands", "United States", 3, 1, 4, 5),
	newMatch(50, "round-of-16", "Argentina", "Australia", 2, 1, 1, 3),
	newMatch(51, "round-of-16", "France", "Poland", 3, 1, 7, 1),
	newMatch(52, "round-of-16", "England", "Senegal", 3, 0, 3, 3),
	newMatch(53, "round-of-16", "Japan", "Croatia", 1, 1, 5, 4, 8, 5, 1, 3),
	newMatch(54, "round-of-16", "Brazil", "South Korea", 4, 1, 5, 4),
	newMatch(55, "round-of-16", "Morocco", "Spain", 0, 0, 0, 4, 0, 8, 3, 0),
	newMatch(56, "round-of-16", "Portugal", "Switzerland", 6, 1, 6, 6),
	newMatch(57, "quarter-finals", "Croatia", "Brazil", 1, 1, 2, 5, 3, 7, 4, 2),
	newMatch(58, "quarter-finals", "Netherlands", "Argentina", 2, 2, 2, 1, 2, 8, 3, 4),
	newMatch(59, "quarter-finals", "Morocco", "Portugal", 1, 0, 3, 9),
	newMatch(60, "quarter-finals", "England", "France", 1, 2, 5, 2),
	newMatch(61, "semi-finals", "Argentina", "Croatia", 3, 0, 2, 4),
	newMatch(62, "semi-finals", "France", "Morocco", 2, 0, 2, 3),
	newMatch(63, "third-place", "Croatia", "Morocco", 2, 1, 6, 3),
	newMatch(64, "final", "Argentina", "France", 3, 3, 4, 3, 6, 5, 4, 2),
}

func newMatch(id int32, stage, homeTeam, awayTeam string, homeScore, awayScore, homeCorners90, awayCorners90 int32, extraTimeAndPenalties ...int32) *worldcupcornerspkg.WorldCupCornerMatch {
	match := &worldcupcornerspkg.WorldCupCornerMatch{
		Id:              id,
		Stage:           stage,
		HomeTeam:        homeTeam,
		AwayTeam:        awayTeam,
		HomeScore:       homeScore,
		AwayScore:       awayScore,
		HomeCorners90:   homeCorners90,
		AwayCorners90:   awayCorners90,
		HomeCornersFull: homeCorners90,
		AwayCornersFull: awayCorners90,
	}
	if len(extraTimeAndPenalties) == 4 {
		match.HomeCornersFull = extraTimeAndPenalties[0]
		match.AwayCornersFull = extraTimeAndPenalties[1]
		match.HasPenaltyShootout = true
		match.HomePenaltyScore = extraTimeAndPenalties[2]
		match.AwayPenaltyScore = extraTimeAndPenalties[3]
	}
	return match
}
