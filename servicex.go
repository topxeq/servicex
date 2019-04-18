package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/kardianos/service"
	"github.com/topxeq/tk"

	"github.com/gorilla/websocket"
)

var versionG string = "0.01a"

// cmd/service
var runModeG string = ""
var currentOSG string = ""
var basePathG string = ""
var configFileNameG string = ""
var currentPortG string = ""

// var currentPortSG string = ""
var servicexBasePathG = "e:\\servicex\\"
var servicexDomainNameG = "servicex.domain.com:7466"

var defaultBasePathG string
var defaultConfigFileNameG string = "servicex.cfg"

var serverUrlG = ""

var serviceModeG bool = false

var exit = make(chan struct{})

func plByMode(formatA string, argsA ...interface{}) {
	if runModeG == "cmd" {
		tk.Pl(formatA, argsA...)
	} else {
		tk.AddDebugF(formatA, argsA...)
	}
}

type program struct {
	BasePath string
}

func (p *program) Start(s service.Service) error {
	serviceModeG = true

	go p.run()

	return nil
}

func (p *program) run() {
	go doWork()
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func doWork() {

	go Svc()

	for {
		select {
		case <-exit:
			os.Exit(0)
			return
		}
	}
}

func stopWork() {

	// logWithTime("Service stop running!")
	exit <- struct{}{}
}

func checkOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{CheckOrigin: checkOrigin}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		tk.Pl("upgrade: %v", err)
		return
	}
	defer c.Close()

	tk.Pl("conn: %s\n", r.RemoteAddr)

	reqT := r

	reqT.ParseForm()

	tk.Pl("%#v", reqT.Form)

	userT := tk.GetFormValueWithDefaultValue(reqT, "user", "")

	var errT error

	if userT == "" {
		if errT = c.WriteMessage(websocket.TextMessage, []byte("命令格式错误")); errT != nil {
			tk.Pl("send err: %v", errT.Error())
			return
		}
		return
	}

	for {
		var cmdLineT string

		messageTypeT, messageT, errT := c.ReadMessage()

		if errT != nil {
			tk.Pl("receive error: %v", errT.Error())
			return
		}

		cmdLineT = string(messageT)

		tk.Pl("Received: (%v) %v", messageTypeT, cmdLineT)

		if errT != nil {
			tk.Pl("receive error: %v", errT.Error())
			return
		}

		tk.Pl("Received: %v", cmdLineT)

		var cmdListT []string

		errT = json.Unmarshal([]byte(cmdLineT), cmdListT)

		if errT != nil {
			if errT = c.WriteMessage(websocket.TextMessage, []byte("命令格式错误")); errT != nil {
				tk.Pl("send err: %v", errT.Error())
				return
			}

			continue
		}

		if errT = c.WriteMessage(websocket.TextMessage, []byte(tk.Spr("无效的命令：%v", cmdLineT))); errT != nil {
			tk.Pl("send err: %v", errT.Error())
			return
		}
	}
}

func startWebSocketServer(portA string) {
	defer func() {
		if r := recover(); r != nil {
			tk.LogWithTimeCompact("startWebSocketServer: Recovered: %v\n%v", r, string(debug.Stack()))
		}
	}()

	tk.LogWithTimeCompact("trying startWebSocketServer, port: %v", portA)

	http.HandleFunc("/wapi", webSocketHandler)

	err := http.ListenAndServe(":"+portA, nil)
	if err != nil {
		plByMode("ListenAndServeHttp: %v", err.Error())
		tk.LogWithTimeCompact("ListenAndServeWebSocket: %v", err.Error())
	}
}

func startHttpServer(portA string) {
	defer func() {
		if r := recover(); r != nil {
			tk.LogWithTimeCompact("startHttpServer: Recovered: %v\n%v", r, string(debug.Stack()))
		}
	}()

	tk.LogWithTimeCompact("trying startHttpServer, port: %v", portA)

	http.HandleFunc("/japi", japiHandler)

	err := http.ListenAndServe(":"+portA, nil)
	if err != nil {
		plByMode("ListenAndServeHttp: %v", err.Error())
		tk.LogWithTimeCompact("ListenAndServeHttp: %v", err.Error())
	}

}

func startHttpsServer(portA string) {
	plByMode("https port: %v", portA)
	tk.LogWithTimeCompact("trying startHttpsServer, port: %v", portA)

	err := http.ListenAndServeTLS(":"+portA, filepath.Join(servicexBasePathG, "server.crt"), filepath.Join(servicexBasePathG, "server.key"), nil)
	if err != nil {
		plByMode("ListenAndServeHttps: %v", err.Error())
	}
}

func doJapi(res http.ResponseWriter, req *http.Request) string {

	defer func() {
		if r := recover(); r != nil {
			tk.AddDebugF("japi: Recovered: %v\n%v", r, string(debug.Stack()))
			tk.AddDebugF("japi Recovered: %v\n%v", r, string(debug.Stack()))
		}
	}()

	if req != nil {
		req.ParseForm()
	}

	reqT := strings.ToLower(tk.GetFormValueWithDefaultValue(req, "req", ""))

	if res != nil {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Content-Type", "text/json;charset=utf-8")
	}

	res.WriteHeader(http.StatusOK)

	switch reqT {

	case "debug":
		{
			tk.Pl("%v", req)
			a := make([]int, 3)
			a[5] = 8

			return tk.GenerateJSONPResponse("success", tk.IntToStr(a[5]), req)
		}

	case "getdebug":
		{
			res.Header().Set("Content-Type", "text/plain;charset=utf-8")

			res.WriteHeader(http.StatusOK)

			return tk.GenerateJSONPResponse("success", tk.GetDebug(), req)
		}

	case "cleardebug":
		{
			tk.ClearDebug()
			return tk.GenerateJSONPResponse("success", "", req)
		}

	case "md5":
		{
			strT := tk.GetFormValueWithDefaultValue(req, "text", "")

			return tk.GenerateJSONPResponse("success", tk.MD5Encrypt(strT), req)
		}

	case "showip":
		{
			return tk.GenerateJSONPResponse("success", req.RemoteAddr, req)
		}

	default:
		return tk.GenerateJSONPResponse("fail", tk.Spr("unknown request: %v", req), req)
	}
}

func japiHandler(w http.ResponseWriter, req *http.Request) {
	rs := doJapi(w, req)

	w.Header().Set("Content-Type", "text/plain")

	w.Write([]byte(rs))
}

func Svc() {
	tk.SetLogFile(filepath.Join(basePathG, "servicex.log"))

	defer func() {
		if v := recover(); v != nil {
			tk.LogWithTimeCompact("panic in svc %v", v)
		}
	}()

	if runModeG != "cmd" {
		runModeG = "service"
	}

	plByMode("runModeG: %v", runModeG)

	tk.DebugModeG = true

	tk.LogWithTimeCompact("servicex V%v", versionG)
	tk.LogWithTimeCompact("os: %v, basePathG: %v, configFileNameG: %v", runtime.GOOS, basePathG, defaultConfigFileNameG)

	if tk.GetOSName() == "windows" {
		plByMode("Windows mode")
		currentOSG = "win"
		basePathG = "c:\\servicex"
		configFileNameG = "servicexwin.cfg"
	} else {
		plByMode("Linux mode")
		currentOSG = "linux"
		basePathG = "/servicex"
		configFileNameG = "servicexlinux.cfg"
	}

	if !tk.IfFileExists(basePathG) {
		os.MkdirAll(basePathG, 0777)
	}

	tk.SetLogFile(filepath.Join(basePathG, "servicex.log"))

	currentPortG := "7466"

	cfgFileNameT := filepath.Join(basePathG, configFileNameG)
	if tk.IfFileExists(cfgFileNameT) {
		plByMode("Process config file: %v", cfgFileNameT)
		fileContentT := tk.LoadSimpleMapFromFile(cfgFileNameT)

		if fileContentT != nil {
			currentPortG = fileContentT["port"]
			servicexBasePathG = fileContentT["servicexBasePath"]
			servicexDomainNameG = fileContentT["servicexDomainName"]
		}
	}

	plByMode("currentPortG: %v, servicexBasePathG: %v, servicexDomainNameG: %v", currentPortG, servicexBasePathG, servicexDomainNameG)

	tk.LogWithTimeCompact("currentPortG: %v, servicexBasePathG: %v, servicexDomainNameG: %v", currentPortG, servicexBasePathG, servicexDomainNameG)

	tk.LogWithTimeCompact("Service started.")
	tk.LogWithTimeCompact("Using config file: %v", cfgFileNameT)

	go startHttpServer(currentPortG)

	go startHttpsServer(tk.IntToStr(tk.StrToIntWithDefaultValue(currentPortG, 7466) + 1))
	go startWebSocketServer(tk.IntToStr(tk.StrToIntWithDefaultValue(currentPortG, 7466) - 1))

}

func initSvc() *service.Service {
	svcConfigT := &service.Config{
		Name:        "servicex",
		DisplayName: "servicex",
		Description: "servicex service by TopXeQ V" + versionG,
	}

	prgT := &program{BasePath: basePathG}
	var s, err = service.New(prgT, svcConfigT)

	if err != nil {
		tk.LogWithTimeCompact("%s unable to start: %s\n", svcConfigT.DisplayName, err)
		return nil
	}

	return &s
}

func runCmd(cmdLineA []string) {
	cmdT := ""

	for _, v := range cmdLineA {
		if !strings.HasPrefix(v, "-") {
			cmdT = v
			break
		}
	}

	// if cmdT == "" {
	// 	fmt.Println("empty command")
	// 	return
	// }

	var errT error

	basePathG = tk.GetSwitchWithDefaultValue(cmdLineA, "-base=", "")

	if strings.TrimSpace(basePathG) == "" {
		basePathG, errT = filepath.Abs(defaultBasePathG)

		if errT != nil {
			fmt.Printf("invalid base path: %v\n", defaultBasePathG)
			return
		}
	}

	// verboseT := ifSwitchExists(cmdLineA, "-v")

	tk.EnsureMakeDirs(basePathG)

	if !tk.IfFileExists(basePathG) {
		fmt.Printf("base path not exists: %v, use current directory instead\n", basePathG)
		basePathG = "."
		return
	}

	if !tk.IsDirectory(basePathG) {
		fmt.Printf("base path not exists: %v\n", basePathG)
		return
	}

	// fmt.Printf("base path: %v\n", basePathG)

	switch cmdT {
	case "version":
		fmt.Printf("servicex V%v", versionG)
	case "test":
		{
			tk.Pl("servicexBasePathG: %v", servicexBasePathG)
		}
		break
	case "go":
		go doWork()

		for {
			tk.SleepSeconds(1)
		}
	case "", "run":
		s := initSvc()

		if s == nil {
			tk.LogWithTimeCompact("Failed to init service")
			break
		}

		err := (*s).Run()
		if err != nil {
			tk.LogWithTimeCompact("Service \"%s\" failed to run: %v.", (*s).String(), err)
		}
	case "installonly":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}

		err := (*s).Install()
		if err != nil {
			fmt.Printf("Failed to install: %s\n", err)
			return
		}

		fmt.Printf("Service \"%s\" installed.\n", (*s).String())

	case "install":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}

		fmt.Printf("Installing service \"%v\"...\n", (*s).String())

		err := (*s).Install()
		if err != nil {
			fmt.Printf("Failed to install: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" installed.\n", (*s).String())

		fmt.Printf("Starting service \"%v\"...\n", (*s).String())

		err = (*s).Start()
		if err != nil {
			fmt.Printf("Failed to start: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" started.\n", (*s).String())
	case "uninstall":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}

		err := (*s).Stop()
		if err != nil {
			fmt.Printf("Failed to stop: %s\n", err)
		} else {
			fmt.Printf("Service \"%s\" stopped.\n", (*s).String())
		}

		err = (*s).Uninstall()
		if err != nil {
			fmt.Printf("Failed to remove: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" removed.\n", (*s).String())
	case "reinstall":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}

		err := (*s).Stop()
		if err != nil {
			fmt.Printf("Failed to stop: %s\n", err)
		} else {
			fmt.Printf("Service \"%s\" stopped.\n", (*s).String())
		}

		err = (*s).Uninstall()
		if err != nil {
			fmt.Printf("Failed to remove: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" removed.\n", (*s).String())

		err = (*s).Install()
		if err != nil {
			fmt.Printf("Failed to install: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" installed.\n", (*s).String())

		err = (*s).Start()
		if err != nil {
			fmt.Printf("Failed to start: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" started.\n", (*s).String())
	case "start":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}

		err := (*s).Start()
		if err != nil {
			fmt.Printf("Failed to start: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" started.\n", (*s).String())
	case "stop":
		s := initSvc()

		if s == nil {
			fmt.Printf("Failed to install")
			break
		}
		err := (*s).Stop()
		if err != nil {
			fmt.Printf("Failed to stop: %s\n", err)
			return
		}
		fmt.Printf("Service \"%s\" stopped.\n", (*s).String())
	default:
		fmt.Println("unknown command")
		break
	}

}

func main() {

	if strings.HasPrefix(runtime.GOOS, "win") {
		defaultBasePathG = "c:\\servicex"
	} else {
		defaultBasePathG = "/servicex"
	}

	if len(os.Args) < 2 {
		plByMode("servicex V%v is in service(server) mode. Running the application without any arguments will cause it in service mode.\n", versionG)
		serviceModeG = true

		s := initSvc()

		if s == nil {
			tk.LogWithTimeCompact("Failed to init service")
			return
		}

		err := (*s).Run()
		if err != nil {
			tk.LogWithTimeCompact("Service \"%s\" failed to run.", (*s).String())
		}

		tk.Pl("err: %#v", err.Error())
		return
	}

	if tk.GetOSName() == "windows" {
		plByMode("Windows mode")
		currentOSG = "win"
		basePathG = "c:\\servicex"
		configFileNameG = "servicexwin.cfg"
	} else {
		plByMode("Linux mode")
		currentOSG = "linux"
		basePathG = "/servicex"
		configFileNameG = "servicexlinux.cfg"
	}

	if !tk.IfFileExists(basePathG) {
		os.MkdirAll(basePathG, 0777)
	}

	tk.SetLogFile(filepath.Join(basePathG, "servicex.log"))

	currentPortG := "7466"

	cfgFileNameT := filepath.Join(basePathG, configFileNameG)
	if tk.IfFileExists(cfgFileNameT) {
		plByMode("Process config file: %v", cfgFileNameT)
		fileContentT := tk.LoadSimpleMapFromFile(cfgFileNameT)

		if fileContentT != nil {
			currentPortG = fileContentT["port"]
			servicexBasePathG = fileContentT["servicexBasePath"]
			servicexDomainNameG = fileContentT["servicexDomainName"]
		}
	}

	plByMode("currentPortG: %v, servicexBasePathG: %v, servicexDomainNameG: %v", currentPortG, servicexBasePathG, servicexDomainNameG)

	runCmd(os.Args[1:])

}
