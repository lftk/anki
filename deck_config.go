package anki

import (
	"iter"
	"time"

	"github.com/lftk/anki/internal/pb"
	"google.golang.org/protobuf/proto"
)

type DeckConfig struct {
	ID       int64
	Name     string
	Modified time.Time
	USN      int
	Config   *pb.DeckConfig
}

var DefaultDeckConfig = &pb.DeckConfig{
	LearnSteps:                      nil,
	RelearnSteps:                    nil,
	NewPerDay:                       20,
	ReviewsPerDay:                   200,
	NewPerDayMinimum:                0,
	InitialEase:                     2.5,
	EasyMultiplier:                  1.3,
	HardMultiplier:                  1.2,
	LapseMultiplier:                 0.0,
	IntervalMultiplier:              1.0,
	MaximumReviewInterval:           36500,
	MinimumLapseInterval:            1,
	GraduatingIntervalGood:          1,
	GraduatingIntervalEasy:          4,
	NewCardInsertOrder:              pb.DeckConfig_NEW_CARD_INSERT_ORDER_DUE,
	NewCardGatherPriority:           pb.DeckConfig_NEW_CARD_GATHER_PRIORITY_DECK,
	NewCardSortOrder:                pb.DeckConfig_NEW_CARD_SORT_ORDER_TEMPLATE,
	ReviewOrder:                     pb.DeckConfig_REVIEW_CARD_ORDER_DAY,
	NewMix:                          pb.DeckConfig_REVIEW_MIX_MIX_WITH_REVIEWS,
	InterdayLearningMix:             pb.DeckConfig_REVIEW_MIX_MIX_WITH_REVIEWS,
	LeechAction:                     pb.DeckConfig_LEECH_ACTION_TAG_ONLY,
	LeechThreshold:                  8,
	DisableAutoplay:                 false,
	CapAnswerTimeToSecs:             60,
	ShowTimer:                       false,
	StopTimerOnAnswer:               false,
	SecondsToShowQuestion:           0.0,
	SecondsToShowAnswer:             0.0,
	QuestionAction:                  pb.DeckConfig_QUESTION_ACTION_SHOW_ANSWER,
	AnswerAction:                    pb.DeckConfig_ANSWER_ACTION_BURY_CARD,
	WaitForAudio:                    true,
	SkipQuestionWhenReplayingAnswer: false,
	BuryNew:                         false,
	BuryReviews:                     false,
	BuryInterdayLearning:            false,
	FsrsParams_4:                    nil,
	FsrsParams_5:                    nil,
	FsrsParams_6:                    nil,
	DesiredRetention:                0.9,
	Other:                           nil,
	HistoricalRetention:             0.9,
	ParamSearch:                     "",
	IgnoreRevlogsBeforeDate:         "",
	EasyDaysPercentages:             nil,
}

func (c *Collection) AddDeckConfig(config *DeckConfig) error {
	id := config.ID
	if id == 0 {
		id = time.Now().UnixMilli()
	}

	inner, err := proto.Marshal(config.Config)
	if err != nil {
		return err
	}

	args := []any{
		id,
		config.Name,
		config.USN,
		timeUnix(config.Modified),
		inner,
	}
	id, err = sqlInsert(c.db, addDeckConfigQuery, args...)
	if err == nil {
		config.ID = id
	}
	return err
}

func (c *Collection) GetDeckConfig(id int64) (*DeckConfig, error) {
	return sqlGet(c.db, scanDeckConfig, getDeckConfigQuery+" WHERE id = ?", id)
}

func (c *Collection) DeleteDeckConfig(id int64) error {
	return sqlExecute(c.db, deleteDeckConfigQuery, id)
}

type ListDeckConfigsOptions struct{}

func (c *Collection) ListDeckConfigs(*ListDeckConfigsOptions) iter.Seq2[*DeckConfig, error] {
	return sqlSelectSeq(c.db, scanDeckConfig, getDeckConfigQuery)
}

func scanDeckConfig(_ sqlQueryer, row sqlRow) (*DeckConfig, error) {
	var c DeckConfig
	var mod int64
	var config []byte
	if err := row.Scan(&c.ID, &c.Name, &mod, &c.USN, &config); err != nil {
		return nil, err
	}
	c.Modified = time.Unix(mod, 0)
	c.Config = new(pb.DeckConfig)
	if err := proto.Unmarshal(config, c.Config); err != nil {
		return nil, err
	}
	return &c, nil
}
