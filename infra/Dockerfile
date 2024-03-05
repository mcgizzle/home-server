# Use Debian Buster as base image
FROM debian:buster

# Install SSH
RUN apt-get update && apt-get install -y openssh-server sudo python3

# Make sure Python 3 is available as `python`
RUN ln -s /usr/bin/python3 /usr/bin/python

RUN mkdir /var/run/sshd

# Set up root password (optional, if you want an additional way to log in)
#RUN echo 'root:YOUR_PASSWORD' | chpasswd

# Copy the public key to authorized_keys
COPY id_ed25519.pub /root/.ssh/authorized_keys

# Correct permissions for .ssh and authorized_keys
RUN chmod 700 /root/.ssh && chmod 600 /root/.ssh/authorized_keys

RUN service ssh start

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]

