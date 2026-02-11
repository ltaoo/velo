//go:build !darwin && !windows

package webview

import "fmt"

func open_webview(opts *BoxWebviewOptions) {
	fmt.Println("Webview is not supported on this platform yet.")
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
