FROM docker

ENV GOPATH /go 
ENV PATH $GOPATH/bin:/usr/local/go/bin:/usr/local/bin:/go/src/sigs.k8s.io/kubefed/bin:$PATH

RUN wget https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz  && \
    tar -xvf go1.12.6.linux-amd64.tar.gz && \
    mv go /usr/local && \
    rm -rf go1.12.6.linux-amd64.tar.gz
WORKDIR /go/src/sigs.k8s.io/kubefed
# RUN GO111MODULE="on" go get sigs.k8s.io/kind@v0.3.0

# COPY Makefile Makefile
# COPY pkg pkg
# COPY cmd cmd 
# COPY test test
# COPY scripts scripts
# COPY vendor vendor

#RUN ./scripts/download-binaries.sh 