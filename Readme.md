# Prerequisites

tshark

mysql

golang>=1.22.5

python=3.11

# How to Run

    cd ~
    git clone https://github.com/king0ffire/webapp.git
    git clone https://github.com/KimiNewt/pyshark.git
    cd pyshark/src
    pip setup.py install
    cd ../webapp
    pip install -r ./scripts/requirements.txt



Adapt `config.ini`, `loganalyzepythonserver.service` and `loganalyzewebapp.service` to your server

    sudo cp ./loganalyzepythonserver.service /etc/systemd/system/
    sudo cp ./loganalyzewebapp.service /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl start loganalyzewebapp.service