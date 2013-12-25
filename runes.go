package golol

import (
	"errors"
	"fmt"
	"github.com/TrevorSStone/goamf"
)

type RunePage struct {
	Name      string
	Current   bool
	ID        int
	RuneSlots []RuneSlot
}

type RuneSlot struct {
	ID       int
	MinLevel int
	TypeID   int
	Color    string
	Rune
}

type Rune struct {
	ID          int
	Tier        int
	Description string
	Name        string
}

const (
	RedID    = 1
	YellowID = 3
	BlueID   = 5
	QuintID  = 7
)

func unmarshalRunePages(response amf.Object) (runePages []RunePage, err error) {

	if data, ok := response["data"].(amf.TypedObject); ok {
		if body, ok := data.Object["body"].(amf.TypedObject); ok {
			// if body.ObjectType != "com.riotgames.platform.summoner.AllPublicSummonerDataDTO" {
			// 	return runePages, errors.New("Body Object Type is not AllPublicSummonerDataDTO")
			// }
			if spellBook, ok := body.Object["spellBook"].(amf.TypedObject); ok {
				if bookPages, ok := spellBook.Object["bookPages"].(amf.TypedObject); ok {

					if array, ok := bookPages.Object["array"].([]interface{}); ok {
						runePages = make([]RunePage, len(array))
						for k, v := range array {
							if entry, ok := v.(amf.TypedObject); ok {
								runePages[k] = unmarshalRunePage(entry)
							}
						}
						return runePages, nil
					}

				} else {
					return runePages, errors.New("bookPages missing from response")
				}
			} else {
				return runePages, errors.New("Spellbook missing from response")

			}

		}
	}
	return runePages, errors.New(fmt.Sprintf("Response Data Not Acceptable %v", response))
}

func unmarshalRunePage(response amf.TypedObject) (runePage RunePage) {
	var ok bool
	if runePage.Name, ok = response.Object["name"].(string); !ok {
		fmt.Println("RunePage Name Missing")
	}
	if runePage.Current, ok = response.Object["current"].(bool); !ok {
		fmt.Println("RunePage Current Missing")
	}
	if pageID, ok := response.Object["pageId"].(float64); ok {
		runePage.ID = int(pageID)
	} else {
		fmt.Println("RunePage PageID Missing")
	}
	if slotEntries, ok := response.Object["slotEntries"].(amf.TypedObject); ok {
		if array, ok := slotEntries.Object["array"].([]interface{}); ok {
			runePage.RuneSlots = make([]RuneSlot, len(array))
			for k, v := range array {
				if entry, ok := v.(amf.TypedObject); ok {
					if runeslotentry, ok := entry.Object["runeSlot"].(amf.TypedObject); ok {
						runePage.RuneSlots[k] = unmarshalRuneSlot(runeslotentry)
						if runeentry, ok := entry.Object["rune"].(amf.TypedObject); ok {
							runePage.RuneSlots[k].Rune = unmarshalRune(runeentry)
						}
					}

				}
			}
		}
	}
	return
}

func unmarshalRuneSlot(response amf.TypedObject) (runeSlot RuneSlot) {
	if slotid, ok := response.Object["id"].(uint32); ok {
		runeSlot.ID = int(slotid)
	} else {
		fmt.Println("Rune Slot ID not found")
	}
	if minlevel, ok := response.Object["minLevel"].(uint32); ok {
		runeSlot.MinLevel = int(minlevel)
	} else {
		fmt.Println("Rune Slot MinLevel not found")
	}

	if runeType, ok := response.Object["runeType"].(amf.TypedObject); ok {
		if runetypeid, ok := runeType.Object["runeTypeId"].(uint32); ok {
			runeSlot.TypeID = int(runetypeid)
		} else {
			fmt.Println("Rune Slot runeTypeId not found")
		}
		if runeSlot.Color, ok = runeType.Object["name"].(string); !ok {
			fmt.Println("Rune Slot name not found")
		}
	}
	return
}

func unmarshalRune(response amf.TypedObject) (newRune Rune) {
	var ok bool
	if newRune.Name, ok = response.Object["name"].(string); !ok {
		fmt.Println("Rune Name Missing")
	}
	if newRune.Description, ok = response.Object["description"].(string); !ok {
		fmt.Println("Rune description Missing")
	}
	if tier, ok := response.Object["tier"].(uint32); ok {
		newRune.Tier = int(tier)
	} else {
		fmt.Println("Rune Tier not found")
	}
	if itemID, ok := response.Object["itemId"].(uint32); ok {
		newRune.ID = int(itemID)
	} else {
		fmt.Println("Rune itemId not found")
	}
	return
}
