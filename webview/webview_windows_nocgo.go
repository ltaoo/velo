//go:build windows && !cgo

package webview

import "fmt"

func open_webview(opts *BoxWebviewOptions) {
	fmt.Println("Webview (WebView2) requires CGO; building without UI on Windows.")
}

func Terminate()                       {}
func setTitle(title string)            {}
func setSize(width, height int)        {}
func setMinSize(width, height int)     {}
func setMaxSize(width, height int)     {}
func setPosition(x, y int)            {}
func getPosition() (int, int)          { return 0, 0 }
func getSize() (int, int)              { return 0, 0 }
func show()                            {}
func hide()                            {}
func minimize()                        {}
func maximize()                        {}
func fullscreen()                      {}
func unFullscreen()                    {}
func restore()                         {}
func setAlwaysOnTop(onTop bool)        {}
func setURL(url string)                {}
func close_webview()                   {}
