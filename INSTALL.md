## Installation
It's not required to install, you can run directly from console and the proxy will work right after compiling, using standard solana public nodes.

## Installing as a service
- Create a new user, it can be called solproxy
- Copy the content of gosol/main into /home/solproxy (it needs to contain compiled sources) or just download a package from this website and put everything under /home/solproxy
- Create a directory /home/solproxy/log owned by solproxy user
- Run sudo vi /etc/systemd/system/solproxy.service and place the following content into solproxy.service file

<pre>[Unit]
After=network-online.target
Wants=network-online.target
Description=Solana Proxy Service

[Service]
User=solproxy
LimitNOFILE=524288
LimitMEMLOCK=1073741824
LimitNICE=-10
Nice=-10
ExecStart=/bin/sh -c 'cd /home/solproxy; export GODEBUG=gctrace=1; started=`date --rfc-3339=seconds`; echo Starting Solana Proxy $started; ./main 1>"log/log-$started.txt" 2>"log/error-$started.log.txt";'
Type=simple
PrivateNetwork=false
PrivateTmp=false
ProtectSystem=false
ProtectHome=false
KillMode=control-group
Restart=always
DefaultTasksMax=65536
TasksMax=65536
RestartSec=30
StartLimitIntervalSec=200
StartLimitBurst=10

[Install]
WantedBy=multi-user.target</pre>
- Run systemctl daemon-reload
- Run systemctl enable worker-node

The proxy should be now running
