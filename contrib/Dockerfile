FROM debian:stretch
MAINTAINER Markus Lindenberg <markus@lindenberg.io>

ENV DEBIAN_FRONTEND noninteractive
ENV DEBIAN_PRIORITY critical
ENV DEBCONF_NOWARNINGS yes
RUN echo 'APT::Get::Assume-Yes "true";' > /etc/apt/apt.conf.d/90assumeyes
RUN echo 'APT::Get::Install-Recommends "false";\nAPT::Get::Install-Suggests "false";' > /etc/apt/apt.conf.d/90norecommends

RUN apt-get update && apt-get install \
	git build-essential devscripts sudo fakeroot equivs lsb-release quilt dh-autoreconf lintian \
	&& apt-get clean

RUN mkdir /input /output
RUN adduser --system --group --home /build build

ADD build.sh /usr/local/sbin/build.sh
RUN chmod a+x /usr/local/sbin/build.sh
CMD ["/usr/local/sbin/build.sh"]
