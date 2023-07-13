package utils

import (
	"fmt"
	"testing"
)

func TestColorCli(t *testing.T) {
	//默认的不带任何效果的字体显示
	fmt.Println(Green("字体：Green"))
	fmt.Println(LightGreen("字体：LightGreen"))
	fmt.Println(Cyan("字体：Cyan"))
	fmt.Println(LightCyan("字体：LightCyan"))
	fmt.Println(Red("字体：Red"))
	fmt.Println(LightRed("字体：LightRed"))
	fmt.Println(Yellow("字体：Yellow"))
	fmt.Println(Black("字体：Black"))
	fmt.Println(DarkGray("字体：DarkGray"))
	fmt.Println(LightGray("字体：LightGray"))
	fmt.Println(White("字体：White"))
	fmt.Println(Blue("字体：Blue"))
	fmt.Println(LightBlue("字体：LightBlue"))
	fmt.Println(Purple("字体：Purple"))
	fmt.Println(LightPurple("字体：LightPurple"))
	fmt.Println(Brown("字体：Brown"))
	fmt.Println(Blue("字体：Blue", 1, 1))

	//带闪烁效果的彩色字体显示
	fmt.Println(Green("闪烁字体：Green", 1, 1))
	fmt.Println(LightGreen("闪烁字体：LightGreen", 1))
	fmt.Println(Cyan("闪烁字体：Cyan", 1))
	fmt.Println(LightCyan("闪烁字体：LightCyan", 1))
	fmt.Println(Red("闪烁字体：Red", 1))
	fmt.Println(LightRed("闪烁字体：LightRed", 1))
	fmt.Println(Yellow("闪烁字体：Yellow", 1))
	fmt.Println(Black("闪烁字体：Black", 1))
	fmt.Println(DarkGray("闪烁字体：DarkGray", 1))
	fmt.Println(LightGray("闪烁字体：LightGray", 1))
	fmt.Println(White("闪烁字体：White", 1))
	fmt.Println(Blue("闪烁字体：Blue", 1))
	fmt.Println(LightBlue("闪烁字体：LightBlue", 1))
	fmt.Println(Purple("闪烁字体：Purple", 1))
	fmt.Println(LightPurple("闪烁字体：LightPurple", 1))
	fmt.Println(Brown("闪烁字体：Brown", 1))

	//带下划线效果的字体显示
	fmt.Println(Green("闪烁且带下划线字体：Green", 1, 1, 1))
	fmt.Println(LightGreen("闪烁且带下划线字体：LightGreen", 1, 1))
	fmt.Println(Cyan("闪烁且带下划线字体：Cyan", 1, 1))
	fmt.Println(LightCyan("闪烁且带下划线字体：LightCyan", 1, 1))
	fmt.Println(Red("闪烁且带下划线字体：Red", 1, 1))
	fmt.Println(LightRed("闪烁且带下划线字体：LightRed", 1, 1))
	fmt.Println(Yellow("闪烁且带下划线字体：Yellow", 1, 1))
	fmt.Println(Black("闪烁且带下划线字体：Black", 1, 1))
	fmt.Println(DarkGray("闪烁且带下划线字体：DarkGray", 1, 1))
	fmt.Println(LightGray("闪烁且带下划线字体：LightGray", 1, 1))
	fmt.Println(White("闪烁且带下划线字体：White", 1, 1))
	fmt.Println(Blue("闪烁且带下划线字体：Blue", 1, 1))
	fmt.Println(LightBlue("闪烁且带下划线字体：LightBlue", 1, 1))
	fmt.Println(Purple("闪烁且带下划线字体：Purple", 1, 1))
	fmt.Println(LightPurple("闪烁且带下划线字体：LightPurple", 1, 1))
	fmt.Println(Brown("闪烁且带下划线字体：Brown", 1, 1))

}
