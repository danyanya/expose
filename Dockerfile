FROM centurylink/ca-certs

ADD expose /

ENTRYPOINT [ "/expose" ]
