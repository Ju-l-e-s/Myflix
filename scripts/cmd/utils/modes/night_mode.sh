#!/bin/bash
# Nuit - Économie & Silence
vcgencmd display_power 0
cpupower frequency-set -g powersave || echo "powersave" > /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
hdparm -S 120 /dev/sda
logger "Night mode activated: HDMI OFF, CPU PowerSave, HDD Sleep in 10m"
