import subprocess
import time


# a.go를 실행하는 명령어
#command_a = "go run .\\tcpserver\\tcpserver_multi_session_manager\\server_goroutin\\tcpsever_multi_session_manager.go 127.0.0.1 8000"

# 참여노드 3개 실행 명령어
command_a = "go run ./parti_node/tcp_peer_servent.go ./parti_node/client_module.go ./parti_node/mcn_module.go 127.0.0.1 8000" #매개변수는 참여노드의 Ip port (2개)
command_b = "go run ./parti_node/tcp_peer_servent.go ./parti_node/client_module.go ./parti_node/mcn_module.go 127.0.0.1 8001"
command_c = "go run ./parti_node/tcp_peer_servent.go ./parti_node/client_module.go ./parti_node/mcn_module.go 127.0.0.1 8002"
command_d = "go run ./parti_node/tcp_peer_servent.go ./parti_node/client_module.go ./parti_node/mcn_module.go 127.0.0.1 8003"


# 검증노드 실행 명령어
command_ver = "go run ./verify_node/tcpclient.go" 

# 참여노드 실행
process_a = subprocess.Popen(command_a, shell=True)
process_b = subprocess.Popen(command_b, shell=True)
process_c = subprocess.Popen(command_c, shell=True)
process_d = subprocess.Popen(command_d, shell=True)

time.sleep(2)  # Sleep for 2 seconds

# 검증노드 실행
print("start verificaiton node") 
process_ver = subprocess.Popen(command_ver, shell=True)


# 각 프로세스의 종료를 기다림
process_a.wait()
process_b.wait()
process_c.wait()
process_d.wait()
process_ver.wait()
