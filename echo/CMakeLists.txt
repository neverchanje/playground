

add_executable(server server.cpp)
target_link_libraries(server ${SILLY_LIBRARY})

add_executable(client client.cc)
target_link_libraries(client ${SILLY_LIBRARY})
