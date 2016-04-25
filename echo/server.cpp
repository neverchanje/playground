/**
 * Copyright (C) 2016, Wu Tao All rights reserved.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

#include <sys/socket.h>
#include <netinet/in.h>
#include <cstring>
#include <string>
#include <arpa/inet.h>
#include <chrono>
#include <iostream>

class ServerSocket {

 public:

  static ServerSocket Create() {
    int fd;
    struct sockaddr_in addr;

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd == -1) {
      throw std::runtime_error("socket");
    }

    bzero(&addr, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(10000);
    addr.sin_addr.s_addr = htonl(INADDR_ANY);

    int r = bind(fd, (struct sockaddr *) &addr, sizeof(addr));
    if (r == -1) {
      throw std::runtime_error("bind");
    }

    r = listen(fd, 100);
    if (r == -1) {
      throw std::runtime_error(std::string("listen") + strerror(errno));
    }

    return ServerSocket(addr, fd);
  }

  ServerSocket(const struct sockaddr_in &addr, int fd) :
      addr_(addr), fd_(fd) {
  }

  virtual ~ServerSocket() {
    // file descriptor will be closed in subclasses
  }

  int Accept() {
    int r = accept(fd_, NULL, NULL);
    if (r == -1) {
      throw std::runtime_error(std::string("accept") + strerror(errno));
    }
    return r;
  }

 protected:
  struct sockaddr_in addr_;
  int fd_;
};

class EchoServer: public ServerSocket {
 public:

  explicit EchoServer(const ServerSocket &serv) :
      ServerSocket(serv) { }

  ~EchoServer() {
    close(fd_);
  }

  void Echo(int conn) {
    ssize_t n = read(conn, buf_, 1024);
    if (n < 0) {
      std::cerr << "Nothing read" << std::endl;
      return;
    }
    buf_[n] = '\0';
    std::cout << buf_ << std::endl;
    write(conn, buf_, static_cast<size_t >(n));
  }

 private:
  char buf_[1024];
};

int main() {
  try {
    EchoServer server(ServerSocket::Create());
    while (true) {
      int conn = server.Accept();
      if (conn > 0) {
        server.Echo(conn);
        close(conn);
      }
    }
  } catch (const std::exception &e) {
    std::cerr << e.what() << std::endl;
  }
  return 0;
}