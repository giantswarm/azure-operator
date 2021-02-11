package ignition

const PythonExeLinkUnit = `[Unit]
Description=Link the Azure-provided python executable to make it available to MMA extension
[Service]
Type=oneshot
ExecStart=/usr/bin/ln -s /usr/share/oem/python/bin/python /opt/bin/python2
[Install]
WantedBy=multi-user.target
`
