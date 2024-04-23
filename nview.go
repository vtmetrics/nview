package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type VTuber struct {
	Name string `json:"name"`
	// "Indie" for unaffiliated
	Affiliation string `json:"affiliation"`
	CCV         int    `json:"ccv"`
	NView       int    `json:"n_view"`
	vtStatsID   VtStatsVtuberID
}

func (vt *VTuber) String() string {
	return fmt.Sprintf("%v (%v) is a %vview (with %v CCV)", vt.Name, vt.Affiliation, vt.NView, vt.CCV)
}

func main() {
	// make --help print the usage
	flag.Usage = func() {
		fmt.Println("Usage: nview -name <name> [-output <output>] [-log <log>]")
		flag.PrintDefaults()
	}
	var (
		name         string
		outputFormat string
		logLevel     string
	)
	flag.StringVar(&name, "name", "", "Name of the VTuber")
	flag.StringVar(&name, "n", "", "Alias for -name")
	flag.StringVar(&outputFormat, "output", "text", "Output format (text, number, json)")
	flag.StringVar(&outputFormat, "o", "text", "Alias for -output")
	flag.StringVar(&logLevel, "log", "warn", "Log level (debug, info, warn, error)")
	flag.StringVar(&logLevel, "l", "warn", "alias for -log")
	flag.Parse()
	if name == "" {
		fmt.Println("Please provide a name")
		return
	}
	switch logLevel {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		fmt.Println("Invalid log level")
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}
	slog.Info("Fetching catalog...")
	catalog, err := fetchCatalog()
	if err != nil {
		slog.Error("failed to fetch catalog", "error", err)
		return
	}
	slog.Info("Catalog fetched")
	vtuber, err := NewVTuber(name, catalog)
	if err != nil {
		slog.Error("failed to fetch data", "error", err)
		return
	}

	// Update the CCV and NView
	switch outputFormat {
	case "text":
		fmt.Println(vtuber)
	case "json":
		b, err := json.Marshal(vtuber)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	case "number":
		fmt.Println(vtuber.NView)
	default:
		fmt.Println("Invalid output format")
	}
}

type VTuberView interface {
	updateCCV() (int, error)
	updateNView() int
}

type VTuberProfile interface {
	updateProfile(s string)
}

type VtStatsVtuberInfoItem struct {
	VtuberID        string `json:"vtuberId"`
	NativeName      string `json:"nativeName"`
	EnglishName     string `json:"englishName"`
	JapaneseName    string `json:"japaneseName"`
	ThumbnailUrl    string `json:"thumbnailUrl"`
	TwitterUsername string `json:"twitterUsername"`
	DebuttedAt      int    `json:"debuttedAt"`
	RetiredAt       int    `json:"retiredAt"`
}

type VtStatsChannelInfoItem struct {
	ChannelID  json.Number `json:"channelId"`
	PlatformID string      `json:"platformId"`
	VtuberID   string      `json:"vtuberId"`
	Platform   string      `json:"platform"`
}

type VtStatsGroupInfoItem struct {
	GroupID      string   `json:"groupId"`
	IsRoot       bool     `json:"root"`
	NativeName   string   `json:"nativeName"`
	EnglishName  string   `json:"englishName"`
	JapaneseName string   `json:"japaneseName"`
	Children     []string `json:"children"`
}

type VtStatsCatalog struct {
	Vtubers  []VtStatsVtuberInfoItem  `json:"vtubers"`
	Channels []VtStatsChannelInfoItem `json:"channels"`
	Groups   []VtStatsGroupInfoItem   `json:"groups"`
}

type VtStatsStreamInfoItem struct {
	Platform     string      `json:"platform"`
	PlatformID   string      `json:"platformId"`
	StreamID     string      `json:"streamId"`
	ChannelID    json.Number `json:"channelId"`
	Title        string      `json:"title"`
	ThumbnailUrl string      `json:"thumbnailUrl"`
	ScheduleTime int         `json:"scheduleTime"`
	StartTime    int         `json:"startTime"`
	EndTime      int         `json:"endTime"`
	ViewerAvg    int         `json:"viewerAvg"`
	ViewerMax    int         `json:"viewerMax"`
	LikeMax      int         `json:"likeMax"`
	UpdatedAt    int         `json:"updatedAt"`
	Status       string      `json:"status"`
}

type VtStatsVtuberID struct {
	VtuberID   string   `json:"vtuberId"`
	ChannelIDs []string `json:"channelId"`
	GroupID    string   `json:"groupId"`
}

func fetchCatalog() (VtStatsCatalog, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "vt-api.poi.cat",
		Path:   "api/v4/catalog",
	}
	start := time.Now()
	res, err := http.Get(u.String())
	if err != nil {
		return VtStatsCatalog{}, err
	}
	// Measure the time it takes to fetch the data
	defer res.Body.Close()
	end := time.Now()
	slog.Debug("Completed fetching catalog", "duration", end.Sub(start))
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return VtStatsCatalog{}, err
	}
	catalog := VtStatsCatalog{}
	json.Unmarshal(data, &catalog)
	return catalog, nil
}

func (vt *VTuber) updateNView() int {
	vt.NView = computeNView(vt.CCV)
	return vt.NView
}

func (vt *VTuber) updateCCV() (int, error) {
	ccv, err := computeCCV(vt.vtStatsID.ChannelIDs)
	if err != nil {
		slog.Error("Error while updating CCV: %v", err)
		return -1, err
	}
	vt.CCV = ccv
	return vt.CCV, nil
}

func computeCCV(channelIDs []string) (int, error) {
	v := url.Values{}
	v.Set("channelIds", strings.Join(channelIDs, ","))
	u := url.URL{
		Scheme:   "https",
		Host:     "vt-api.poi.cat",
		Path:     "api/v4/streams/ended",
		RawQuery: v.Encode(),
	}
	slog.Debug("Fetching data", "url", u.String())
	start := time.Now()
	res, err := http.Get(u.String())
	if err != nil {
		slog.Error("Error while fetching data: %v", err)
		return -1, err
	}
	// Measure the time it takes to fetch the data
	defer res.Body.Close()
	end := time.Now()
	slog.Debug("Completed fetching data", "duration", end.Sub(start))
	data, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Error while reading data: %v", err)
		return -1, err
	}
	streamItems := make([]VtStatsStreamInfoItem, 0)
	json.Unmarshal(data, &streamItems)
	ccv := 0
	for _, item := range streamItems {
		ccv += item.ViewerAvg
	}
	if len(streamItems) > 0 {
		ccv /= len(streamItems)
	}
	return ccv, nil
}

func computeNView(ccv int) int {
	if ccv == 0 {
		return 0
	}
	return int(math.Log10(float64(ccv))) + 1
}

func (ct VtStatsCatalog) getIDInfo(name string) (VtStatsVtuberID, error) {
	vtuberID := ""
	for _, vtuber := range ct.Vtubers {
		vtName := vtuber.EnglishName
		if vtName == "" {
			vtName = vtuber.NativeName
		}
		if vtName == name {
			vtuberID = vtuber.VtuberID
			break
		}
	}
	if vtuberID == "" {
		return VtStatsVtuberID{}, fmt.Errorf("vtuber not found: %v", name)
	}

	// Get the channel id
	channelIDs := make([]string, 0)
	for _, channel := range ct.Channels {
		if channel.VtuberID != "" && channel.VtuberID == vtuberID {
			channelIDs = append(channelIDs, string(channel.ChannelID))
		}
	}
	if channelIDs == nil {
		fmt.Printf("Channel not found: %v", name)
		return VtStatsVtuberID{}, fmt.Errorf("channel not found: %v", name)
	}

	groupID := ""
	for _, group := range ct.Groups {
		for _, vtuber := range group.Children {
			if vtuber == fmt.Sprintf("vtuber:%v", vtuberID) {
				groupID = group.GroupID
				break
			}
		}
		if groupID != "" {
			break
		}
	}
	if groupID == "" {
		fmt.Println("Group not found")
		groupID = "indie"
	}

	return VtStatsVtuberID{
		VtuberID:   vtuberID,
		ChannelIDs: channelIDs,
		GroupID:    groupID,
	}, nil
}

func (ct VtStatsCatalog) getAffiliation(groupID string) string {
	if groupID == "others" {
		return "Indie"
	}
	for _, group := range ct.Groups {
		if group.GroupID == groupID {
			if group.EnglishName == "" {
				return group.NativeName
			} else {
				return group.EnglishName
			}
		}
	}
	return "Indie"
}

func NewVTuber(name string, catalog VtStatsCatalog) (*VTuber, error) {
	idInfo, err := catalog.getIDInfo(name)
	if err != nil {
		return nil, err
	}
	affiliation := catalog.getAffiliation(idInfo.GroupID)
	ccv, err := computeCCV(idInfo.ChannelIDs)
	if err != nil {
		return nil, err
	}
	nView := computeNView(ccv)
	return &VTuber{
		Name:        name,
		Affiliation: affiliation,
		CCV:         ccv,
		NView:       nView,
		vtStatsID:   idInfo,
	}, nil
}
