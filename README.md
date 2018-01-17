# PDO批量工具

 pdo是一个批量执行远程服务器指令的工具,非常的灵活和简便. 不依赖agent,只通过ssh就可以工作,可以使用key的方式也可以通过用户名和密码.而且做了一些优化和控制,可以解决日常中很多的需求.
 pdo已经经历了两个版本的迭代,这次是重构的第三版增加了很多额外控制和代码的优化.

### 功能

* [x] 直接执行命令
* [x] 本地命令执行
* [x] 远程命令执行
* [x] 复制命令
* [x] 返回错误号
* [x] 输出多种格式
* [x] 可以指定用户密码或者私钥
* [ ] 列计算
* [ ] diff
* [ ] md5
* [ ] 输入机器列表正则
* [ ] 输入列表优化列表 在相同的主机不同的path的情况下.
* [ ] 彩色终端效果.
* [ ] 安全命令过滤.
* [ ] 日志信息增强.
* [ ] color自定义
* [x] 支持tail -f 的命令关闭
* [x] 支持自定义询问间隔
* [x] 支持超时,并发度,间隔等控制功能.

### help

```
PDO is a simple tool that parallel do something using ssh , no agent.

Usage:

  pdo  [flags]

  pdo <input control> [thread control] [output control] [subcommand] <function> [Argument]

Available Commands:
  cal         A brief description of your command
  console     A brief description of your command
  copy        copy file or directory to destination
  help        Help about any command
  md5         A brief description of your command
  print       A brief description of your command
  script      A brief description of your command
  setup       A brief description of your command
  tail        A brief description of your command
  version     Print the version number of pdo

Flags:
      --ask int                  thread control,ask every numbers,default is 0 means no ask expect the first one
      --auth-knownHosts string   authentication ,specified known_hosts file (default "/.ssh/known_hosts")
  -p, --auth-passwd string       authentication ,specified password
      --auth-priv string         authentication ,specified private key file (default "/.ssh/id_rsb")
  -u, --auth-user string         authentication ,specified user name
  -r, --concurrent int           thread control , concurrent processing (default 1)
  -c, --config string            config file (default "/.pdo/pdo.yml")
      --header                   output header information  (default true)
  -h, --help                     help for pdo
  -f, --in-file string           input from file
  -F, --in-format string         input format eg:json yaml  (default "row")
  -R, --in-last-fail             input from last failure list
  -S, --in-last-success          input from last success list
  -E, --in-regex string          input host regex to filter
      --log-backup int           log ,log backup number files (default 7)
      --log-level string         log , output log level (default "info")
      --log-maxfile int          log , max value for each log file (default 10000)
      --log-path string          log ,log output directory (default "./logs")
      --out string               output format eg: json yaml row  (default "text")
  -o, --out-directory string     output save in directory
      --out-file string          output save to new file
      --out-file-append string   output append to a file
      --out-nocolor              output no color
      --out-regex string         output highlight by regular
  -q, --quiet                    output quiet mode , no summary and header information
      --summary                  output summary information (default true)
  -T, --time-inter duration      thread control,interval time between concurrent jobs
  -t, --time-over duration       thread control,over the time ,kill process. (default 5m0s)
  -y, --yes                      thread control,input yes when ask

Examples:
  cat host.list | pdo -r 10 "pwd"

Use "pdo [command] --help" for more information about a command.

```

### 结构

v2

```
Usage: pdo2 [input control][thread control] [output control][subcommand] <content>
```

v3

```
Usage: pdo2 <input control> [thread control] [output control] [subcommand] <function> [Argument]
```


#### input control:

```
    -f  --in-file <file>    from File name.
    -R  --in-last-fail      from last failure list.
     	--in-last-success		  from last Success list.
    -P 	--in-plugin <plugin name> <args> from plugin list
    default             from pipe,eg: cat file | pdo2
    --in-format      yaml or json,default row by row .
    -E   --in-regex         机器列表输入正则
```


#### output control:


```
    default             display after finish.
    --show <option> --out <option>
    							Print the output from the
    							ssh command using the
								specified outputter. The
								builtins are 'key', 'yaml',
                        'overstatestage',
                        'newline_values_only',
                        'txt', 'raw',
                        'no_return', 'virt_query',
                        'compact', 'json',
                        'highstate', 'nested',
                         'quiet', 'pprint'.
    -o  --out-directory=OUTPUT_DIRECTORY         save to directory for every host.
    --out-file=OUTPUT_FILE
                        Write the output to the specified file.
    --out-file-append,
                        Append the output to the specified file.
    --out-regex         Display output  highlight by regular
    --out-no-color      Disable all colored output.
    --summary 			汇总信息 默认打开
    --term  		   彩色终端
    -q    --quiet   quiet mode ,no summary and no header information.
    --header        头部信息 默认打开
```


#### thread control:

```
    -r <int>            concurrent processing ,default 1.
    -t <10s>            over the time ,kill process.
    -T <1m>             Interval time between concurrent programs.
    -y                  default need input y at the first host.
    --ask  <int>        ask every numbers.default no ask .
```

#### configure:

```
-c --config <file>   specified configure file . default ~/.pdo/pdo.conf
```

#### subcommand :

##### Operation subcommand

```
    copy <file> <destination>   copy file to remote host.
    script <script file> <args>     execute script file on remote host.
    cmd <config shortcmd>       use short cmd in the pdo.conf.
    tailf <file>                  tail -f file . display row by row ,support ctrl+c kill remote tail command
    console         enter in console interface
```

##### Summary subcommand

```
    md5 <file>               md5sum file and count md5.
    cal <sum> <avg1> 			Column calculation  sum avg
```

#### basic subcommand

```
    help <subcommand>           get subcommand help.
    print                       print host list. hostname path.
    version                     get the pdo version.
    setup  <directory>          setup the configuration at the first time .default  directory ~/.pdo
```
#### Authentication Options:

```
    --auth-priv=SSH_PRIV     set ssh private key file ,default ~/.ssh/id_rsa
    -u --auth-user=SSH_USER     set the default user to attempt to use when authenticating, default current user
    --auth-passwd=SSH_PASSWD
                        set the default password to attempt to use when authenticating default null, if set password to instand of privet key
```

#### OUTPUT CONTENT:

```
web249:
    ----------
    retcode:
        0
    stderr:
    stdout:
        abc
```

