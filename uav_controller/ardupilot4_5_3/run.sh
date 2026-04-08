#!/bin/bash

# Run arducopter in the background
./arducopter4_5_3 -S --base-port 5760 --home ${UAV_HOME_LOCATION:-"39.482594,-0.346265,0.0,0.0"} --model + --defaults copter.parm --sim-port-in 5501 --sim-port-out 5502 --rc-in-port 5503 --irlock-port 9005 &

sleep 5 #wit for arducopter to boot
# Run the Java application
java -jar ./uavController.jar config.json &

# Wait for all background processes to finish
wait
