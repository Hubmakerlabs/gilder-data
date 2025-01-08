package locale

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

type TimeZone struct {
	Name      st    `json:"name"`
	Offset    int   `json:"offset"`
	DstName   st    `json:"dst_name,omitempty"`
	DstOffset int   `json:"dst_offset,omitempty"`
	DstStart  int64 `json:"dst_start,omitempty"`
	DstEnd    int64 `json:"dst_end,omitempty"`
}

type TimeZones map[st]TimeZone

func (t TimeZones) GetSortedNames() (s []st) {
	for i := range t {
		s = append(s, i)
	}
	sort.Strings(s)
	return
}

type Nation struct {
	Id             no   `json:"id"`
	Name           st   `json:"name"`
	Iso3           st   `json:"iso3"`
	Iso2           st   `json:"iso2"`
	NumericCode    st   `json:"numeric_code"`
	PhoneCode      st   `json:"phone_code"`
	Capital        st   `json:"capital"`
	Currency       st   `json:"currency"`
	CurrencyName   st   `json:"currency_name"`
	CurrencySymbol st   `json:"currency_symbol"`
	Tld            st   `json:"tld"`
	Native         st   `json:"native"`
	Emoji          st   `json:"emoji"`
	Languages      []st `json:"languages"`
	TimeZones      `json:"timezones"`
}

const (
	countries          = "https://github.com/dr5hn/countries-states-cities-database/raw/refs/heads/master/json/countries+states+cities.json"
	availableTimeZones = "https://www.timeapi.io/api/timezone/availabletimezones"
	getZoneInfo        = "https://www.timeapi.io/api/timezone/zone?timeZone="
	JsonFile           = "countriestimezoneslanguages.json"
)

func GetData(jsonFile st) (o st) {
	if jsonFile == "" {
		jsonFile = JsonFile
	}
	var err er
	var fi os.FileInfo
	if fi, err = os.Stat(jsonFile); err == nil {
		modTime := fi.ModTime()
		update := time.Now().Add(time.Hour * 24 * 90)
		if !modTime.After(update) {
			// no need to update more than once a season
			var b by
			if b, err = os.ReadFile(JsonFile); !chk.E(err) {
				o = st(b)
				return
			}
		}
	}
	var c Countries
	var res *http.Response
	// first get the available timezones
	if res, err = http.Get(availableTimeZones); chk.E(err) {
		return
	}
	var b by
	if b, err = io.ReadAll(res.Body); chk.E(err) {
		return
	}
	tzs := make(TimeZones)
	var zoneNames []st
	if err = json.Unmarshal(b, &zoneNames); chk.E(err) {
		return
	}
	// next get the details for each named timezone
	for _, v := range zoneNames {
		if res, err = http.Get(getZoneInfo + v); chk.E(err) {
			return
		}
		if b, err = io.ReadAll(res.Body); chk.E(err) {
			return
		}
		var z timeZone
		if err = json.Unmarshal(b, &z); chk.E(err) {
			return
		}
		if z.TimeZone == "" {
			continue
		}
		zi := TimeZone{
			Offset: z.StandardUtcOffset.Seconds,
		}
		if z.HasDayLightSaving {
			zi.DstName = z.DstInterval.DstName
			zi.DstOffset = z.DstInterval.DstOffsetToUtc.Seconds
			zi.DstStart = z.DstInterval.DstStart.Unix()
			zi.DstEnd = z.DstInterval.DstEnd.Unix()
		}
		tzs[z.TimeZone] = zi
	}
	// next, get the country data
	if res, err = http.Get(countries); chk.E(err) {
		return
	}
	if b, err = io.ReadAll(res.Body); chk.E(err) {
		return
	}
	if err = json.Unmarshal(b, &c); chk.E(err) {
		return
	}
	// condense the data down to the essentials we require, and link the time zones
	// to the ones we gathered before (as many overlap over the same longitude).
	var cc []Nation
	for _, country := range c {
		ccc := Nation{
			Id:             country.Id,
			Name:           country.Name,
			Iso3:           country.Iso3,
			Iso2:           country.Iso2,
			NumericCode:    country.NumericCode,
			PhoneCode:      country.PhoneCode,
			Capital:        country.Capital,
			Currency:       country.Currency,
			CurrencyName:   country.CurrencyName,
			CurrencySymbol: country.CurrencySymbol,
			Tld:            country.Tld,
			Native:         country.Native,
			Emoji:          country.Emoji,
			TimeZones:      make(TimeZones),
		}
		lang := GetLanguageByIso2(country.Iso2)
		if lang != nil {
			ccc.Languages = (*lang).Languages
		} else {
			// default to english if we don't know it from our language db
			ccc.Languages = append(ccc.Languages, "English")
		}
		for _, tz := range country.Tzs {
			var ok bo
			if ccc.TimeZones[tz.ZoneName], ok = tzs[tz.ZoneName]; ok {
				x := ccc.TimeZones[tz.ZoneName]
				x.Name = tz.Abbreviation
				ccc.TimeZones[tz.ZoneName] = x
			}
		}
		cc = append(cc, ccc)
	}
	if b, err = json.Marshal(&cc); chk.E(err) {
		return
	}
	// cache the current version so we can avoid making it again any time too soon
	chk.E(os.WriteFile(jsonFile, b, 0660))
	return fmt.Sprintf("%s\n", b)
}
