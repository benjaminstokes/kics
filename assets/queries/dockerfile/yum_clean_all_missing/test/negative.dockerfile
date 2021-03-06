FROM alpine:3.5
RUN apk add --update py2-pip
RUN yum install \
    yum clean all
COPY requirements.txt /usr/src/app/
RUN pip install --no-cache-dir -r /usr/src/app/requirements.txt
COPY app.py /usr/src/app/
COPY templates/index.html /usr/src/app/templates/
EXPOSE 5000
CMD ["python", "/usr/src/app/app.py"]

FROM alpine:3.4
RUN yum -y install \
    yum clean all
