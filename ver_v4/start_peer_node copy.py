import argparse
import subprocess
import time


# a.go를 실행하는 명령어
#command_a = "go run .\\tcpserver\\tcpserver_multi_session_manager\\server_goroutin\\tcpsever_multi_session_manager.go 127.0.0.1 8000"

# 참여노드 3개 실행 명령어
command_a = "./pr_make_tree" #매개변수는 참여노드의 Ip port (2개)


# 검증노드 실행 명령어
command_b = "./pr_node" 

# 참여노드 실행
process_a = subprocess.Popen(command_a, shell=True)
process_b = subprocess.Popen(command_b, shell=True)

# 검증노드 실행



# 각 프로세스의 종료를 기다림
process_a.wait()
process_b.wait()
