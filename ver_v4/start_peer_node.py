import os
import argparse
import subprocess
import time

current_dir = os.path.dirname(os.path.abspath(__file__))

command_a = os.path.join(current_dir, "pr_make_tree")  # "./pr_make_tree" 대신 경로를 지정
command_b = os.path.join(current_dir, "pr_node")  # "./pr_node" 대신 경로를 지정

process_a = subprocess.Popen(command_a, shell=True)
process_b = subprocess.Popen(command_b, shell=True)

# 각 프로세스의 종료를 기다림
process_a.wait()
process_b.wait()
