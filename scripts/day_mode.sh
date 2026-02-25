#!/bin/bash
# Jour - Performance ondemand
vcgencmd display_power 1
cpupower frequency-set -g ondemand || echo "ondemand" > /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
logger "Day mode activated: HDMI ON, CPU OnDemand"
