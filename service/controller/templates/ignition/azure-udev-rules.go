package ignition

// Only needed until https://github.com/kinvolk/init/pull/41 is included into flatcar image.

const UdevRules = `

ACTION=="add|change", SUBSYSTEM=="block", ENV{ID_VENDOR}=="Msft", ENV{ID_MODEL}=="Virtual_Disk", GOTO="azure_disk"
GOTO="azure_end"

LABEL="azure_disk"
# Root has a GUID of 0000 as the second value
# The resource/resource has GUID of 0001 as the second value
ATTRS{device_id}=="?00000000-0000-*", ENV{fabric_name}="root", GOTO="azure_names"
ATTRS{device_id}=="?00000000-0001-*", ENV{fabric_name}="resource", GOTO="azure_names"
# Wellknown SCSI controllers
ATTRS{device_id}=="{f8b3781a-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi0", GOTO="azure_datadisk"
ATTRS{device_id}=="{f8b3781b-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi1", GOTO="azure_datadisk"
ATTRS{device_id}=="{f8b3781c-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi2", GOTO="azure_datadisk"
ATTRS{device_id}=="{f8b3781d-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi3", GOTO="azure_datadisk"
GOTO="azure_end"

# Retrieve LUN number for datadisks
LABEL="azure_datadisk"
ENV{DEVTYPE}=="partition", PROGRAM="/bin/sh -c 'readlink /sys/class/block/%k/../device|cut -d: -f4'", ENV{fabric_name}="$env{fabric_scsi_controller}/lun$result", GOTO="azure_names"
PROGRAM="/bin/sh -c 'readlink /sys/class/block/%k/device|cut -d: -f4'", ENV{fabric_name}="$env{fabric_scsi_controller}/lun$result", GOTO="azure_names"
GOTO="azure_end"

# Create the symlinks
LABEL="azure_names"
ENV{DEVTYPE}=="disk", SYMLINK+="disk/azure/$env{fabric_name}", TAG+="systemd"
ENV{DEVTYPE}=="partition", SYMLINK+="disk/azure/$env{fabric_name}-part%n", TAG+="systemd"

LABEL="azure_end"
`
