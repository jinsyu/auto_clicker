# Auto Clicker

정확한 시간에 자동으로 클릭해주는 프로그램입니다.

## 기능
- NTP 서버를 통한 정확한 시간 동기화
- 밀리초 단위의 정확한 타이밍
- 간단한 GUI 인터페이스
- 실시간 남은 시간 표시

## 다운로드
[Releases](https://github.com/jinsyu/auto_clicker/releases) 페이지에서 운영체제에 맞는 버전을 다운로드하세요.

## 사용 방법
1. 프로그램을 실행합니다.
2. 원하는 시간을 24시간 형식으로 입력합니다. (예: 14:30:00)
3. 클릭하고자 하는 위치에 마우스 커서를 놓습니다.
4. "시작" 버튼을 클릭합니다.
5. 설정한 시간이 되면 자동으로 클릭이 실행됩니다.

## 빌드하기

### 요구사항
- Go 1.21 이상
- gcc 컴파일러

### Windows
```
bash
go mod tidy
go build -o autoclicker.exe
```

### Linux
```
bash
sudo apt-get install gcc libc6-dev libx11-dev xorg-dev libxtst-dev libpng++-dev
sudo apt-get install xcb libxcb-xkb-dev x11-xkb-utils libx11-xcb-dev libxkbcommon-x11-dev libxkbcommon-dev
go mod tidy
go build -o autoclicker
```

### macOS
```
bash
brew install pkg-config gcc
go mod tidy
go build -o autoclicker
```

## 이제 새 버전을 릴리즈하려면:
### 1. 변경사항을 커밋합니다
```
bash
git add .
git commit -m "Initial commit"
```

### d2. 태그를 생성하고 푸시합니다
```
bash
git tag v1.0.0
git push origin main --tags
```
