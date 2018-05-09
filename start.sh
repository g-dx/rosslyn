#!/bin/bash

PROGRAM="mate-terminal -e /home/g-dx/Workspaces/Go/bin/rosslyn --profile=Rosslyn --hide-menubar --window --geometry=100x100"
WINID=$(wmctrl -lx | grep -i "Rosslyn v0.0.1" | awk 'NR==1{print $1}')

if [ $WINID ]; then
    wmctrl -ia $WINID &
 #  exit 0
else
    $PROGRAM &
 #  exit 0
fi
