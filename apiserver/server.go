package apiserver

import (
	"bufio"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ruraomsk/TLServer/logger"
	"github.com/ruraomsk/VPUserver/handlers"
	"github.com/ruraomsk/VPUserver/middleWare"
	"github.com/ruraomsk/VPUserver/sockets/mainScreen"
	"github.com/unrolled/secure"
	"net/http"
	"os"
	"strings"
	"time"
)

//MainServer настройка основного сервера
func MainServer(conf *ServerConf) (srvHttp *http.Server, srvHttps *http.Server) {
	mainScreenHub := mainScreen.NewMainScreenHub()
	//mainCrossHub := mainCross.NewCrossHub()
	//controlCrHub := controlCross.NewCrossHub()
	//techArmHub := techArm.NewTechArmHub()
	//alarmHub := alarm.NewAlarmHub()
	//xctrlHub := xctrl.NewXctrlHub()
	//gsHub := greenStreet.NewGSHub()
	//dcHub := dispatchControl.NewDCHub()
	//chatHub := chat.NewChatHub()
	//
	//go device.StartReadDevices()
	//go mainMapHub.Run()
	//go mainCrossHub.Run()
	//go controlCrHub.Run()
	//go techArmHub.Run()
	//go alarmHub.Run()
	//go xctrlHub.Run()
	//go gsHub.Run()
	//go chatHub.Run()
	//go dcHub.Run()

	// Создаем engine для соединений
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	setLogFile()

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(secureHandle())
	router.LoadHTMLGlob(conf.StaticPath)

	//скрипт и иконка которые должны быть доступны всем
	router.StaticFS("/static", http.Dir(conf.StaticPath))

	//заглушка страница 404
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "notFound.html", nil)
	})

	//начальная страница перенаправление  / -> /map
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/MainPage")
	})

	//основная страничка с картой
	router.GET("/MainPage", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	//сокет главного экрана
	router.GET("/MainPageW", func(c *gin.Context) {
		mainScreen.HMainScreen(c, mainScreenHub)
	})

	//------------------------------------------------------------------------------------------------------------------
	//обязательный общий путь
	mainRouter := router.Group("/user")
	mainRouter.Use(middleWare.JwtAuth())       //мидл проверки токена
	mainRouter.Use(middleWare.AccessControl()) //мидл проверки url пути

	//--------- SocketS--------------

	//управленеие (общее)
	mainRouter.GET("/:slug/manage", func(c *gin.Context) { //обработка создание и редактирования пользователя (страничка)
		c.HTML(http.StatusOK, "manage.html", nil)
	})
	mainRouter.POST("/:slug/manage", handlers.DisplayAccInfo)          //обработка создание и редактирования пользователя
	mainRouter.POST("/:slug/manage/changepw", handlers.ActChangePw)    //обработчик для изменения пароля
	mainRouter.POST("/:slug/manage/delete", handlers.ActDeleteAccount) //обработчик для удаления аккаунтов
	mainRouter.POST("/:slug/manage/add", handlers.ActAddAccount)       //обработчик для добавления аккаунтов
	mainRouter.POST("/:slug/manage/update", handlers.ActUpdateAccount) //обработчик для редактирования данных аккаунта
	mainRouter.POST("/:slug/manage/resetpw", handlers.ActResetPw)      //обработчик для сброса пароля администратором
	//------------------------------------------------------------------------------------------------------------------
	//роутер для фаил сервера, он закрыт токеном, скачивать могут только авторизированные пользователи

	fileServer := router.Group("/file")
	fileServer.Use(middleWare.JwtFile())

	fsStatic := fileServer.Group("/static")
	fsStatic.StaticFS("/static", http.Dir(conf.StaticPath+"/static"))
	//fsStatic.StaticFS("/img", http.Dir(conf.StaticPath+"/img"))
	//fsStatic.StaticFS("/markdown", http.Dir(conf.StaticPath+"/markdown"))

	//fsWeb := fileServer.Group("/web")
	//fsWeb.StaticFS("/resources", http.Dir(conf.WebPath+"/resources"))
	//fsWeb.StaticFS("/js", http.Dir(conf.WebPath+"/js"))
	//fsWeb.StaticFS("/css", http.Dir(conf.WebPath+"/css"))

	//------------------------------------------------------------------------------------------------------------------
	// Запуск HTTP сервера
	srvHttp = &http.Server{Handler: router, Addr: conf.PortHTTP, ErrorLog: logger.Warning}
	srvHttps = &http.Server{Handler: router, Addr: conf.PortHTTPS, ErrorLog: logger.Warning}
	return
}

var secureHandle = func() gin.HandlerFunc {
	return func(c *gin.Context) {
		secureMidle := secure.New(secure.Options{
			SSLRedirect: true,
		})
		err := secureMidle.Process(c.Writer, c.Request)
		if err != nil {
			return
		}
		c.Next()
	}
}

func setLogFile() {
	path := logger.LogGlobalConf.GinLogPath + "/ginLog.log"
	readF, _ := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	path2 := logger.LogGlobalConf.GinLogPath + "/ginLogW.log"
	writeF, _ := os.OpenFile(path2, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	scanner := bufio.NewScanner(readF)
	writer := bufio.NewWriter(writeF)
	for scanner.Scan() {
		str := scanner.Text()
		if str == "" {
			continue
		}
		splitStr := strings.Split(str, " ")
		timea, err := time.Parse("2006/01/02", splitStr[1])
		if err != nil {
			continue
		}
		if !time.Now().After(timea.Add(time.Hour * 24 * 30)) {
			_, _ = writer.WriteString(scanner.Text() + "\n")
		}
	}
	_ = writer.Flush()
	_ = readF.Close()
	_ = writeF.Close()

	_ = os.Remove(path)
	_ = os.Rename(path2, path)

	file, _ := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	gin.DefaultWriter = file

}
