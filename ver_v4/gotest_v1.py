#!/usr/bin/python

import os
import _thread
import time
from mininet.net import Containernet
from mininet.node import Controller
from mininet.cli import CLI
from mininet.link import TCLink
from mininet.log import info, setLogLevel

'''

SET NETWORK TOPOLOGY

'''

setLogLevel('info')

net = Containernet(controller=Controller)

info('*** Adding controller\n')
net.addController('c0')

info('*** Adding docker containers\n')
valclient = net.addDocker('valclient', ip='172.19.0.97', dimage="valclient:latest") # 172.17.0.2
val = net.addDocker('val', ip='172.19.0.98', dimage="val_node:latest") # 172.17.0.3
makepath = net.addDocker('makepath', ip='172.19.0.99', dimage="omtree_mesh:latest") # 172.17.0.4

# s = net.addSwitch('s0')

# net.addLink(s, val)
# net.addLink(s, valclient)
# net.addLink(s, makepath)

N = 80 
d = []
for i in range(N):
    d.append(net.addDocker('d%s'%i, ip='172.19.0.%s'%(100+i), dimage="pr_node:latest")) # 172.17.0.5 ~ 172.17.0.84

info('*** Adding switches\n')
S = 11
s = []
for i in range(S):
    s.append(net.addSwitch('s%s'%i))

info('*** Creating links\n')
bw = 50 
net.addLink(s[0], s[2], cls=TCLink, delay='2ms', bw=1000)
net.addLink(s[1], s[2], cls=TCLink, delay='4ms', bw=1000) #v3,5 => 4, v6 => 10
net.addLink(s[2], s[5], cls=TCLink, delay='8ms', bw=1000)
net.addLink(s[2], s[4], cls=TCLink, delay='10ms', bw=1000)
net.addLink(s[4], s[6], cls=TCLink, delay='7ms', bw=1000)
net.addLink(s[3], s[6], cls=TCLink, delay='12ms', bw=1000)
net.addLink(s[6], s[7], cls=TCLink, delay='6ms', bw=1000) #v3 => 3,  v5 => 10, 
net.addLink(s[4], s[8], cls=TCLink, delay='8ms', bw=1000) #v3,5,6 => 8, v7 => 15
net.addLink(s[6], s[9], cls=TCLink, delay='5ms', bw=1000)
net.addLink(s[4], s[10], cls=TCLink, delay='3ms', bw=1000)

net.addLink(s[0], val)
net.addLink(s[0], makepath)

net.addLink(s[0], d[0])

'''
#net.addLink(s[], d[])
'''

for i in range(1, N): 
    if i % 8== 0:
        net.addLink(s[1], d[i])
    elif i % 8 == 1 :
        net.addLink(s[9], d[i])
    elif i % 8 == 2:
        net.addLink(s[8], d[i])
    elif i % 8 == 3 :
        net.addLink(s[5], d[i])
    elif i % 8 == 4 :
        net.addLink(s[3], d[i])
    elif i % 8 == 5 :
        net.addLink(s[7], d[i])
    elif i % 8 == 6 :
        net.addLink(s[10], d[i])
    elif i % 8 == 7 :
        net.addLink(s[0], d[i])

'''
for i in range(0, 20):
    net.addLink(s[0], d[i])

for i in range(20, 25):
    net.addLink(s[1], d[i])

for i in range(25, 29):
    net.addLink(s[4], d[i])

for i in range(29, 36):
    net.addLink(s[5], d[i])

for i in range(36, 48):
    net.addLink(s[7], d[i])

for i in range(48, 51):
    net.addLink(s[8], d[i])

for i in range(51, 60):
    net.addLink(s[9], d[i])
'''

def run_verclient():
    os.system('docker exec mn.valclient ./valclient > result/log/hp2b/val_client/valclient')
    info('*** start valclient ***\n')

_thread.start_new_thread(run_verclient, ())
info('@@@@@@@@@@@@@@@ run_verclient')
    
info('*** Starting network\n')
net.start()

info('*** Testing connectivity\n')
net.ping([val, d[0]])
net.ping([d[0], d[1]])


'''

EXECUTE PEER NODES

'''

def exec_nodes(i) :
    print(i,"th node")
    
    # os.system('docker exec mn.d%s ./pr_node 172.19.0.%s 172.19.0.98'%(i, 100+i)) # run peer on the foreground
    os.system('docker exec mn.d%s ./myprogram 172.19.0.%s 172.19.0.98 > result/log/hp2b/mn_d/%s_log.txt'%(i, 100+i, i)) # run peer on the foreground

    # os.system('docker -d exec mn.d%s ./pr_node 172.19.0.%s 172.19.0.98'%(i, 100+i)) # run peer on the background
    #  <-- if you don't want to see peer-side printed log and make peer run on the background, you can use this code instead of a line above.
    #       if so, you can't make any log on logfiles when it's running on background. so you can erase code on line 113, 114.

for i in range(N):
    _thread.start_new_thread(exec_nodes, (i, ))

'''

EXECUTE VALIDATOR NODE

'''

# WAIT UNTILL ALL PEER NODE IS EXECUTED 
time.sleep(10)

# YOU CAN TEST MANY VALIDATORS WITH DIFFERENCE DIMENSION & REFERENCE NUMBER, AS LONG AS THERE IS PEER NODE ALIVE.

#defualt
# D = list(range(3,6)) # dimension = [3, 4, 5]
# peer_ref = 7 # the number of reference points for each peer
# sck_ref = 7 # the number of reference points for self-cross-check

D = list(range(4,5)) # dimension = [3, 4, 5]
peer_ref = 7 # the number of reference points for each peer
sck_ref = 7 # the number of reference points for self-cross-check

for dim in D: 
#    print(dim , sck_ref)
#    _thread.start_new_thread((lambda : os.system('touch result/log/val_log_%s-%s-%s.txt'%(dim, peer_ref, sck_ref)) ), ( )) # create validator's log file
#    _thread.start_new_thread((lambda : os.system('tail -f result/log/val_log_%s-%s-%s.txt'%(dim, peer_ref, sck_ref)) ), ( )) # track & follow validator's printed log
#    os.system('docker exec mn.val ./val_node %s %s %s > result/log/val_log_%s-%s-%s.txt'%(dim, peer_ref, sck_ref, dim, peer_ref, sck_ref)) # execute validator
    _thread.start_new_thread((lambda : os.system('touch result/log/val_log_%s-%s-%s.txt'%(dim, dim+1, dim+1)) ), ( )) # create validator's log file
    _thread.start_new_thread((lambda : os.system('tail -f result/log/val_log_%s-%s-%s.txt'%(dim, dim+1, dim+1)) ), ( )) # track & follow validator's printed log
    os.system('docker exec mn.val ./val_node %s %s %s > result/log/val_log_%s-%s-%s.txt'%(dim, dim+1, dim+1, dim, dim+1, dim+1)) # execute validator


info('*** Running CLI\n')
CLI(net)
info('*** Stopping network')
net.stop()

 