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
#include <silly/UnixTimestamp.h>

class ClientSocket {

 public:

  static ClientSocket Create(const std::string &address) {
    int fd;
    struct sockaddr_in addr;

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd == -1) {
      throw std::runtime_error("socket");
    }

    bzero(&addr, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(10000);

    inet_pton(AF_INET, address.c_str(), &addr.sin_addr);
    return ClientSocket(addr, fd);
  }

  ClientSocket(const struct sockaddr_in &addr, int fd) :
      serv_(addr), fd_(fd) {
  }

  ~ClientSocket() {
    close(fd_);
  }

  void Connect() {
    int r = connect(fd_, (struct sockaddr *) &serv_, sizeof(serv_));
    if (r < 0) throw std::runtime_error("connect");
  }

  void CalRoundTime() {
    auto now = std::to_string(silly::UnixTimestamp::Now().MicrosSinceEpoch());
    write(fd_, now.c_str(), now.length());
    std::cout << "sending: " << now << std::endl;

    ssize_t n = read(fd_, recv_, 1024);
    if (n < 0) {
      std::cerr << "Nothing read" << std::endl;
      return;
    }
    recv_[n] = '\0';

    char *pEnd;
    errno = 0;
    auto time1 = strtoll(recv_, &pEnd, 10);
    auto time2 = silly::UnixTimestamp::Now().MicrosSinceEpoch();
    std::cout << recv_ << std::endl;
    std::cout << time2 << std::endl;
    std::cout << "difference: " << (time2 - time1) << std::endl;
  }

  void Write() {
    auto now = silly::UnixTimestamp::Now().ToString();
    write(fd_, now.c_str(), now.length());
  }

  const char *RawData() const {
    return recv_;
  }

 private:
  struct sockaddr_in serv_;
  int fd_;
  char recv_[1024];
};

int main() {
  try {
    ClientSocket client = ClientSocket::Create("127.0.0.1");
    client.Connect();
    client.CalRoundTime();
  } catch (const std::exception &e) {
    std::cerr << e.what() << std::endl;
  }
  return 0;
}