package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/beevik/ntp"
	"github.com/go-vgo/robotgo"
)

// NTP 서버에서 시간 가져오기
func getNTPTime() (time.Time, error) {
	// 신뢰할 수 있는 NTP 서버 목록
	ntpServers := []string{
		"time.google.com",     // Google NTP
		"time.windows.com",    // Microsoft NTP
		"pool.ntp.org",        // NTP Pool Project
		"time.cloudflare.com", // Cloudflare NTP
	}

	var lastErr error
	// 각 서버에 대해 3번씩 시도
	for _, server := range ntpServers {
		for attempts := 0; attempts < 3; attempts++ {
			ntpTime, err := ntp.Time(server)
			if err == nil {
				fmt.Printf("NTP 서버 %s 에서 시간 동기화 성공\n", server)
				return ntpTime, nil
			}
			lastErr = err
			time.Sleep(time.Second) // 재시도 전 대기
		}
		fmt.Printf("NTP 서버 %s 접속 실패\n", server)
	}
	return time.Time{}, fmt.Errorf("모든 NTP 서버 접속 실패: %v", lastErr)
}

// 시간 동기화 관리자
type TimeSync struct {
	diff        time.Duration
	lastSync    time.Time
	syncMutex   sync.RWMutex
	maxDrift    time.Duration
	syncChannel chan struct{}
}

func NewTimeSync() *TimeSync {
	return &TimeSync{
		maxDrift:    time.Second * 30, // 30초 이상 차이나면 재동기화
		syncChannel: make(chan struct{}),
	}
}

func (ts *TimeSync) start() {
	go ts.syncRoutine()
}

func (ts *TimeSync) syncRoutine() {
	// 초기 동기화
	ts.sync()

	// 주기적 동기화 (1분마다 검사)
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		ts.sync()
	}
}

func (ts *TimeSync) sync() {
	ntpTime, err := getNTPTime()
	if err != nil {
		fmt.Printf("시간 동기화 실패: %v\n", err)
		return
	}

	ts.syncMutex.Lock()
	ts.diff = -time.Until(ntpTime)
	ts.lastSync = time.Now()
	ts.syncMutex.Unlock()

	fmt.Printf("시간 차이 업데이트: %v\n", ts.diff)
}

func (ts *TimeSync) getNow() time.Time {
	ts.syncMutex.RLock()
	defer ts.syncMutex.RUnlock()

	// 마지막 동기화로부터 너무 오래 지났으면 강제 동기화
	if time.Since(ts.lastSync) > ts.maxDrift {
		go ts.sync()
	}

	return time.Now().Add(ts.diff)
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("Auto Clicker")

	// 시간 동기화 관리자 초기화
	timeSync := NewTimeSync()
	timeSync.start()

	// 시간 입력 필드들
	hourEntry := widget.NewEntry()
	minuteEntry := widget.NewEntry()
	secondEntry := widget.NewEntry()

	// 초기값 설정 (현재 시간 + 5분)
	initialTime := timeSync.getNow().Add(5 * time.Minute)
	hourEntry.SetText(fmt.Sprintf("%02d", initialTime.Hour()))
	minuteEntry.SetText(fmt.Sprintf("%02d", initialTime.Minute()))
	secondEntry.SetText(fmt.Sprintf("%02d", initialTime.Second()))

	// 현재 시간과 남은 시간을 표시할 레이블
	currentTimeLabel := widget.NewLabel("현재 시간: --:--:--.---")
	remainingTimeLabel := widget.NewLabel("목표 시간이 설정되지 않았습니다")
	serverTimeLabel := widget.NewLabel("NTP 서버 시간 동기화 중...")

	// 목표 시간 저장 변수
	var targetTime time.Time
	var isWaiting bool
	var stopChan chan bool

	// 대신 TimeSync의 상태를 표시
	go func() {
		for {
			timeSync.syncMutex.RLock()
			serverTimeLabel.SetText(fmt.Sprintf("서버 시간과의 차이: %v", timeSync.diff))
			timeSync.syncMutex.RUnlock()
			time.Sleep(time.Second)
		}
	}()

	// 시작/중지 버튼
	startBtn := widget.NewButton("시작", nil)

	resetUI := func() {
		isWaiting = false
		remainingTimeLabel.SetText("목표 시간이 설정되지 않았습니다")
		startBtn.SetText("시작")
	}

	// 현재 시간 업데이트 고루틴
	go func() {
		for {
			currentTime := timeSync.getNow()
			currentTimeLabel.SetText(fmt.Sprintf("현재 시간: %02d:%02d:%02d.%03d",
				currentTime.Hour(),
				currentTime.Minute(),
				currentTime.Second(),
				currentTime.Nanosecond()/1000000))

			if isWaiting {
				remaining := targetTime.Sub(currentTime)
				if remaining > 0 {
					hours := int(remaining.Hours())
					minutes := int(remaining.Minutes()) % 60
					seconds := int(remaining.Seconds()) % 60
					milliseconds := int(remaining.Milliseconds()) % 1000
					remainingTimeLabel.SetText(fmt.Sprintf("남은 시간: %02d:%02d:%02d.%03d",
						hours, minutes, seconds, milliseconds))
				} else {
					resetUI()
				}
			}
			time.Sleep(time.Millisecond)
		}
	}()

	startBtn.OnTapped = func() {
		if isWaiting {
			if stopChan != nil {
				close(stopChan)
				stopChan = nil
			}
			resetUI()
			return
		}

		hour := parseIntSafe(hourEntry.Text)
		minute := parseIntSafe(minuteEntry.Text)
		second := parseIntSafe(secondEntry.Text)

		now := timeSync.getNow()
		targetTime = time.Date(
			now.Year(), now.Month(), now.Day(),
			hour, minute, second, 0, now.Location(),
		)

		if targetTime.Before(now) {
			targetTime = targetTime.Add(24 * time.Hour)
		}

		isWaiting = true
		startBtn.SetText("중지")
		fmt.Printf("목표 시간: %v\n", targetTime)

		stopChan = make(chan bool)
		go func() {
			for timeSync.getNow().Before(targetTime) {
				select {
				case <-stopChan:
					return
				default:
					time.Sleep(time.Microsecond)
				}
			}

			if isWaiting {
				robotgo.Click()
				fmt.Printf("클릭 실행 시간: %v\n", timeSync.getNow())
			}
			resetUI()
		}()
	}

	hourEntry.SetPlaceHolder("00")
	minuteEntry.SetPlaceHolder("00")
	secondEntry.SetPlaceHolder("00")

	timeInputs := container.NewHBox(
		widget.NewLabel("시:"),
		hourEntry,
		widget.NewLabel("분:"),
		minuteEntry,
		widget.NewLabel("초:"),
		secondEntry,
	)

	content := container.NewVBox(
		widget.NewLabel("Auto Clicker"),
		container.NewHBox(currentTimeLabel),
		container.NewHBox(remainingTimeLabel),
		container.NewHBox(serverTimeLabel),
		widget.NewLabel("클릭할 시간을 입력하세요 (24시간 형식)"),
		timeInputs,
		startBtn,
	)

	window.SetContent(content)
	window.Resize(fyne.NewSize(400, 300))
	window.ShowAndRun()
}

func parseIntSafe(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
