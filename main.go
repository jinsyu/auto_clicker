package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/beevik/ntp"
	"github.com/go-vgo/robotgo"
)

// NTP 서버에서 시간 가져오기
func getNTPTime() time.Time {
	// 여러 NTP 서버를 시도
	ntpServers := []string{
		"time.google.com",
		"time.windows.com",
		"pool.ntp.org",
		"time.apple.com",
	}

	var ntpTime time.Time
	var err error

	for _, server := range ntpServers {
		ntpTime, err = ntp.Time(server)
		if err == nil {
			fmt.Printf("NTP 서버 %s 에서 시간 동기화 성공\n", server)
			return ntpTime
		}
		fmt.Printf("NTP 서버 %s 접속 실패: %v\n", server, err)
	}

	fmt.Println("모든 NTP 서버 접속 실패, 로컬 시간 사용")
	return time.Now()
}

// 시간 차이 계산
func getTimeDiff() time.Duration {
	ntpTime := getNTPTime()
	localTime := time.Now()
	return ntpTime.Sub(localTime)
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("Auto Clicker")

	// 시간 입력 필드들
	hourEntry := widget.NewEntry()
	minuteEntry := widget.NewEntry()
	secondEntry := widget.NewEntry()

	// 현재 시간과 남은 시간을 표시할 레이블
	currentTimeLabel := widget.NewLabel("현재 시간: --:--:--.---")
	remainingTimeLabel := widget.NewLabel("목표 시간이 설정되지 않았습니다")
	serverTimeLabel := widget.NewLabel("NTP 서버 시간 동기화 중...")

	// 목표 시간 저장 변수
	var targetTime time.Time
	var isWaiting bool
	var stopChan chan bool
	var timeDiff time.Duration

	// NTP 서버와 시간 차이 계산
	go func() {
		timeDiff = getTimeDiff()
		fmt.Printf("서버 시간과의 차이: %v\n", timeDiff)
		serverTimeLabel.SetText(fmt.Sprintf("서버 시간과의 차이: %v", timeDiff))

		// 주기적으로 시간 차이 업데이트 (5분마다)
		ticker := time.NewTicker(5 * time.Minute)
		go func() {
			for range ticker.C {
				timeDiff = getTimeDiff()
				fmt.Printf("서버 시간과의 차이 업데이트: %v\n", timeDiff)
				serverTimeLabel.SetText(fmt.Sprintf("서버 시간과의 차이: %v", timeDiff))
			}
		}()
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
			currentTime := time.Now().Add(timeDiff)
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

		now := time.Now().Add(timeDiff)
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
			for time.Now().Add(timeDiff).Before(targetTime) {
				select {
				case <-stopChan:
					return
				default:
					time.Sleep(time.Microsecond)
				}
			}

			if isWaiting {
				robotgo.Click()
				fmt.Printf("클릭 실행 시간: %v\n", time.Now().Add(timeDiff))
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
