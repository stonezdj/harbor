FROM goharbor/harbor-e2e-engine:latest-api
RUN pip3 install xmlrunner && tdnf install nodejs -y && npm install -g junit-viewer
RUN mkdir /app 
COPY ./openapi-generator-cli.jar /app
COPY ./Makefile   /app
COPY ./run_api.sh /app
COPY ./api.py     /app

