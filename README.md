批量执行工具PDO,主要是解决减少批量执行的繁锁,更安全便捷的操作命令,尤其是解决ORP的目录依赖的问题.

* 工具名称: pdo(parallel do something) 

         work@yf-orp-apollo.yf01:router$ which pdo
         /usr/local/otools/bin/pdo
         work@yf-orp-apollo.yf01:router$ which pdo2
         /usr/local/otools/bin/pdo2

* pdo V1 因为还有些地方在使用所以,将version 1的文档链接起来. [pdov1.md](pdov1.md)
* pdo V2 是最新的工具,取消了原来依赖ofind的方式,以下主要是介绍pdo2这个工具.
* pdo V2 优化了输出,可以时时写入,可以超时中断,区分屏幕输出与文件写入,屏幕输出为一次性输出,文件写入为时时写入.
* pdo V2 优化了并发输出,可以让并发速度更快,并且加入了很多其它复杂的功能.

* 更新历史

	```
// Version 2.0.20140821 增加print打印列表和prev提前显示的功能
// Version 2.0.20141121 fix bugs: -y 第一个job会先执行完才继续 ; ssh 输出的warning ; -o 输出文件冲突的问题; -o 失败不显示状态.
// Version 2.0.20151228 增加bns noahTree -b 的input
// Version 2.0.20151231 增加指定User Name
// Version 2.0.20160130 retry -R 优先及最高. setup完善.
// Version 2.0.20160202 -bns -noah 增加多个的支持,中间用逗号分割.
```

## 安装

### 依赖

1. 需要有一个中控机与被管理机器建立了无密码的密钥关系.
2. 需要有go语言的环境,进行编译安装.这里没有提供bin文件.
3. 自己所测试的环境有,centos,Redhat,osx.

### 编译

先获取依赖的第三方库: 

	```
    go get github.com/cihub/seelog
    go get github.com/robfig/config
    ```

安装go 环境.

    ```
    go build pdo2.go
    ```
    
### 配置
 
 第一次创建配置文件,会创建~/.pdo/pdo.conf ~/.pdo/log.xml 两个文件. 可以进行定制.
 
 ```
 pdo2 setup 
 ```  
    
###  Pdo Help 

```
Usage: pdo2 [input control][thread control] [output control][subcommand] <content>

  input control:
    -f <file>           from File "HOST PATH".
    -a <orp appname>    from database.
    -p <orp product>    from database.
    -R                  from last failure list.
    -bns <bns service>    from bns service, eg: pdo2 -bns redis.ksarch.all -r 10 "pwd"
    -noah <noah tree path>    from noah tree path, eg: pdo2 -noah BAIDU_WAIMAI_WAIMAI -r 10 "pwd"
    default             from pipe,eg: cat file | pdo2

  output control:
    default             display after finish.
    -show <row>         display line by line.
    -o <dir>            save to directory.

  thread control:
    -r <int>            concurrent processing ,default 1.
    -t <10s>            over the time ,kill process.
    -T <1m>             Interval time between concurrent programs.
    -y                  default need input y .
    -q                  quiet mode , not display the head infomation.

  subcommand :
    copy <file> <destination>   copy file to remote host.
    script <script file>        execute script file on remote host.
    cmd <config shortcmd>       use shortcmd in the pdo.conf.
    tail <file>                 mulit tail -f file .
    md5sum <file>               get md5sum file and count md5.
    help <subcommand>           get subcommand help.
    print                       print host list. hostname path.
    version                     get the pdo version.
    setup                       setup the configuration at the first time .[not finished]
    conf                        save the used args.[not finished]

  Examples:
  ##simple ,read from pipe.
    cat list | pdo2 "pwd"
  ##-a from orp , -r  concurrent processing
    pdo2 -a download-client -r 10 "pwd"
  ##-show row ,show line by line
    pdo2 -p tieba -y -show row "pwd"
  ##copy files
    pdo2 -a download-client copy 1.txt /tmp/
  ##excute script files
    pdo2 -a download-client script test.sh
  ## local command
    pdo2 -a download-client "scp a.txt {{.Host}}:{{.Path}}/log/"
```

## pdo 结构

``` 
Usage: pdo2 [input control][thread control] [output control][subcommand] <content>
 ```

分为四部分:

* input control 输入来源及控制,主要是不同的列表输入方式与控制 .
* thread control 并发线程控制
* output contorl 不同的输出方式
* [subcommand] <content> 命令部分 可以使用subcommand 二级命令 也可以直接使用命令 直接使用命令使用引号括起来.

**注意点: 所有的控制参数需要在二级命令之前,如果在命令之后,将失去作用.**

[pdo图](http://gitlab.baidu.com/zhangjian12/pdo/blob/master/PDO%E5%B7%A5%E5%85%B7.png)

##  input control 输入

获取机器列表和相对应的路径有三种途径.

1. -f 文件,host的列表文件,可以是一列,也可以是两列有相关的目录依赖.后面有例子.
2. -a app名字;-p 产品名;-a支持多app采用 app1,app2逗号分隔.
3. 标准输入 cat 1.host | pdo
4. -R当使用的时候,使用的是上次失败的列表.详细查看例子"Retry功能"

###  输入列表过滤

1. -i yf01,dbl01,cq02 过滤机房名称,多个可用逗号隔开.(过滤是说去除)
2. -I JX/TC  过滤逻辑机房,配置在~/.pdo/pdo.conf 中(自定义都可以)
3. --host= 指定host访问 主要适用ORP 一台物理机器上多个相同app的容器场景,只能配合-a/-p使用.

配置文件中: 

```
        [IDC]
        JX:yf01,cq01,dbl01,ai01
        TC:cq02,tc,m1,db01
```

## output control 输出控制

输出方式有三种:

* 默认是事后输出,只有等远程目标机器执行完所有的请求才会输出标准输出.
* -o <dir> 是将标准输出保存为按host的文件,如果目录下面存在相同的host 会在原来的基础上加1 如: cq02-orp-app001.cq02_1
* -show <row>  按行输出,将标准输出添加上host标签 按行进行输出. 
* -show <row> -match <string> 按行输出可以-match 字符串,高亮显示 .

## thread control 线程控制 

线程控制主要是pdo进程及线程并发度,超时相关的控制 .

*    -r <int>       线程并发度控制,默认是1, rd有最大上限.
*    -t <10s>      超时控制,如果超时将会有killed over time的提示.输入时间带单位,如10s(10秒),5m(5分) ,1h(1小时)
*    -T <1m>        间隔控制,主要是线程之间或者并发度之间的时间控制,有点类似在命令里面输入sleep ,区别就是 使用-T 输出是立即输出的屏幕,使用sleep 是sleep之后返回的.
*   -y                不使用-y,默认是需要进行命令和单机确认.
*   -q                主要是quiet模式,不显示头部信息.
    

## subcommand : 子命令

二级子命令,主要是封装了很多功能,可以简单直接的使用. 

```    
*  copy <file> <destination>   复制文件 可以文件也可以是目录.支持相对和绝对路径.
*  script <script file>         将本地的一个脚本,在远程执行.
*  cmd <config shortcmd>       短命令,主要是用于常用复杂命令,可以将短命令配置在pdo.conf中.
*  tail <file>                 相当于mulit tail -f file . 会在一个屏幕同时输出多个终端的内容,建议不要对大日志文件里面操作.
*  md5sum <file>              查看某个文件的md5值,并且进行统计.
*  help <subcommand>           get subcommand help. 子命令help 暂时还无.
*  version                     get the pdo version.
*  setup                       setup the configuration at the first time .[not finished]
*  conf                        save the used args.[not finished]
```
        
## pdo应用举例

### 配置文件

```
     [PDO]
     Log:/home/work/.pdo/log/pdo.log  //日志配置

     [Mysql] //数据库配置
     Host:st01-orp-con00.st01
     Port:3306
     DBname:orp
     User:rd
     Pass:MhxzKhl
 
     [IDC] //过滤物理idc/或者机器名后缀
     JX:yf01,cq01,dbl01,ai01
     TC:cq02,tc,m1,db01
 
     [CMD] //短命令
     restart: bash bin/orpControl.sh N%%N%%N%%restart
```

### host列表文件

机器列表主要是两列,第一列必需有为host,第二列如果存在必需是路径 可以是相对和绝对路径.

```
    cat godir/1.list  
    yf-orp-pre01.vm /home/work/orp001
    yf-orp-app01.yf01 /home/work/orp001
    yf-orp-app02.yf01 /home/work/orp001
```

### 使用管道方式

```
    cat godir/1.list | pdo2 -r 2 "pwd"
    >>>> Welcome zhangjian12...
    yf-orp-pre01.vm          -/home/work/orp001    yf-orp-app01.yf01        -/home/work/orp001
    yf-orp-app02.yf01        -/home/work/orp001    yf-orp-app03.yf01        -/home/work/
```
    
### 使用app方式与简写命令

使用数据库app方式,以orptest app为例,cmd为缩写命令= bash bin/orpControl.sh N%%N%%N%%restart

```
    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest cmd restart
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
    #--Total--#  2
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
```
    
### 使用idc过滤

```
    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest -i yf01 cmd restart 
    >>>> Welcome zhangjian12...
    dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app17.m1          -/home/work/orp001
     #--Total--#  2
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
```
    
### 使用逻辑机房过滤

```
    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest  -I JX cmd restart
    >>>> Welcome zhangjian12...
    m1-orp-app17.m1          -/home/work/orp001    m1-orp-app25.m1          -/home/work/orp001
   
    #--Total--#  2
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
```
    
###  目录文件输出

使用带-o 指定输出目录,将不会再打印在屏幕上,主要是对grep日志这种需求使用.反之就会输出在屏幕上.
    
 ```  
    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest  -o xxxout "pwd"
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
   
    #--Total--#  25
    #---CMD---#  pwd
    Continue (y/n):y
    go on ...
    [1/25] yf-orp-app01.yf01  [SUCCESS].
    Continue (y/n):y
    go on ...
    [2/25] yf-orp-app02.yf01  [SUCCESS].
 ```
    
### 超时killed进程

```
    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest -t 1s -r 3 "cat log/ral-zoo.log"
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
     #--Total--#  26
    #---CMD---#  cat log/ral-zoo.log
    Continue (y/n):y
    go on ...
    [1/26] yf-orp-app01.yf01  [Time Over KILLED].
    Continue (y/n):y
    go on ...
    [2/26] yf-orp-app04.yf01  [Time Over KILLED].
```
        
### copy文件

```
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -3  | pdo2 copy get.sh /tmp/
    >>>> Welcome zhangjian12...
    yf-orp-upload05.yf01     -/home/work           yf-orp-upload01.yf01     -/home/work
    yf-orp-upload02.yf01     -/home/work

    #--Total--#  3
    #---CMD---#  get.sh --> /tmp/
    Continue (y/n):y
    go on ...
    [1/3] yf-orp-upload05.yf01  [SUCCESS].
    
    Continue (y/n):y
    go on ...
    [2/3] yf-orp-upload01.yf01  [SUCCESS].
   
    
    //检查下文件 
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -3  | pdo "ls /tmp/get.sh"
    >>>> Welcome zhangjian12...
    yf-orp-upload05.yf01     -/home/work           yf-orp-upload01.yf01     -/home/work
    yf-orp-upload02.yf01     -/home/work
    
    #--Total--#  3
    #---CMD---#  ls /tmp/get.sh
    Continue (y/n):y
    go on ...
    [1/3] yf-orp-upload05.yf01  [SUCCESS].
    /tmp/get.sh
    
    Continue (y/n):y
    go on ...
    [2/3] yf-orp-upload01.yf01  [SUCCESS].
    /tmp/get.sh
    
    [3/3] yf-orp-upload02.yf01  [SUCCESS].
    /tmp/get.sh
```

### Retry功能

这次多加两台服务器,有两台是没有这个脚本文件的.

```
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -5  | pdo "ls /tmp/get.sh"
    >>>> Welcome zhangjian12...
    yf-orp-upload05.yf01     -/home/work           yf-orp-upload01.yf01     -/home/work
    yf-orp-upload02.yf01     -/home/work           yf-orp-upload03.yf01     -/home/work
    yf-orp-upload04.yf01     -/home/work

    #--Total--#  5
    #---CMD---#  ls /tmp/get.sh
    Continue (y/n):y
    go on ...
    [1/5] yf-orp-upload05.yf01  [SUCCESS].
    /tmp/get.sh
    
    Continue (y/n):y
    go on ...
    [2/5] yf-orp-upload01.yf01  [SUCCESS].
    /tmp/get.sh
    
    [3/5] yf-orp-upload02.yf01  [SUCCESS].
    /tmp/get.sh
    
    [4/5] yf-orp-upload03.yf01  [FAILED].
    ls: /tmp/get.sh: No such file or directory
    
    [5/5] yf-orp-upload04.yf01  [FAILED].
    ls: /tmp/get.sh: No such file or directory
    
    //使用-R 就可以直接拿到上一次执行失败的列表.
    work@yf-orp-apollo.yf01:upload_server$pdo -R "ls /tmp/get.sh"
    >>>> Welcome zhangjian12...
    yf-orp-upload03.yf01     -/home/work           yf-orp-upload04.yf01     -/home/work
    
    
    #--Total--#  2
    #---CMD---#  ls /tmp/get.sh
    Continue (y/n):y
    go on ...
    [1/2] yf-orp-upload03.yf01  [FAILED].
    ls: /tmp/get.sh: No such file or directory
    
    //如果是使用的ctrl+C中断了列表,-R会记录未执行完(包括已经执行但失败的列表)
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -5  | pdo  -T 10s "ls /tmp/get.sh"
    >>>> Welcome zhangjian12...
    yf-orp-upload05.yf01     -/home/work           yf-orp-upload01.yf01     -/home/work
    yf-orp-upload02.yf01     -/home/work           yf-orp-upload03.yf01     -/home/work
    yf-orp-upload04.yf01     -/home/work

    #--Total--#  5
    #---CMD---#  ls /tmp/get.sh
    Continue (y/n):y
    go on ...
    [1/5] yf-orp-upload05.yf01  [SUCCESS].
    /tmp/get.sh
    
    Continue (y/n):y
    go on ...
    [2/5] yf-orp-upload01.yf01  [SUCCESS].
    /tmp/get.sh
    
    ^Cwork@yf-orp-apollo.yf01:upload_server$ pdo -R "ls /tmp/get.sh"
    >>>> Welcome zhangjian12...
    yf-orp-upload02.yf01     -/home/work           yf-orp-upload03.yf01     -/home/work
    yf-orp-upload04.yf01     -/home/work
    
    #--Total--#  3
    #---CMD---#  ls /tmp/get.sh
    Continue (y/n):y
    go on ...
    [1/3] yf-orp-upload02.yf01  [SUCCESS].
    /tmp/get.sh
    
    Continue (y/n):
```
 
### script 脚本执行功能

```
    work@yf-orp-apollo.yf01:upload_server$ cat t.sh
    #!/bin/bash

    cd /tmp/ && pwd
    echo "test"
    touch /tmp/t.log
```
执行
```    
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -3  | pdo2 script t.sh
    >>>> Welcome zhangjian12...
    yf-orp-upload05.yf01     -/home/work           yf-orp-upload01.yf01     -/home/work
    yf-orp-upload02.yf01     -/home/work

    #--Total--#  3
    #---CMD---#  Script: t.sh
    Continue (y/n):y
    go on ...
    [1/3] yf-orp-upload05.yf01  [SUCCESS].
    /tmp
    test
```
        
###  行显示与匹配

这个功能有两种使用场景:

1. 有点类似multi tail 可以实现同时tail多个日志,显示在一个屏幕内,而且可以对match的字符串进行高亮显示.
2. 如果输出是单行输出,没有状态显示会显示得加的美观和可参考性.

所以这种显示方式取决于时间的先后顺序,交错输出.

拿redis的迁移过程为例子: 

> redis迁移至少有原来的一主一从,新主和新从.在迁移的过程中需要同时观察四台服务器的变化.如果是每次ssh四台服务器tail 日志是很麻烦而且容易出错.

现在使用pdo命令:
 
 ```       
        //操作的主机列表1.list
        tc-arch-redis40.tc /home/arch/redis-ting-listen-shard3   //old master 
        cq02-arch-redis80.cq02 /home/arch/redis-ting-listen-shard3 //new master 
        yf-arch-redis40.yf01 /home/arch/redis-ting-listen-shard3 //old slave 
        jx-arch-redis80.jx /home/arch/redis-ting-listen-shard3  //new slave 
        第一步操作:  yf-arch-redis40.yf01为主 --> cq02-arch-redis80.cq02 

        #命令
        #cat 1.list | pdo2 -r 5 -y -show row  -match "success" "tail -f log/redis.log"
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:56:51 * Slave ask for new-synchronization  //被要求同步 
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 13:56:58 * (non critical): Master does not understand REPLCONF listening-port: Reading from master: Connection timed out
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:56:58 * Slave ask for synchronization
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:56:58 * Starting BGSAVE for SYNC
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:56:58 * Background saving started by pid 22855
        > yf-arch-redis40.yf01      >> [22855] 06 Jan 13:58:31 * DB saved on disk   //dump到磁盘
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:58:31 * Background saving terminated with success
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 13:58:31 * MASTER <-> SLAVE sync: receiving 1868940396 bytes from master  //从接收到主的文件
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 13:58:47 * MASTER <-> SLAVE sync: Loading DB in memory //将接收到的文件加载到内存
        > yf-arch-redis40.yf01      >> [11523] 06 Jan 13:58:47 * Synchronization with slave succeeded  //文件同步成功
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 14:01:21 # Update masterstarttime[1382324097] after loading db
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 14:01:21 * AA: see masterstarttime: ip[10.36.114.56], port[9973], timestamp[1382324097]
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 14:01:21 * Write aof_global_offset[92961804447] to new aof_file[46] success
        > cq02-arch-redis80.cq02    >> [14752] 06 Jan 14:01:21 * MASTER <-> SLAVE sync: Finished with success //slave完成主从同步,说明第一步已经结束.

```

说明:
 
1.  因为是tail -f 是不会主动退出命令,所以需要使用-y 和使用-r 来增加并发量,不然会先进行单台显示 ,而不会显示后面的.
2.  match是匹配字符 串,暂时不支持正则,会进行高亮显示.红色显示.
3.  -show现在只支持row这一种方式,默认方式还是原来的缓存输出方式.  

以下是一个测试脚本:随机打印数字 1.sh

```
        #!/bin/bash
        for x in `seq 1 10` ; do
            echo $x
            sleep $[ ( $RANDOM % 4 )  + 1 ]s
        done 
        
         //可以使用如下命令:
       # cat 1.list | pdo -r 5 -y -show row  -match "5" -e 1.sh
```

还有更多的组合,可以找实验.
