#!/bin/sh

# Ensure log directory exists if not mounted
mkdir -p /app/logs

# Run arducopter in the background, redirecting output to logs
${ARDUPILOT_INSTANCE:-./arducopter4_5_3} -S --base-port 5760 --home ${UAV_HOME_LOCATION:-"39.482594,-0.346265,0.0,0.0"} --model + --defaults copter.parm --sim-port-in 5501 --sim-port-out 5502 --rc-in-port 5503 --irlock-port 9005 > /app/logs/arducopter.log 2>&1 &

sleep 5 # wait for arducopter to boot

# Run the Java application, redirecting output to logs
java -Ddebug=${DEBUG:-false} -jar ./uavController.jar config.json > /app/logs/uav_controller.log 2>&1 &

# Wait for all background processes to finish
wait
