#!/bin/bash

# Vérification des privilèges
if [[ $EUID -ne 0 ]]; then
   echo "Ce script doit être exécuté en tant que root (sudo)."
   exit 1
fi

CONFIG_FILE="/boot/firmware/config.txt"
[ ! -f "$CONFIG_FILE" ] && CONFIG_FILE="/boot/config.txt"

echo "--- Configuration des LED dans $CONFIG_FILE ---"

# Ajout des paramètres via un bloc dédié pour éviter les doublons
MARKER="## LED_CONTROL_SECTION"
if ! grep -q "$MARKER" "$CONFIG_FILE"; then
    tee -a "$CONFIG_FILE" <<EOFC

$MARKER
# Désactivation LED Power et Activité
dtparam=pwr_led_trigger=default-on
dtparam=pwr_led_activelow=off
dtparam=act_led_trigger=none
dtparam=act_led_activelow=off
# Désactivation LED Ethernet (RP1)
dtparam=eth_led0=4
dtparam=eth_led1=4
$MARKER
EOFC
    echo "Paramètres ajoutés au config.txt."
else
    echo "Section de configuration déjà présente."
fi

echo -e "\n--- Configuration de l'EEPROM ---"

# Mise à jour de l'EEPROM pour la gestion de l'alimentation
CURRENT_CONF=$(rpi-eeprom-config)
NEW_CONF=$(echo "$CURRENT_CONF" | sed 's/^POWER_OFF_ON_HALT=.*/POWER_OFF_ON_HALT=1/')
NEW_CONF=$(echo "$NEW_CONF" | sed 's/^WAKE_ON_GPIO=.*/WAKE_ON_GPIO=0/')

# Si les paramètres n'existaient pas, on les ajoute
[[ "$NEW_CONF" != *"POWER_OFF_ON_HALT"* ]] && NEW_CONF+=$'\nPOWER_OFF_ON_HALT=1'
[[ "$NEW_CONF" != *"WAKE_ON_GPIO"* ]] && NEW_CONF+=$'\nWAKE_ON_GPIO=0'

echo "$NEW_CONF" > /tmp/eeprom_new.conf
rpi-eeprom-config --apply /tmp/eeprom_new.conf

echo -e "\n--- Diagnostic NVMe (LED Bleue) ---"

if command -v nvme &> /dev/null; then
    DEVICE=$(ls /dev/nvme0n1 2>/dev/null)
    if [ -n "$DEVICE" ]; then
        echo "Vérification du support de gestion d'énergie (APST) pour minimiser l'activité bus :"
        nvme smart-log "$DEVICE" | grep -E "power_state|critical_warning"
        echo "Note: Si le SSD supporte des états profonds (PS3/PS4), la LED s'éteindra au repos."
    fi
else
    echo "nvme-cli non installé. Diagnostic NVMe sauté."
fi

echo -e "\nTerminé. Un redémarrage est nécessaire pour appliquer les changements."
