package golol

import (
	"errors"
	"fmt"
	"github.com/TrevorSStone/goamf"
)

type Summoner struct {
	IconId       int
	Level        int
	ID           int
	InternalName string
	AccountID    int
	Name         string
}

func unmarshalSummoner(response amf.Object) (summoner Summoner, err error) {

	if data, ok := response["data"].(amf.TypedObject); ok {
		if body, ok := data.Object["body"].(amf.TypedObject); ok {
			if body.ObjectType != "com.riotgames.platform.summoner.PublicSummoner" {
				return summoner, errors.New("Body Object Type is not PublicSummoner")
			}
			if summoner.InternalName, ok = body.Object["internalName"].(string); !ok {
				return summoner, errors.New("InternalName missing from response")
			}
			if summoner.Name, ok = body.Object["name"].(string); !ok {
				return summoner, errors.New("Name missing from response")
			}

			if summonerID, ok := body.Object["summonerId"].(float64); ok {
				summoner.ID = int(summonerID)
			} else {
				return summoner, errors.New("summonerId missing from response")
			}
			if level, ok := body.Object["summonerLevel"].(float64); ok {
				summoner.Level = int(level)
			} else {
				return summoner, errors.New("summonerLevel missing from response")
			}
			if accountID, ok := body.Object["acctId"].(float64); ok {
				summoner.AccountID = int(accountID)
			} else {
				return summoner, errors.New("acctId missing from response")
			}
			if iconID, ok := body.Object["profileIconId"].(uint32); ok {
				summoner.IconId = int(iconID)
			} else {
				return summoner, errors.New("profileIconId missing from response")
			}
			return summoner, err

		}
	}
	return summoner, errors.New(fmt.Sprintf("Response Data Not Acceptable %v", response))
}
