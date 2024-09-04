# 数据结构

filestatus: 文件的从属，文件的地址，文件的分析状态
PythonProcessStatus： 一条python命令及其是否在运行


# 类间依赖
底层类：file： filestatus,.
 database
 socket：socketstatus
上层类：util


# 程序运行中数据分类

任务状态变化：文件信息中，Python Status中。变化时将通知用户。

# 程序处理状态

created -> running -> finished

created -> running -> failed

process: RUNNING and IDLE

# 后端运行逻辑

启动golang和python，golang作为网页后端，python作为文件分析服务。

### golang初始化数据库连接

连接数据库，创建新表 userinfo 和 fileinfo。

### 初始化config 文件

### Python初始化并listen端口

### 初始化网页后端主数据存储

Session Status Manager管理用户信息。其中一个map来存储用户有哪些文件，一个map存储用户有哪些连接（一个用户可以通过不同的浏览器窗口开不同的连接。

Cache Queue 管理已存储的文件信息。

Python Status Manager 管理由python处理的文件的信息。

绑定Lisner，和python连接，听python发来的消息。

### 网页入口

接收发来的文件。为文件创建UUID。

当Cache Queue已满时，强制删除最老的信息，并删除文件。

#### 将文件信息写入内存和本地存储和数据库

保存到本地，创建 文件信息 对象，在user的file map中添加，在fileinfo表中添加，在cache queue中添加。

开始解析dbglog。创建3个所需的任务（其第一是dbglog分析）。启动RPC，更新相关数据状态，向python发送命令。此时文件的控制权在python中，文件可能处于锁定状态。

前端网页重定向。

### 文件信息列表网页

前端将请求websocket。

websocket的信息加入user的websocket map中。 听网页的命令：删除所有该用户的文件或开始sctp分析。

### DBGLOG分析页面

通过userid和fileid找到文件信息，并显示。

### 因用户指令和删除所有的文件

查找userid中所有的文件信息，每个进行一次强制删除。

### 因Cache Queue满，删除最老的文件

### 删除单个文件的逻辑

关闭并删除3个所需python的命令信息。从cache queue中删除，从数据库中删除，从user的file map中删除，从本地删除。