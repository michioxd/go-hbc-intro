# go-hbc-intro

The Homebrew Channel banner built with Go and ebiten. I made this for fun and to learn Go.

It is a simple implementation of the Homebrew Channel banner, which is a homebrew application for the Wii console. The banner is displayed when the application is launched and provides information about the application.

![image](https://github.com/user-attachments/assets/fb293fd1-0fdd-427f-956d-091072e7bd17)

## Building

```bash
git clone https://github.com/michioxd/go-hbc-intro.git
cd go-hbc-intro
go build -ldflags="-H windowsgui" -o ghi.exe main.go
```

## Running

```bash
ghi.exe
```

## License

All [assets](/assets/) are licensed under the GNU General Public License. See from [`hbc` respository](https://github.com/fail0verflow/hbc/tree/master?tab=GPL-2.0-1-ov-file#readme) for more information.

Go code is licensed under the MIT License. See [LICENSE](LICENSE) for more information.
