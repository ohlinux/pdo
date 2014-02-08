#!/bin/bash

grep -l "^appName:" /home/work/orp[0-9][0-9][0-9]/orp.conf | while read file  ; do

eval $(awk '{if($1 ~ /orpPath/){printf "apppath=%s\n",$2};if($1 ~ /appName/){printf "appName=%s",$2}}' $file)

if [  -d "$apppath" ];then
    cd $apppath

    if  ! ` ps aux | grep "$apppath" | grep php-cgi > /dev/null ` ; then
    echo $appName
    {{.CMD}}
    fi
fi

done
