web:
	cache
	dataaccess
		database.go：mysql存取
		file.go:本地文件存取
	doc
	pythonscripts：python服务的所有文件
	service:
		accounting
			accounting.go:用于将数据库查询出来的信息排序并分类
		lowermanger
			filemanager.go: 管理filestatus对象，记录文件信息
			socketmanager.go：管理用于和python服务通信的socket对象
			websocketmanager.go：管理用于和web前端通信的websocket对象
		topmanager
			pythonmanager.go：管理文件处理时如何调用python，并维护一个用于记录一个文件在python中处理进程的pythonstatus对象
			servercachemanager：服务器有文件缓存数量限制，这里面有管理一个queue，其中放了filestatus的对象
			sessionmanager：管理用户的session信息，其中包括用户拥有哪些file，拥有哪些websocket。
		service.go：网页收到用户的一些请求后的逻辑，比如说如何上传文件（涉及到文件保存，文件名分配，filestatus分配等等）。
	util:
		config.go:读取本地配置confi.ini信息用于配置服务
		cookie.go：如何生成一个加密的cookie
		enumerate：枚举管理
		log.go：初始化一个全局的logrus来进行服务器全程的日志管理
		util.go：排序算法
	web.go: handlefunc和listenandserve