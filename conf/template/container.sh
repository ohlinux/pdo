#!/bin/bash

grep -l "^appName:" /home/work/orp[0-9][0-9][0-9]/orp.conf | while read file  ; do

eval $(awk '{if($1 ~ /orpPath/){printf "apppath=%s\n",$2};if($1 ~ /appName/){printf "appName=%s",$2}}' $file)

echo $appName
if [  -d "$apppath" ];then
    cd $apppath

    {{.CMD}}

fi

done
