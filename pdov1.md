ORP批量执行工具PDO,主要是解决减少批量执行的繁锁,更安全便捷的操作命令,尤其是解决ORP的目录依赖的问题.

* 工具名称: pdo(parallel do something)

         work@yf-orp-apollo.yf01:router$ which pdo
         /usr/local/otools/bin/pdo
         work@yf-orp-apollo.yf01:router$ which pdo2
         /usr/local/otools/bin/pdo2

* pdo V2 是最新的工具,取消了原来依赖ofind的方式,以下主要是介绍pdo2这个工具.
* pdo V2 优化了输出,可以时时写入,可以超时中断,区分屏幕输出与文件写入,屏幕输出为一次性输出,文件写入为时时写入.

## pdo 结构

### pdo 来源

获取机器列表和相对应的路径有三种途径.

1. -f 文件
2. -a app名字 -a支持多app采用 app1,app2逗号分隔.
2. -p 产品名 
3. 标准输入 cat 1.host | ./pdo2 

### pdo 过滤

1. -i yf01,dbl01,cq02 过滤机房名称,多个可用逗号隔开.
2. -I JX/TC  过滤逻辑机房,配置在-c configure file 中

配置文件中: 

        [IDC]
        JX:yf01,cq01,dbl01,ai01
        TC:cq02,tc,m1,db01
        
### pdo 功能

会有主机和命令和单台执行确认.

* -r 数字 ,并发量,默认为1
* -C configure , 配置文件,默认为~/.pdo/pdo.conf
* -o dir , 输出目录,默认为空,会打印在屏幕,如果添加则只会打印到文件中.
* -cmd 命令, 命令缩写,在配置文件中.
* -t 超时结束,超时时间默认5min钟.
* -y 不用输入确认.
* -c copy file/dir ,复制文件/或者目录,目的的目录必需存在.
* -e script 执行脚本.
* --log 日志输出文件,默认定义在配置文件中,如果没有定义就在当前目录的pdo.log

        [PDO]
        Log:/home/work/.pdo/log/pdo.log

* -T 执行间隔时间,默认为0,比命令中加sleep效果好.
* -R retry ,失败之后retry功能,会记录上次失败的列表和ctrl+c未执行的列表.
* -temp template模板名字,在配置中.
* -b build-in 在使用template的模板时候可以嵌入脚本.
* -V 查看version 
* -show 查看显示方式"row" 行显示方式<row>.
* -match 在行显示的时候可以进行match字符串.高亮显示.
* -rule 在行显示的模式下,可以使用conf中的rule,来定义不同match的字符串的动作.

### pdo 未完成功能

* 去重与多实例,产品线重启并发优化.
* web页面展示功能.

## pdo应用举例

### 配置文件

     [PDO]
     Log:/home/work/.pdo/log/pdo.log

     [Mysql]
     Host:st01-orp-con00.st01
     Port:3306
     DBname:orp
     User:rd
     Pass:MhxzKhl
 
     [IDC]
     JX:yf01,cq01,dbl01,ai01
     TC:cq02,tc,m1,db01
 
     [CMD]
     restart: bash bin/orpControl.sh N%%N%%N%%restart

### host列表文件

    cat godir/1.list
    yf-orp-pre01.vm /home/work/orp001
    yf-orp-app01.yf01 /home/work/orp001
    yf-orp-app02.yf01 /home/work/orp001
    yf-orp-app03.yf01 /home/work/orp001
    yf-orp-app04.yf01 /home/work/orp001
    1-orp-app17.m1 /home/work/orp001
    m1-orp-app25.m1 /home/work/orp001
    m1-orp-app0220.m1 /home/work/orp001
    m1-orp-app0154.m1 /home/work/orp004
    cq01-orp-app0242.cq01 /home/work/orp003
    ai-orp-app01.ai01 /home/work/orp004
    db-orp-app17.db01 /home/work/orp003
    db-orp-app63.db01 /home/work/orp001

### 使用管道方式

    cat godir/1.list | pdo2 -r 2 "pwd"
    >>>> Welcome zhangjian12...
    yf-orp-pre01.vm          -/home/work/orp001    yf-orp-app01.yf01        -/home/work/orp001
    yf-orp-app02.yf01        -/home/work/orp001    yf-orp-app03.yf01        -/home/work/orp001
    yf-orp-app04.yf01        -/home/work/orp001    1-orp-app17.m1           -/home/work/orp001
    m1-orp-app25.m1          -/home/work/orp001    m1-orp-app0220.m1        -/home/work/orp001
    m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
    ai-orp-app01.ai01        -/home/work/orp004    db-orp-app17.db01        -/home/work/orp003
    db-orp-app63.db01        -/home/work/orp001

    #--Total--#  13
    #---CMD---#  pwd
    //每一次确认
    Continue (y/n):y
    go on ...
    [1/13] yf-orp-app01.yf01  [SUCCESS].
    /home/work/orp001
    
    Continue (y/n):[1/13] yf-orp-pre01.vm  [SUCCESS].
    /home/work/orp001
    //单台执行完 第二次确认 
    Continue (y/n):
    
### 使用app方式与简写命令

使用数据库app方式,以orptest app为例,-cmd为缩写命令= bash bin/orpControl.sh N%%N%%N%%restart

    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest -cmd restart
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
    yf-orp-app03.yf01        -/home/work/orp001    yf-orp-app04.yf01        -/home/work/orp001
    yf-orp-app00.yf01        -/home/work/orp001    yf-orp-app0148.yf01      -/home/work/orp004
    dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app17.m1          -/home/work/orp001
    m1-orp-app25.m1          -/home/work/orp001    m1-orp-app0220.m1        -/home/work/orp001
    m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
    cq01-orp-app0179.cq01    -/home/work/orp001    cq01-orp-app0131.cq01    -/home/work/orp005
    st01-orp-app03.st01      -/home/work/orp001    st01-orp-app04.st01      -/home/work/orp001
    st01-orp-app02.st01      -/home/work/orp001    st01-orp-app00.st01      -/home/work/orp001
    st01-orp-app05.st01      -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
    cq02-orp-app0287.cq02    -/home/work/orp001    jx-orp-app17.jx          -/home/work/orp001
    ai-orp-app10.ai01        -/home/work/orp001    db-orp-app17.db01        -/home/work/orp003


    #--Total--#  24
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
    
### 使用idc过滤

    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest -cmd restart -i yf01
    >>>> Welcome zhangjian12...
    dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app17.m1          -/home/work/orp001
    m1-orp-app25.m1          -/home/work/orp001    m1-orp-app0220.m1        -/home/work/orp001
    m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
    cq01-orp-app0179.cq01    -/home/work/orp001    cq01-orp-app0131.cq01    -/home/work/orp005
    st01-orp-app03.st01      -/home/work/orp001    st01-orp-app04.st01      -/home/work/orp001
    st01-orp-app02.st01      -/home/work/orp001    st01-orp-app00.st01      -/home/work/orp001
    st01-orp-app05.st01      -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
    cq02-orp-app0287.cq02    -/home/work/orp001    jx-orp-app17.jx          -/home/work/orp001
    ai-orp-app10.ai01        -/home/work/orp001    db-orp-app17.db01        -/home/work/orp003


    #--Total--#  18
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
    
### 使用逻辑机房过滤

    work@yf-orp-apollo.yf01:godir$ pdo2 -a orptest -cmd restart -I JX
    >>>> Welcome zhangjian12...
    m1-orp-app17.m1          -/home/work/orp001    m1-orp-app25.m1          -/home/work/orp001
    m1-orp-app0220.m1        -/home/work/orp001    m1-orp-app0154.m1        -/home/work/orp004
    st01-orp-app03.st01      -/home/work/orp001    st01-orp-app04.st01      -/home/work/orp001
    st01-orp-app02.st01      -/home/work/orp001    st01-orp-app00.st01      -/home/work/orp001
    st01-orp-app05.st01      -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
    cq02-orp-app0287.cq02    -/home/work/orp001    jx-orp-app17.jx          -/home/work/orp001
    db-orp-app17.db01        -/home/work/orp003

    #--Total--#  13
    #---CMD---#  bash bin/orpControl.sh N%%N%%N%%restart
    Continue (y/n):
    
### -o输入与屏幕输出

使用带-o 指定输出目录,将不会再打印在屏幕上,主要是对grep日志这种需求使用.反之就会输出在屏幕上.
    
    work@yf-orp-apollo.yf01:godir$ ./pdo -a orptest  -o xxxout "pwd"
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
    yf-orp-app03.yf01        -/home/work/orp001    yf-orp-app04.yf01        -/home/work/orp001
    yf-orp-app00.yf01        -/home/work/orp001    yf-orp-app0148.yf01      -/home/work/orp004
    dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app17.m1          -/home/work/orp001
    m1-orp-app25.m1          -/home/work/orp001    m1-orp-app0220.m1        -/home/work/orp001
    m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
    cq01-orp-app0179.cq01    -/home/work/orp001    cq01-orp-app0131.cq01    -/home/work/orp005
    st01-orp-app03.st01      -/home/work/orp001    st01-orp-app04.st01      -/home/work/orp001
    st01-orp-app02.st01      -/home/work/orp001    st01-orp-app00.st01      -/home/work/orp001
    st01-orp-app05.st01      -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
    cq02-orp-app0287.cq02    -/home/work/orp001    cq02-orp-app0212.cq02    -/home/work/orp001
    jx-orp-app17.jx          -/home/work/orp001    ai-orp-app10.ai01        -/home/work/orp001
    db-orp-app17.db01        -/home/work/orp003

    #--Total--#  25
    #---CMD---#  pwd
    Continue (y/n):y
    go on ...
    [1/25] yf-orp-app01.yf01  [SUCCESS].
    Continue (y/n):y
    go on ...
    [2/25] yf-orp-app02.yf01  [SUCCESS].
    [3/25] yf-orp-app03.yf01  [SUCCESS].
    [4/25] yf-orp-app04.yf01  [SUCCESS].
    [5/25] yf-orp-app00.yf01  [SUCCESS].
    [6/25] yf-orp-app0148.yf01  [SUCCESS].
    [7/25] dbl-orp-app0109.dbl01  [SUCCESS].
    [8/25] m1-orp-app17.m1  [SUCCESS].
    [9/25] m1-orp-app25.m1  [SUCCESS].
    [10/25] m1-orp-app0220.m1  [SUCCESS].
    [11/25] m1-orp-app0154.m1  [SUCCESS].
    [12/25] cq01-orp-app0242.cq01  [SUCCESS].
    [13/25] cq01-orp-app0179.cq01  [SUCCESS].
    [14/25] cq01-orp-app0131.cq01  [SUCCESS].
    [15/25] st01-orp-app03.st01  [SUCCESS].
    [16/25] st01-orp-app04.st01  [SUCCESS].
    [17/25] st01-orp-app02.st01  [SUCCESS].
    [18/25] st01-orp-app00.st01  [SUCCESS].
    [19/25] st01-orp-app05.st01  [SUCCESS].
    [20/25] cq02-orp-app0258.cq02  [SUCCESS].
    [21/25] cq02-orp-app0287.cq02  [SUCCESS].
    [22/25] cq02-orp-app0212.cq02  [SUCCESS].
    [23/25] jx-orp-app17.jx  [SUCCESS].
    [24/25] ai-orp-app10.ai01  [SUCCESS].
    [25/25] db-orp-app17.db01  [SUCCESS].

### 超时killed进程

    work@yf-orp-apollo.yf01:godir$ ./pdo -a orptest -t 1s -o xxxout/t 1s -o xxxout/ -r 3 "cat "og/ral-zoo.log"
    >>>> Welcome zhangjian12...
    yf-orp-app01.yf01        -/home/work/orp001    yf-orp-app02.yf01        -/home/work/orp001
    yf-orp-app03.yf01        -/home/work/orp001    yf-orp-app04.yf01        -/home/work/orp001
    yf-orp-app00.yf01        -/home/work/orp001    yf-orp-app0148.yf01      -/home/work/orp004
    dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app17.m1          -/home/work/orp001
    m1-orp-app25.m1          -/home/work/orp001    m1-orp-app0220.m1        -/home/work/orp001
    m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
    cq01-orp-app0179.cq01    -/home/work/orp001    cq01-orp-app0131.cq01    -/home/work/orp005
    st01-orp-app03.st01      -/home/work/orp001    st01-orp-app04.st01      -/home/work/orp001
    st01-orp-app02.st01      -/home/work/orp001    st01-orp-app00.st01      -/home/work/orp001
    st01-orp-app05.st01      -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
    cq02-orp-app0287.cq02    -/home/work/orp001    cq02-orp-app0211.cq02    -/home/work/orp001
    cq02-orp-app0212.cq02    -/home/work/orp001    jx-orp-app17.jx          -/home/work/orp001
    ai-orp-app10.ai01        -/home/work/orp001    db-orp-app17.db01        -/home/work/orp003


    #--Total--#  26
    #---CMD---#  cat log/ral-zoo.log
    Continue (y/n):y
    go on ...
    [1/26] yf-orp-app01.yf01  [Time Over KILLED].
    Continue (y/n):y
    go on ...
    [2/26] yf-orp-app04.yf01  [Time Over KILLED].
    [3/26] yf-orp-app03.yf01  [Time Over KILLED].
    [4/26] yf-orp-app02.yf01  [Time Over KILLED].
    [5/26] yf-orp-app0148.yf01  [SUCCESS].
    [6/26] dbl-orp-app0109.dbl01  [SUCCESS].
    [7/26] yf-orp-app00.yf01  [Time Over KILLED].
    [8/26] m1-orp-app0220.m1  [SUCCESS].
    [9/26] m1-orp-app25.m1  [Time Over KILLED].
    [10/26] m1-orp-app17.m1  [Time Over KILLED].
    [11/26] m1-orp-app0154.m1  [SUCCESS].
    [12/26] cq01-orp-app0242.cq01  [SUCCESS].
    [13/26] cq01-orp-app0179.cq01  [Time Over KILLED].
    [14/26] cq01-orp-app0131.cq01  [SUCCESS].
    [15/26] st01-orp-app03.st01  [Time Over KILLED].
    [16/26] st01-orp-app04.st01  [Time Over KILLED].
    [17/26] st01-orp-app02.st01  [Time Over KILLED].
    [18/26] st01-orp-app00.st01  [Time Over KILLED].
    [19/26] st01-orp-app05.st01  [Time Over KILLED].
    [20/26] cq02-orp-app0211.cq02  [SUCCESS].
    [21/26] cq02-orp-app0258.cq02  [SUCCESS].
    [22/26] cq02-orp-app0287.cq02  [SUCCESS].
    [23/26] ai-orp-app10.ai01  [SUCCESS].
    [24/26] cq02-orp-app0212.cq02  [SUCCESS].
    [25/26] jx-orp-app17.jx  [Time Over KILLED].
    [26/26] db-orp-app17.db01  [SUCCESS].
    
### -c copy文件

    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -3  | pdo -c get.sh /tmp/
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
    
    [3/3] yf-orp-upload02.yf01  [SUCCESS].
    
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

### Retry功能

这次多加两台服务器,有两台是没有这个脚本文件的.

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
 
 ### -e脚本执行功能

    work@yf-orp-apollo.yf01:upload_server$ cat t.sh
    #!/bin/bash

    cd /tmp/ && pwd
    echo "test"
    touch /tmp/t.log
    
    work@yf-orp-apollo.yf01:upload_server$ get_instance_by_service picupload.orp.all | head -3  | pdo -e t.sh
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


### 模板功能

模板功能主要是解决重复的脚本修改动作,可以固化成一些模板,直接使用.

* 配置中可以自己添加模板

        work@yf-orp-apollo.yf01:godir$ cat ~/.pdo/pdo.conf
        [TEMPLATE]
        container : /home/work/.pdo/template/container.sh
        
* 模板内容,这个模版主要是在一台服务器上的orpxxx目录里面进行操作. {{.CMD}} 就是会被替换的位置.

        work@yf-orp-apollo.yf01:godir$ cat /home/work/.pdo/template/container.sh
        #!/bin/bash
        grep -l "^appName:" /home/work/orp[0-9][0-9][0-9]/orp.conf | while read file  ; do
        eval $(awk '{if($1 ~ /orpPath/){printf "apppath=%s\n",$2};if($1 ~ /appName/){printf "appName=%s",$2}}' $file)
        echo $appName
        if [  -d "$apppath" ];then
                cd $apppath
        
                    {{.CMD}}
        
        fi
        done
        
* 使用嵌入命令
 
            work@yf-orp-apollo.yf01:godir$ pdo -a orptest -temp container "pwd"
            >>>> Welcome zhangjian12...
            yf-orp-app02.yf01        -/home/work/orp001    yf-orp-app03.yf01        -/home/work/orp001
            yf-orp-app00.yf01        -/home/work/orp001    yf-orp-app0148.yf01      -/home/work/orp004
            dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app0220.m1        -/home/work/orp001
            m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
            cq01-orp-app0179.cq01    -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
            cq02-orp-app0287.cq02    -/home/work/orp001    cq02-orp-app0211.cq02    -/home/work/orp001
            jx-orp-app17.jx          -/home/work/orp001    db-orp-app17.db01        -/home/work/orp003


            #--Total--#  14
            #---CMD---#  pwd
            Continue (y/n):y
            go on ...
            [1/14] yf-orp-app02.yf01  [SUCCESS].
            orptest
            /home/work/orp001
            jingyan
            /home/work/orp002
            pc_anti
            /home/work/orp003
            bakan
            /home/work/orp004
            smallapp
            /home/work/orp006
            appui
            /home/work/orp008
            
            Continue (y/n):n
            exit ...

* 还可以嵌入脚本

        //脚本内容
        work@yf-orp-apollo.yf01:godir$ cat 1.sh
        
        echo "1.sh"
        pwd
       
        //嵌入脚本使用-b
        work@yf-orp-apollo.yf01:godir$ pdo -a orptest -temp container -b 1.sh
        >>>> Welcome zhangjian12...
        yf-orp-app02.yf01        -/home/work/orp001    yf-orp-app03.yf01        -/home/work/orp001
        yf-orp-app00.yf01        -/home/work/orp001    yf-orp-app0148.yf01      -/home/work/orp004
        dbl-orp-app0109.dbl01    -/home/work/orp003    m1-orp-app0220.m1        -/home/work/orp001
        m1-orp-app0154.m1        -/home/work/orp004    cq01-orp-app0242.cq01    -/home/work/orp003
        cq01-orp-app0179.cq01    -/home/work/orp001    cq02-orp-app0258.cq02    -/home/work/orp001
        cq02-orp-app0287.cq02    -/home/work/orp001    cq02-orp-app0211.cq02    -/home/work/orp001
        jx-orp-app17.jx          -/home/work/orp001    db-orp-app17.db01        -/home/work/orp003
        
        
        #--Total--#  14
        #---CMD---#
        Continue (y/n):y
        go on ...
        [1/14] yf-orp-app02.yf01  [SUCCESS].
        orptest
        1.sh
        /home/work/orp001
        jingyan
        1.sh
        /home/work/orp002
        pc_anti
        1.sh
        /home/work/orp003
        bakan
        1.sh
        /home/work/orp004
        smallapp
        1.sh
        /home/work/orp006
        appui
        1.sh
        /home/work/orp008
        
###  行显示与匹配

这个功能有两种使用场景:

1. 有点类似multi tail 可以实现同时tail多个日志,显示在一个屏幕内,而且可以对match的字符串进行高亮显示.
2. 如果输出是单行输出,没有状态显示会显示得加的美观和可参考性.

所以这种显示方式取决于时间的先后顺序,交错输出.

拿redis的迁移过程为例子: 

> redis迁移至少有原来的一主一从,新主和新从.在迁移的过程中需要同时观察四台服务器的变化.如果是每次ssh四台服务器tail 日志是很麻烦而且容易出错.

现在使用pdo命令:
        
        //操作的主机列表1.list
        tc-arch-redis40.tc /home/arch/redis-ting-listen-shard3   //old master 
        cq02-arch-redis80.cq02 /home/arch/redis-ting-listen-shard3 //new master 
        yf-arch-redis40.yf01 /home/arch/redis-ting-listen-shard3 //old slave 
        jx-arch-redis80.jx /home/arch/redis-ting-listen-shard3  //new slave 
        第一步操作:  yf-arch-redis40.yf01为主 --> cq02-arch-redis80.cq02 

        #命令
        #cat 1.list | pdo -r 5 -y -show row  -match "success" "tail -f log/redis.log"
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

说明:
 
1.  因为是tail -f 是不会主动退出命令,所以需要使用-y 和使用-r 来增加并发量,不然会先进行单台显示 ,而不会显示后面的.
2.  match是匹配字符 串,暂时不支持正则,会进行高亮显示.红色显示.
3.  -show现在只支持row这一种方式,默认方式还是原来的缓存输出方式.  

以下是一个测试脚本:随机打印数字 1.sh

        #!/bin/bash
        for x in `seq 1 10` ; do
            echo $x
            sleep $[ ( $RANDOM % 4 )  + 1 ]s
        done 
        
         //可以使用如下命令:
       # cat 1.list | pdo -r 5 -y -show row  -match "5" -e 1.sh

还有更多的组合,可以找实验.
