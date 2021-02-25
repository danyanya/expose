FROM centurylink/ca-certs

ADD ./expose /usr/bin/expose

ENTRYPOINT [ "/usr/bin/expose" ]
