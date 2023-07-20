# -*- coding: utf-8 -*-
import matplotlib.pyplot as plt
import numpy as np
import math
import argparse
import heapq  # 우선순위 큐 구현을 위함
from sklearn.mixture import GaussianMixture
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d.proj3d import proj_transform
from mpl_toolkits.mplot3d.axes3d import Axes3D
from matplotlib.patches import FancyArrowPatch
import time
from socket import *
import struct

DIM = 4

LOCAL = 0

class PrevPathInfo:
    parent = None
    idxIndata = None
    child = None
    cluster_size = None
    gmm_labels = None

    def update(self, parent, idxIndata, child, cluster_size, gmm_labels):
        self.parent = parent
        self.idxIndata = idxIndata
        self.child = child
        self.cluster_size = cluster_size
        self.gmm_labels = gmm_labels

prev_path_info = PrevPathInfo()


class Arrow3D(FancyArrowPatch):

    def __init__(self, x, y, z, dx, dy, dz, *args, **kwargs):
        super().__init__((0, 0), (0, 0), *args, **kwargs)
        self._xyz = (x, y, z)
        self._dxdydz = (dx, dy, dz)

    def draw(self, renderer):
        x1, y1, z1 = self._xyz
        dx, dy, dz = self._dxdydz
        x2, y2, z2 = (x1 + dx, y1 + dy, z1 + dz)

        xs, ys, zs = proj_transform((x1, x2), (y1, y2), (z1, z2), self.axes.M)
        self.set_positions((xs[0], ys[0]), (xs[1], ys[1]))
        super().draw(renderer)

    def do_3d_projection(self, renderer=None):
        x1, y1, z1 = self._xyz
        dx, dy, dz = self._dxdydz
        x2, y2, z2 = (x1 + dx, y1 + dy, z1 + dz)

        xs, ys, zs = proj_transform((x1, x2), (y1, y2), (z1, z2), self.axes.M)
        self.set_positions((xs[0], ys[0]), (xs[1], ys[1]))

        return np.min(zs)

def draw_coordinates(data):
    data_ = data[1:,:]
    fig = plt.figure(figsize=(10,10))
    ax = fig.add_subplot(projection='3d')
    if data.shape[1] == 3:
        ax.scatter(data_[:,0], data_[:,1], data_[:,2], c='red', marker='o', s=30)
        ax.scatter(data[0, 0], data[0, 1], data[0, 2], c='black', marker='o', s=30)

        plt.grid(True)
        ax.set_xlabel('x')
        ax.set_ylabel('y')
        ax.set_zlabel('z')
        plt.show()
    elif data.shape[1] == 2:
        ax.scatter(data_[:, 0], data_[:, 1], c='red', marker='o', s=30)
        ax.scatter(data[0, 0], data[0, 1], c='black', marker='o', s=30)

        plt.grid(True)
        ax.set_xlabel('x')
        ax.set_ylabel('y')
        plt.show()


    #plt.savefig('coordinates.png')
    return

def calc_delay(x1, y1, z1, x2, y2, z2):
    return math.sqrt((x1-x2)**2 + (y1-y2)**2 + (z1-z2)**2)

def calc_delay_ndim(a, b):
    s = 0
    for i in range(DIM):
        s += (a[i] - b[i]) ** 2
    return math.sqrt(s)

def gmm_clustering(data, cluster_num, draw):
    data_ = data[1:, :]
    gmm = GaussianMixture(n_components=cluster_num, random_state=2).fit(data_)
    gmm_labels = gmm.predict(data_)
    prob = gmm.predict_proba(data_)

    # gmm_labels = np.array([1, 0, 4, 3, 3, 3, 1, 1, 0, 0, 0, 0, 2, 4, 2, 4, 4, 4])

    if draw:
        fig = plt.figure(figsize=(10, 10))
        ax = fig.add_subplot(projection='3d')

        ax.scatter(data_[:,0], data_[:,1], data_[:,2], c=gmm_labels, cmap='rainbow', s=30)
        ax.scatter(0, 0, 0, c='black', marker='o', s=30)

        ax.set_xlabel('x')
        ax.set_ylabel('y')
        ax.set_zlabel('z')
        plt.suptitle('GMM clustering', fontsize=20)
        plt.show()
        #plt.savefig('gmm.png')

    return gmm_labels, prob

def select_leader(data, idxIndata, cluster_num, gmm_labels):
    data_ = data[1:, :]
    N = len(data)
    cluster_leaders = np.array([[0 for i in range(DIM)]]) #블록 생성 노드
    idxIndata[0].append(0)
    cluster_size = list(range(cluster_num))
    # print(cluster_leaders.shape)

    for k in range(cluster_num):
        k_cluster = data_[gmm_labels == k]
        for i in range(N-1):
            if gmm_labels[i] == k:
                idxIndata[k+1].append(i+1)
        
        # print("In %d-th cluster\n"%(k+1), k_cluster)
        delay = [calc_delay_ndim(data[0], k_cluster[i]) for i in range(len(k_cluster))]
        # cluster_leaders = np.append(cluster_leaders, k_cluster[delay.index(min(delay))], axis=0)
        cluster_leaders = np.vstack((cluster_leaders, k_cluster[delay.index(min(delay))]))
        ### idxIndata[0]에 leader의 idx 하나씩 넣기
        for i in range(N):
            if np.array_equal(data[i], k_cluster[delay.index(min(delay))]):
                idxIndata[0].append(i)
                break
        
        # print("cluster leader: ", k_cluster[delay.index(min(delay))], "delay: ", min(delay))
        cluster_size[k] = len(k_cluster)

    # print("cluster size:\n", cluster_size)

    return idxIndata, cluster_leaders, cluster_size

def create_graph(cluster_size, data, n_data, weighted=False):
    graph = {}

    for i in range(n_data):
        graph[i] = {}
        for j in range(n_data):
            if weighted and i*j == 0:
                graph[i][j] = calc_delay_ndim(data[i], data[j]) / cluster_size[j-1] # weighted by (# of nodes in the cluster)
            elif i != j:
                graph[i][j] = calc_delay_ndim(data[i], data[j])

    return graph

def dijkstra(parent, graph, idxIndata, child, first, k_cluster, cluster_num, MAX_CHILD): 
    '''
    first는 k_cluster의 index
    '''
    S = [0, ]
    n_node = len(graph)
    distance = {node:[float('inf'), first] for node in graph}
    distance[first] = [0, first]
    queue = []

    heapq.heappush(queue, [distance[first][0], first])

    while queue:
        current_distance, current_node = heapq.heappop(queue)

        if distance[current_node][0] < current_distance:
            continue

        for next_node, weight in graph[current_node].items():
            total_distance = current_distance + weight

            if total_distance < distance[next_node][0] and child[idxIndata[cluster_num][current_node]] < MAX_CHILD: # uplink constraint
                # 다음 노드까지 총 거리와 어떤 노드를 통해서 왔는지 입력
                distance[next_node] = [total_distance, current_node]
                heapq.heappush(queue, [total_distance, next_node])
                child[idxIndata[cluster_num][current_node]] = child[idxIndata[cluster_num][current_node]] + 1 # uplink constraint
                S.append(next_node)

    # 마지막 노드부터 첫번째 노드까지 순서대로 출력

    for last in range(0, n_node): # cluster_num >= 1
        delay_sum = 0
        if last != first:
            path = last
            path_output = str(last) + '->'
            a = last
    
            if parent[idxIndata[cluster_num][last]] < 0:
                # if distance[path][1] == 0:
                #     parent[idxIndata[cluster_num][last]] = 0
                if distance[path][0] != float('inf'):
                    parent[idxIndata[cluster_num][last]] = idxIndata[cluster_num][distance[path][1]]

            while distance[path][1] != first:
                path_output += str(distance[path][1]) + '->'
                b = distance[path][1]
                delay_sum = delay_sum + calc_delay_ndim(k_cluster[a], k_cluster[b])
                path = distance[path][1]
                a = b
            path_output += str(first)
            b = first
            delay_sum = delay_sum + calc_delay_ndim(k_cluster[a], k_cluster[b])
            # print(path_output, "delay: ", delay_sum)

    return distance, parent

def find_path_in_cluster(data_, parent, cluster_idx, idxIndata, child, cluster_size, gmm_labels, MAX_CHILD, prev_leaders):
    nodes = data_[gmm_labels == cluster_idx]

    if prev_leaders: # 경로 정보 업데이트
        leader_idx = prev_leaders[cluster_idx] # 이전 리더 노드 사용
    else: # 경로 최초 생성 or 전체 경로 정보 업데이트
        delay = [calc_delay_ndim([0 for j in range(DIM)], nodes[i]) for i in range(len(nodes))]
        leader_idx = delay.index(min(delay))  # cluster Leader 선출 (원점과 가장 가까운 노드)

    cluster_size[cluster_idx] = len(nodes)

    graphIncluster = create_graph(cluster_size, nodes, len(nodes), False)

    distances, parent = dijkstra(parent, graphIncluster, idxIndata, child, leader_idx, nodes, cluster_idx + 1, MAX_CHILD)  # cluster leader로부터 tree 생성

    return parent

def find_path_in_allclusters(data_, parent, idxIndata, child, cluster_num, cluster_size, gmm_labels, MAX_CHILD):
    for k in range(cluster_num):
        find_path_in_cluster(data_, parent, k, idxIndata, child, cluster_size, gmm_labels, MAX_CHILD, prev_leaders)

    return parent

def get_cost(graph, a, b):
    if a < b:
        return graph[a][b]
    else:
        return graph[b][a]

def get_better_node(graph, cluster_size, first, next_node, S):
    new_next_node = next_node
    for k in range(1, len(graph)): # modified
        if k != next_node and (k not in S):
            # if (cluster_size[next_node-1] * graph[first][next_node] + cluster_size[k-1] * (graph[first][next_node] + graph[next_node][k])) > (cluster_size[k-1] * graph[first][k] + cluster_size[next_node-1] * (graph[first][k] + graph[k][next_node])): 
            if (cluster_size[next_node-1] * get_cost(graph, first, next_node) + cluster_size[k-1] * (get_cost(graph, first, next_node) + get_cost(graph, next_node, k)) > cluster_size[k-1] * get_cost(graph, first, k) + cluster_size[next_node-1] * (get_cost(graph, first, k) + get_cost(graph, k, next_node))): 
                new_next_node = k

    return new_next_node

def modified_dijkstra(parent, graph, idxIndata, child, first, k_cluster, cluster_size, cluster_num, MAX_CHILD):
    S = [0, ]
    n_node = len(graph)
    distance = {node:[float('inf'), first] for node in graph}
    distance[first] = [0, first]
    parent[0] = 0
    queue = []

    heapq.heappush(queue, [distance[first][0], first])

    while queue:
        current_distance, current_node = heapq.heappop(queue)

        if distance[current_node][0] < current_distance:
            continue

        for next_node, weight in graph[current_node].items():
            total_distance_temp = current_distance + weight

            if total_distance_temp < distance[next_node][0] and child[idxIndata[cluster_num][current_node]] < MAX_CHILD-min(current_node, 1): # uplink constraint (cluster 내의 노드로 전달할 최소 한 개는 남겨야 함)
                # current -> next_node -> k 일 때와 current -> k -> next_node 일 때 노드의 평균 RTT 최소가 되는지   for all k
                new_next_node = get_better_node(graph, cluster_size, first, next_node, S)
                if new_next_node != next_node: # k가 선택되었을 때, add k to S
                    total_distance = current_distance + get_cost(graph, current_node, new_next_node)
                else:
                    total_distance = total_distance_temp

                # 다음 노드까지 총 거리와 어떤 노드를 통해서 왔는지 입력
                distance[new_next_node] = [total_distance, current_node]
                heapq.heappush(queue, [total_distance, new_next_node])
                child[idxIndata[cluster_num][current_node]] = child[idxIndata[cluster_num][current_node]] + 1 # uplink constraint
                S.append(new_next_node)

                # next_node에 대해서 다시 한번 확인이 필요함
                if new_next_node != next_node:
                    if child[idxIndata[cluster_num][current_node]] < MAX_CHILD-min(current_node, 1):
                        distance[next_node] = [total_distance, current_node]
                        heapq.heappush(queue, [total_distance, next_node])
                        child[idxIndata[cluster_num][current_node]] = child[idxIndata[cluster_num][current_node]] + 1 # uplink constraint
                        S.append(next_node)
                
    # 마지막 노드부터 첫번째 노드까지 순서대로 출력
    for last in range(0, n_node):
        delay_sum = 0
        if last != first:
            path = last
            path_output = str(last) + '->'
            a = last

            if parent[idxIndata[cluster_num][last]] < 0:
                if distance[path][1] == 0 and cluster_num == 0:
                    parent[idxIndata[cluster_num][last]] = 0
                else:
                    parent[idxIndata[cluster_num][last]] = idxIndata[cluster_num][distance[path][1]]
            while distance[path][1] != first:
                path_output += str(distance[path][1]) + '->'
                b = distance[path][1]
                delay_sum = delay_sum + calc_delay_ndim(k_cluster[a], k_cluster[b])
                path = distance[path][1]
                a = b
            path_output += str(first)
            b = first
            delay_sum = delay_sum + delay_sum + calc_delay_ndim(k_cluster[a], k_cluster[b])
            # print(path_output, "delay: ", delay_sum)

    return distance, parent

def calc_delay_idx_ndim(data, a, b):
    s = 0
    for i in range(DIM):
        s += (data[a,i] - data[b,i]) ** 2
    return math.sqrt(s)

def calc_delay_idx(data, a, b):
    return calc_delay(data[a,0], data[a,1], data[a,2], data[b,0], data[b,1], data[b,2])

def calc_e2e_delay(data, parent, to):
    delay_sum = 0
    i = to
    while parent[i] != i:
        delay_sum = delay_sum + calc_delay_idx(data, i, parent[i])
        i = parent[i]
    # print(to, i, delay_sum)
    return delay_sum

def calc_avg_e2e_delay(data, parent):
    N = len(parent)
    total_e2e_delay = 0
    max_e2e_delay = -1
    min_e2e_delay = 10000

    for i in range(N):
        d = calc_e2e_delay(data, parent, i)
        if d > max_e2e_delay:
            max_e2e_delay = d
        if d < min_e2e_delay:
            min_e2e_delay = d
        total_e2e_delay = total_e2e_delay + d

    avg_e2e_delay = total_e2e_delay / N

    return avg_e2e_delay, max_e2e_delay, min_e2e_delay

def _arrow3D(ax, x, y, z, dx, dy, dz, *args, **kwargs):
    '''Add an 3d arrow to an `Axes3D` instance.'''

    arrow = Arrow3D(x, y, z, dx, dy, dz, *args, **kwargs)
    ax.add_artist(arrow)


setattr(Axes3D, 'arrow3D', _arrow3D)

def draw_nodes(data, gmm_labels, fig):
    data_ = data[1:, :]
    ax = fig.add_subplot(projection='3d')
    if data.shape[1] == 3:
        ax.scatter(data_[:, 0], data_[:, 1], data_[:, 2], c=gmm_labels, cmap='rainbow', s=30)
        ax.scatter(data[0, 0], data[0, 1], data[0, 2], c='black', marker='o', s=30)

        plt.grid(True)
        ax.set_xlabel('x')
        ax.set_ylabel('y')
        ax.set_zlabel('z')
        plt.show()
    elif data.shape[1] == 2:
        ax.scatter(data_[:, 0], data_[:, 1], c=gmm_labels, cmap='rainbow', s=30)
        ax.scatter(data[0, 0], data[0, 1], c='black', marker='o', s=30)

        plt.grid(True)
        ax.set_xlabel('x')
        ax.set_ylabel('y')
        plt.show()

    return ax

def draw_tree_path(data, parent, ax):
    for i in range(1, len(data)):
        # parnet[i] -> i 화살표 그리기
        if parent[i] >= 0:
            p = parent[i]
            ax.arrow3D(data[p,0], data[p,1], data[p,2], data[i,0]-data[p,0], data[i,1]-data[p,1], data[i,2]-data[p,2], mutation_scale=20, arrowstyle="->")
    return ax

def draw_mesh_path(data, mesh_conn, ax):
    for i in range(0, len(data)):
        if mesh_conn[i] > 0: # Mesh
            p = mesh_conn[i]
            # ax.arrow3D(data[p,0], data[p,1], data[p,2], data[i,0] - data[p,0], data[i,1]-data[p,1], data[i,2]-data[p,2], mutation_scale=20, arrowstyle="->", linestyle='dashed', ec='blue')
            ax.plot([data[p,0], data[i,0]], [data[p,1], data[i,1]], [data[p,2], data[i,2]], color='b')
    return ax

def draw_omtree(data, parent, gmm_labels):
    if DIM <= 3:
        fig = plt.figure(figsize=(10,10))
        ax = draw_nodes(data, gmm_labels, fig)
        ax = draw_tree_path(data, parent, ax)

        ax.set_xlabel('x')
        ax.set_ylabel('y')
        ax.set_zlabel('z')
        plt.show()
    #plt.savefig('omtree.png')

    return

def find_nearestIncluster(data, idxIndata, src, Incluster_idx):
    min = 9999
    dst_min = 0
    for i in range(len(idxIndata[Incluster_idx+1])):
        dst = idxIndata[Incluster_idx+1][i]
        delay = calc_delay_idx(data, src, dst)
        if delay < min:
            min = delay
            dst_min = dst
    return dst_min

def check_mesh_max(mesh_num, MAX_M):
    max_cnt = 0
    for i in range(len(mesh_num)):
        if mesh_num[i] == MAX_M:
            max_cnt = max_cnt + 1
    if max_cnt == len(mesh_num): # all cluster already has MAX mesh
        return False
    else: # Not yet
        return True

def check_mesh_min(mesh_num, MIN_M):
    min_cnt = 0
    for i in range(len(mesh_num)):
        if mesh_num[i] >= MIN_M:
            min_cnt = min_cnt + 1
    if min_cnt == len(mesh_num): # all cluster has more mesh than MIN
        return True
    else: # Not yet
        return False

# 두 클러스터간 mesh 경로는 최대 1개 조건 확인
def check_redundant_mesh(mesh_dst_cluster, cluster_idx1, cluster_idx2):
    if cluster_idx2 in mesh_dst_cluster[cluster_idx1] or cluster_idx1 in mesh_dst_cluster[cluster_idx2]:
        return False
    else:
        return True

def is_connected(parent, a, b):
    if parent[a] == b or parent[b] == a:
        return True
    else:
        return False

def init():
    parser = argparse.ArgumentParser(description='Construct OM Tree and Mesh based on 3D coordinates.')
    parser.add_argument('--childnum', type=int, required=True, help='Maximum # of child nodes')
    parser.add_argument('--minMesh', type=int, required=True, help='Minimum # of mesh nodes')
    parser.add_argument('--maxMesh', type=int, required=True, help='Maximum # of mesh nodes')
    parser.add_argument('--port', type=int, required=True, help='Port number')

    args = parser.parse_args()     

    # 소켓 객체를 생성합니다. 
    serverSocket = socket(AF_INET, SOCK_STREAM)

    # 소켓 주소 정보 할당 
    # HOST = "141.223.65.115"
    # HOST = "172.19.0.2"

    if LOCAL:
        HOST = "127.0.0.1" # 좌표계 container IP
    else:
        HOST = "172.17.0.4" # "172.19.0.99"
        # HOST = "127.0.0.1"
    PORT = args.port
    ADDR = (HOST, PORT)

    serverSocket.setsockopt(SOL_SOCKET, SO_REUSEADDR, 1)

    print(ADDR)
    serverSocket.bind(ADDR)
    print('bind')

    serverSocket.listen(100)
    print('listen')

    print('waiting for connection')
    if LOCAL:
        clientSocket = None
    else:
        clientSocket, addr_info = serverSocket.accept()
    print('accept')
    print('--client information--')
    clientSocket.setblocking(True)
    print(clientSocket)

    return args, serverSocket, clientSocket

def parse_data(read_buf, data):
    '''
    read_buf:
    Num_node (4bytes) Dim (4bytes) Opt (4bytes)
    Node ID (4bytes), 3D coordinates (x_1,x_2,.., x_n) (dim x 4 bytes)
    Node ID (4bytes), 3D coordinates (x_1,x_2,.., x_n) (dim x 4 bytes)
    …
    Node ID (4bytes), 3D coordinates (x_1,x_2,.., x_n) (dim x 4 bytes)
    '''

    print("parse start!")
    NODE_NUM_SIZE = 4
    DIM_SIZE = 4
    OPT_SIZE = 4

    node_num = struct.unpack_from('>I', read_buf, 0)[0]
    dim = struct.unpack_from('>I', read_buf, NODE_NUM_SIZE)[0]
    TUPLE_SIZE = 4 + dim * 4
    opt = struct.unpack_from('>I', read_buf, NODE_NUM_SIZE+DIM_SIZE)[0]
    print("node_num: %d" % node_num, "dim: %d" % dim)

    update_cluster_idx = None
    if opt: # 이전에 받은 노드들의 좌표가 존재해야 함.
        assert data
    else:
        data = np.zeros(shape=(node_num,dim), dtype=float)

    for i in range(node_num):
        recv_buf = struct.unpack_from('>I%df'%dim, read_buf, NODE_NUM_SIZE + DIM_SIZE + OPT_SIZE + TUPLE_SIZE * i)
        # TODO: append가 아니라 data[id]에 삽입
        idx = recv_buf[0]
        for d in range(dim):
            data[idx, d] = recv_buf[d+1]
        # data[recv_buf[0], 0] = recv_buf[1] # x value
        # data[recv_buf[0], 1] = recv_buf[2] # y value
        # data[recv_buf[0], 2] = recv_buf[3] # z value
        if not(i) and opt:
            update_cluster_idx = prev_path_info.gmm_labels[recv_buf[0]]

    # print("id:", id)
    # print(x, y, z)

    assert np.all(data[0,:] == 0)
    # assert data[0,0] == data[0,1] == data[0,2] == 0

    return data, opt, update_cluster_idx

def read_coordinates(clientSocket, update):
    if LOCAL:
        ####################################
        # 19 nodes
        # x = [0,	7.95,	2.53475,	3.975,	7.04E-05,	-0.0501444,	0.342326,	7.94999,	7.94996,	5.41525,	2.93019,	3.975,	3.90283,	1.85944,	5.05944,	2.02531,	4.71352,	3.975,	3.46163]
        # y = [0,	0,	10.6526,	11.2391,	-1.07482,	-0.513752,	-0.607127,	0.549165,	0.495312,	11.042,	10.3044,	11.3847,	10.8527,	10.9794,	11.6712,	11.0599,	12.3635,	12.625,	12.4754]
        # z = [0,	0,	0,	9.82772,	1.69625,	-0.190934,	0.555883,	-1.42679,	-1.36519,	-0.44532,	-0.0371116,	-0.25471,	0.518292,	8.87917,	9.69374,	10.1241,	8.25824,	7.49207,	7.33622]

        if DIM == 3:
        # 60 nodes (3D)
            x = [0,15.05,2.69111,-3.33389,-1.23206,-2.20232,-1.97666,1.02E-05,15.05,3.68779,2.69112,-1.31221,-2.39884,-0.820931,1.47E-05,15.05,3.68779,2.69112,1.60266,-2.30888,-1.88987,6.06E-06,15.5012,3.74826,2.69111,-1.08339,-2.21877,-1.97666,1.61E-05,15.05,3.68779,2.69111,-0.82882,-2.20232,-1.97666,0.000115648,15.05,3.68779,2.69111,1.59186,-2.30888,1.71811,0.00029045,15.05,3.68779,2.69111,-1.31221,-2.29227,-1.87342,1.02E-05,15.3021,5.02334,2.63438,-1.2956,-2.30888,-1.96004,-0.302979,15.05,3.68779,2.69111]
            y = [0,0,11.7457,18.6594,17.4215,18.9067,18.3484,1.08E-05,1.04E-06,3.62491,11.7456,17.3372,19.0161,19.6034,1.55E-05,2.42E-06,3.62491,11.7456,20.2035,19.1108,18.6109,6.48E-06,0.481842,3.68853,11.7456,18.9534,17.5582,18.5196,1.70E-05,1.47E-06,3.62491,11.7456,17.8458,19.0864,18.3484,0.000121791,5.19E-07,3.62491,11.7456,19.6927,19.1108,22.4072,0.000305713,2.08E-06,3.57777,11.7456,17.3372,18.902,18.4959,1.08E-05,0.192549,5.03019,11.7586,17.2309,18.7506,18.4048,0.0694319,4.15E-06,3.62491,11.7456]
            z = [0,0,0,17.9422,15.0836,15.9751,18.5178,-7.31E-07,0.176677,0.694507,0.112039,15.0892,15.7494,18.3103,-1.04E-06,1.13844,0.694507,0.112046,15.0139,15.2326,18.3337,0.0726652,1.10429,-0.157427,0.112038,14.4753,17.4501,18.3398,-1.14E-06,1.13844,0.624629,0.112059,15.0555,15.7883,18.5178,-8.08E-06,1.13845,0.624629,0.112062,15.6151,15.7431,18.0811,0.0726454,0.264805,0.673658,-0.0562156,15.0892,14.0601,18.3836,-7.31E-07,1.02287,0.601485,0.0880462,15.1357,16.1176,18.3896,-0.347322,1.13844,0.343729,0.112051]

        elif DIM == 5:
        # 60 nodes (5D)
            x1 = [0,	45.527668,	46.079891,	12.226507,	28.734617,	40.707386,	13.170201,	43.107944,	44.722931,	14.370635,	31.172707,	37.95052,	13.647175,	43.049049,	43.737904,	14.878614,	29.435461,	39.130207,	13.861241,	43.200066,	44.368507,	14.234071,	29.146582,	37.835724,	12.778604,	43.021694,	43.976559,	13.828501,	29.707954,	39.239532,	12.890491,	42.848171,	44.905098,	13.221446,	29.6047,	38.054413,	13.676408,	42.667305,	45.067581,	13.983413,	29.763929,	38.413162,	14.170085,	42.68351,	44.698273,	14.188374,	29.707079,	38.835384,	14.116993,	42.996536,	44.503323,	14.233004,	29.68932,	38.706024,	14.070532,	42.984287,	44.462257,	14.010606,	30.174717,	38.910236]
            x2 = [0,	0,	37.481266,	6.716929,	3.242182,	7.879697,	1.026533,	0.981212,	36.159008,	1.610059,	1.701039,	6.574775,	1.560802,	0.944713,	36.396454,	2.315508,	2.699126,	6.639625,	1.216027,	1.068424,	36.389435,	1.409936,	3.32699,	6.495114,	1.437713,	0.967208,	36.224022,	1.927115,	2.935386,	7.523325,	2.424578,	0.936283,	35.634483,	2.589001,	2.782928,	7.736256,	0.02805,	1.289494,	35.64537,	2.777819,	3.240653,	7.636641,	0.927698,	0.924712,	36.495045,	1.31961,	2.682662,	7.010975,	0.717509,	1.450143,	36.385571,	1.299781,	2.99315,	7.19724,	0.909328,	0.925819,	36.412052,	1.863287,	2.722056,	6.57354]
            x3 = [0,	0,	0,	19.561699,	10.976339,	6.240694,	13.770163,	2.307942,	-7.1231,	18.470419,	12.180475,	-0.125987,	13.273771,	0.625833,	-6.604564,	17.909517,	11.024878,	1.275838,	13.229711,	2.190034,	-7.181095,	17.880192,	10.057796,	0.810788,	13.105909,	2.076515,	-8.251102,	18.075171,	10.188979,	2.357322,	13.507622,	1.950398,	-7.248135,	18.014233,	10.151733,	0.705774,	12.78536,	0.690875,	-5.325271,	17.298059,	11.295672,	1.509234,	13.129973,	1.588995,	-5.8485,	17.882593,	10.192395,	1.815566,	12.739432,	0.42386,	-11.472737,	17.778143,	10.266108,	2.02517,	12.97311,	2.035579,	-7.083712,	17.518402,	9.346027,	2.403045]
            x4 = [0,	0,	0,	0,	30.920076,	2.755022,	1.376251,	0.716672,	2.979493,	-1.324352,	27.837852,	4.889807,	1.152199,	1.202019,	2.906438,	-1.677401,	28.168564,	4.735182,	0.616337,	-0.104411,	3.86286,	-1.492011,	28.588598,	4.345372,	0.041063,	0.618046,	3.015308,	-1.116331,	28.48419,	4.547226,	0.244235,	0.335601,	2.415625,	-0.695941,	28.280632,	4.772834,	0.65148,	1.284384,	2.308811,	-0.978102,	28.559446,	4.855224,	0.11029,	0.649143,	2.896338,	-1.120748,	28.235701,	4.128581,	0.055103,	1.258754,	5.482625,	-1.093085,	28.480413,	4.175609,	-0.692904,	0.752755,	3.088933,	-1.031544,	28.861004,	4.531252]
            x5 = [0,	0,	0,	0,	0,	21.887264,	-0.295931,	-0.260988,	-0.092574,	-3.549176,	-1.478687,	21.869926,	-1.469162,	-0.311941,	-0.113422,	-4.179876,	-0.122527,	21.363947,	-1.897889,	-0.979863,	-0.147198,	-3.315423,	0.153042,	21.544394,	-0.406654,	-0.257023,	0.124091,	-3.224701,	-0.126009,	21.263407,	0.224571,	-1.107826,	-0.792276,	-2.496153,	-0.341824,	21.623196,	-1.526802,	-0.18759,	-1.050709,	-3.971289,	-0.088503,	21.146057,	-2.242685,	-0.113127,	0.084153,	-3.73425,	0.106031,	20.802618,	-2.284657,	-0.016469,	1.542191,	-2.141333,	-0.267417,	20.95281,	-3.173578,	-0.282875,	0.201283,	-3.210896,	-0.214559,	20.929029]

        elif DIM == 6:
            x1 = [0,	64.378334,	68.454933,	8.033205,	37.988029,	60.203976,	10.954916,	11.431518,	62.027832,	65.805298,	8.307685,	38.693001,	58.420944,	0.853,	11.362052,	61.202713,	65.682938,	7.509203,	38.620243,	59.141094,	0.888559,	11.662049,	60.960793,	65.857285,	7.790588,	38.796463,	58.619099,	0.900722,	11.787723,	1.268951,	66.138351,	60.377048,	38.843327,	58.793209,	0.875718,	11.727611,	60.873589,	66.729675,	7.644835,	38.79668,	58.023582,	1.205032,	11.690618,	61.011574,	66.175888,	7.834677,	38.358681,	59.037296,	0.90395,	11.803821,	61.484077,	65.36657,	7.856133,	38.152199,	58.892082,	0.884902,	11.499439,	60.996136,	65.901543,	7.659936,	38.43903,	58.215244,	1.018248,	11.790752,	61.344036,	66.079239,	7.228726,	38.823906,	58.813835,	0.922353,	11.692937,	61.67234,	65.405998,	7.720809,	38.437019,	58.103035,	1.268613,	7.761049,	60.91481,	66.86261]
            x2 = [0,	0,	36.37326,	-5.581699,	3.628381,	6.646132,	-2.634003,	1.447216,	0.460887,	35.984203,	-3.1139,	2.212536,	8.052327,	-0.48805,	3.514932,	0.86464,	36.874847,	-1.394759,	2.328019,	7.704707,	-0.47447,	2.157357,	0.913603,	36.685486,	-1.843975,	2.534374,	8.163228,	0.096065,	3.216245,	0.396921,	36.763786,	1.407363,	2.427583,	7.590989,	-0.319924,	1.995296,	0.874707,	35.922256,	-0.815304,	2.636502,	7.715539,	0.371572,	3.262242,	0.882743,	36.677303,	-1.121835,	2.232098,	7.096862,	-0.620645,	3.383415,	0.820899,	36.885681,	-3.289745,	3.776206,	6.168512,	-0.483881,	1.760622,	0.899328,	36.643543,	-0.530215,	3.063305,	8.659066,	-0.404342,	2.101095,	0.956446,	36.882736,	-1.664346,	2.76386,	7.504861,	0.234335,	3.386081,	0.768636,	36.899231,	-3.054997,	1.950388,	6.911117,	0.401028,	-1.218553,	-0.171012,	36.457142]
            x3 = [0,	0,	0,	31.985289,	2.813059,	-3.803722,	13.165397,	13.207709,	-1.922535,	1.788258,	30.637884,	3.647057,	-0.093639,	1.039094,	13.487372,	-1.876323,	2.775208,	30.861465,	3.695139,	0.011073,	1.174681,	13.252962,	-2.929651,	2.668509,	31.317858,	4.51095,	-0.766508,	1.240832,	13.548824,	1.248746,	3.482655,	-3.869938,	4.393257,	-0.234313,	1.020553,	13.553203,	-2.882907,	3.186435,	31.177353,	3.467948,	-2.010716,	1.294384,	13.49352,	-2.620567,	4.004139,	31.531612,	4.036168,	-1.864035,	0.982706,	13.504616,	-1.843267,	1.781712,	29.884184,	4.390468,	-1.115126,	1.219863,	13.221992,	-2.720976,	2.7278,	31.215906,	4.238014,	-0.497343,	1.098913,	13.221882,	-2.186827,	3.636356,	30.382967,	4.632796,	0.23751,	1.063544,	13.602023,	-1.706174,	2.454059,	30.945263,	4.343756,	-3.353815,	1.087796,	31.294924,	-3.122456,	4.602195]
            x4 = [0,	0,	0,	0,	37.440033,	3.204653,	3.240322,	4.754912,	2.991419,	0.390154,	1.651919,	34.775826,	5.475034,	-0.134057,	3.622543,	2.129711,	1.298246,	1.395578,	35.129925,	5.866664,	-0.578959,	4.103427,	1.19624,	1.247427,	2.148527,	34.74131,	4.622186,	-0.281998,	4.163957,	-0.278761,	1.294052,	0.782895,	34.733524,	4.639643,	-0.161578,	4.501274,	1.223203,	1.355011,	1.697225,	34.43774,	4.256635,	-0.492328,	3.367127,	1.627679,	1.441244,	1.809517,	34.77216,	5.127927,	-0.179059,	3.635153,	1.857711,	1.333423,	1.418375,	35.339108,	4.744378,	-0.10871,	3.730767,	1.563289,	1.151192,	1.702159,	35.158463,	5.716263,	-1.3031,	3.529688,	2.273921,	1.203749,	1.249814,	34.771053,	3.950416,	-0.287601,	3.63698,	2.405894,	1.058994,	2.230395,	35.319504,	4.721007,	-1.064851,	1.71652,	2.051298,	1.763135]
            x5 = [0,	0,	0,	0,	0,	22.561596,	2.031113,	3.833928,	0.254515,	1.439127,	6.219229,	0.470139,	22.409985,	2.470786,	0.863766,	0.325958,	2.504304,	7.709635,	1.23347,	22.403622,	-0.682176,	3.346816,	-0.076754,	2.889353,	4.117635,	0.458838,	21.782444,	0.966639,	3.251861,	0.196167,	2.56843,	-0.085411,	0.671145,	21.99095,	1.553856,	5.727256,	0.083955,	2.248819,	6.251828,	0.769841,	21.67626,	1.952111,	2.816581,	0.290329,	2.464945,	7.797046,	2.36547,	21.559429,	2.353927,	4.129903,	0.411015,	2.296538,	5.98015,	2.823538,	21.676533,	3.171574,	4.429909,	0.237621,	2.31812,	7.413223,	1.294798,	22.205494,	2.360809,	3.809725,	0.114017,	3.406904,	5.807486,	0.646106,	22.1901,	2.203514,	3.884716,	0.415809,	1.724259,	7.440004,	2.335759,	21.582531,	2.184639,	8.242501,	0.275697,	3.134234]
            x6 = [0,	0,	0,	0,	0,	0,	18.386269,	16.355457,	4.988603,	4.475208,	-0.960285,	1.737557,	1.001299,	0.964258,	16.196678,	3.450773,	4.78031,	-0.238519,	2.562647,	5.277574,	1.358339,	16.160376,	1.378666,	5.757206,	0.697202,	1.812333,	0.529077,	1.055454,	15.91291,	1.052043,	4.962954,	2.629406,	1.426902,	1.893064,	1.608046,	16.034845,	2.973518,	6.826485,	0.177086,	1.653751,	1.600608,	0.381841,	16.017197,	1.675344,	4.57926,	0.3306,	1.088727,	5.079086,	1.491503,	16.562876,	3.185422,	4.119408,	-1.090032,	1.993738,	3.756689,	0.312622,	15.696047,	1.92824,	4.969508,	-0.695239,	2.347499,	3.810007,	1.165857,	16.117865,	3.901428,	6.25642,	-1.03592,	1.874854,	0.373797,	2.33366,	16.194918,	2.822038,	5.657447,	0.122837,	2.135757,	2.092109,	1.60049,	-0.510496,	1.613373,	5.762681]
        if DIM == 2:
            data = np.vstack((x, y)).T
        elif DIM == 3:
            data = np.vstack((x,y,z)).T
        elif DIM == 4:
            data = np.vstack((x1, x2, x3, x4)).T
        elif DIM == 5:
            data = np.vstack((x1, x2, x3, x4, x5)).T
        elif DIM == 6:
            data = np.vstack((x1, x2, x3, x4, x5, x6)).T
        # data_ = data[1:,:] # data_: except the origin
        ####################################
        opt = update
        if opt:
            update_cluster_idx = 2
        else:
            update_cluster_idx = None

    else:
        read_buf = clientSocket.recv(65535)
        n = len(read_buf)
        # print('received data: ', read_buf, n) # TODO: blocking mode
        # TODO: 검증 노드로부터 받은 메시지 (좌표값)에 따라
        # 1. 특정 클러스터의 노드만: 클러스터링 x, tree & mesh 만 수행
        # 2. 전체 노드: 클러스터링 o, tree & mesh 수행

        if n > 0:
            data, opt, update_cluster_idx = parse_data(read_buf, None)
        else:
            # print("Receive no coord")
            return n, None, None, None
        # data_ = data[1:,:] # data_: except the origin

    print("# of nodes:%d"%len(data))
    print(data)
    if update_cluster_idx:
        print('Update cluster #%d' % (update_cluster_idx))

    return n, data, opt, update_cluster_idx

def build_omtree(data, opt, prev_leaders, update_cluster_idx):
    data_ = data[1:,:]
    start = time.time() # Measure execution time of GMM iteration

    N = len(data)
    best_avg_e2e_delay = 99999999 
    best_n_clusters = -1

    if opt: # TODO: 클러스터링 수행 x, Dijkstra로 Tree만 만듦
        # best_n_clusters, best_gmm_labels, best_prob, best_parent, best_idxIndata 값 설정 필요
        # TODO: cluster leader 그대로
        # prev_path_info
        parent = prev_path_info.parent
        cluster_idx = update_cluster_idx# TODO: 문제가 있는 클러스터 찾아야 함
        idxIndata = prev_path_info.idxIndata
        child = prev_path_info.child
        cluster_size = prev_path_info.cluster_size
        gmm_labels = prev_path_info.gmm_labels

        find_path_in_cluster(data_, parent, cluster_idx, idxIndata, child, cluster_size, gmm_labels, MAX_CHILD, prev_leaders)

        return None, gmm_labels, None, parent, idxIndata
        raise NotImplementedError()
    else:
        for n_clusters in range(2, 11):
            child = [0 for i in range(N)]
            parent = [-1 for i in range(N)]

            idxIndata = [[] for i in range(n_clusters + 1)] # 0 : idx of cluster leaders in data, 1 ~ : idx of cluster members in data

            gmm_labels, prob = gmm_clustering(data, n_clusters, False)

            idxIndata, cluster_leaders, cluster_size = select_leader(data, idxIndata, n_clusters, gmm_labels)
            # print('cluster leaders:', cluster_leaders)

            graphLeaders = create_graph(cluster_size, cluster_leaders, len(cluster_leaders), True)

            distances, parent = modified_dijkstra(parent, graphLeaders, idxIndata, child, 0, cluster_leaders, cluster_size, 0, MAX_CHILD) # 0: 블록 생성 노드

            # draw_omtree(data, parent, gmm_labels)

            parent = find_path_in_allclusters(data_, parent, idxIndata, child, n_clusters, cluster_size, gmm_labels, MAX_CHILD)

            avg_e2e_delay, max_e2e_delay, min_e2e_delay = calc_avg_e2e_delay(data, parent)
            print("# of clusters: ", n_clusters, ", avg. e2e delay: ", avg_e2e_delay, "max e2e delay: ", max_e2e_delay, "min e2e delay: ", min_e2e_delay);
            if avg_e2e_delay < best_avg_e2e_delay:
                best_avg_e2e_delay = avg_e2e_delay
                best_max_e2e_delay = max_e2e_delay
                best_min_e2e_delay = min_e2e_delay
                best_n_clusters = n_clusters
                best_gmm_labels = gmm_labels
                best_prob = prob
                best_parent = parent
                best_cluster_size = cluster_size
                best_child = child
                best_idxIndata = idxIndata

            # draw_omtree(data, parent, gmm_labels)

        print("The best # of clusters: ", best_n_clusters, ", avg. e2e delay: ", best_avg_e2e_delay, "max. e2e delay: ", best_max_e2e_delay, "min. e2e delay: ", best_min_e2e_delay)
        print("time :", time.time() - start, "sec")

    # draw_omtree(data, best_parent, best_gmm_labels)

    prev_path_info.child = best_child
    prev_path_info.cluster_size = best_cluster_size

    return best_n_clusters, best_gmm_labels, best_prob, best_parent, best_idxIndata

def build_mesh(data, best_prob, n_clusters, parent):
    N = len(best_prob) + 1
    second_largest_prob = list(range(N))
    second_largest_prob[0] = -1
    for i in range(1,N):
        second_largest_prob[i] = best_prob[i-1,best_prob.argsort()[i-1,n_clusters-2]]
    print(second_largest_prob)

    mesh_num = list(0 for i in range(n_clusters))
    mesh_dst_cluster = [[-1 for col in range(MAX_M)] for row in range(n_clusters)] # mesh_dst_cluster[cluster_idx][mesh_num[cluster_idx]]
    mesh_conn = [-1 for i in range(N)] # 양방향

    j = N-1
    while j in range(1, N) and check_mesh_max(mesh_num, MAX_M):
        mesh_candi_idx = second_largest_prob.index(max(second_largest_prob)) # second largest probability가 가장 큰 노드의 인덱스
        assert mesh_candi_idx > 0
        second_cluster_idx = best_prob.argsort()[mesh_candi_idx-1,n_clusters-2]
        cluster_idx = best_prob.argsort()[mesh_candi_idx-1,n_clusters-1]
        dst = find_nearestIncluster(data, idxIndata, mesh_candi_idx, second_cluster_idx) # find the nearest node (dst) from mesh_candi_idx
        # print("max of the second prob.: ", max(second_largest_prob), "cluster idx: ", cluster_idx, "second_cluster_idx: ", second_cluster_idx)

        if mesh_num[cluster_idx] < MAX_M and mesh_num[second_cluster_idx] < MAX_M and (not is_connected(parent, mesh_candi_idx, dst)) and check_redundant_mesh(mesh_dst_cluster, cluster_idx, second_cluster_idx): 
            print("node: ", mesh_candi_idx, "in the ", cluster_idx, "-th cluster is connected with the ", second_cluster_idx, "-th cluster (prob.:", best_prob[mesh_candi_idx-1, second_cluster_idx], ")") # mesh_candi_idx --> second_nearest_cluster_idx와 연결

            mesh_dst_cluster[cluster_idx][mesh_num[cluster_idx]] = second_cluster_idx
            mesh_dst_cluster[second_cluster_idx][mesh_num[second_cluster_idx]] = cluster_idx

            mesh_num[cluster_idx] = mesh_num[cluster_idx] + 1
            mesh_num[second_cluster_idx] = mesh_num[second_cluster_idx] + 1

            mesh_conn[mesh_candi_idx] = dst
            mesh_conn[dst] = mesh_candi_idx

        second_largest_prob[mesh_candi_idx] = -1
        j = j - 1

    if not check_mesh_min(mesh_num, MIN_M):
        print("MIN Mesh Not Satisfied!!", mesh_num)
    else:
        print("MIN Mesh Satisfied.", mesh_num)
    print("Mesh fin!")

    return mesh_conn

def draw_path(data, gmm_labels, parent, mesh_conn):
    if DIM <= 3:
        fig = plt.figure(figsize=(10,10))
        ax = draw_nodes(data, gmm_labels, fig)
        ax = draw_tree_path(data, parent, ax)
        ax = draw_mesh_path(data, mesh_conn, ax)

        ax.set_xlabel('x')
        ax.set_ylabel('y')
        ax.set_zlabel('z')
        plt.show()
    #plt.savefig('path.png')
    return

def get_child_num(parent, nodeid):
    childids = [i for i in range(len(parent)) if parent[i] == nodeid and i != nodeid]
    return len(childids), childids

def send_pathInfo(clientSocket, node_num, gmm_labels, leaders, parent, mesh_conn):
    print('Create msg of path information')
    # gmm_labels.astype(np.int32)    
    write_buf = struct.pack('!I', node_num) # byte order: Network

    # (node ID, Mesh ID, cluster_idx, cluster_leader ID, child_num, child ID, child ID, ..., child ID)
    # (I, i, I, I, I, ..., I)
    
    for i in range(node_num):
        nodeid = i
        child_num, childids = get_child_num(parent, nodeid)
        meshid = mesh_conn[nodeid]
        if(i == 0):
            cluster_idx = -1
            leaderid = -1
        else:
            cluster_idx = gmm_labels[i-1]
            leaderid = leaders[cluster_idx]
        print("nodeID: %d, meshID: %d, cluster_idx: %d, cluster_leader ID: %d, child_num: %d" % (nodeid, meshid, cluster_idx, leaderid, child_num), childids)
        write_buf += struct.pack('!IiiiI', nodeid, meshid, cluster_idx, leaderid, child_num)
        for j in range(child_num):
            write_buf += struct.pack('!I', childids[j]) # byte order: Network

    clientSocket.sendall(write_buf)
    print('Sending path information to the valClient', clientSocket)

    return 

if __name__ == "__main__":
    print('start!')

    setattr(Axes3D, 'arrow3D', _arrow3D)

    args, serverSocket, clientSocket = init()
    
    MAX_CHILD = args.childnum
    MIN_M = args.minMesh
    MAX_M = args.maxMesh
    print("Maximum child num: %d\nMinimum mesh num per cluster: %d\nMaximum mesh num per cluster: %d"%(MAX_CHILD, MIN_M, MAX_M))

    prev_leaders = prev_mesh_conn = None

    update = 0
    verSocket = None
    while update == 0:
        # Receive message from blk_generator
        n, data, opt, update_cluster_idx = read_coordinates(clientSocket, update)

        # draw_coordinates(data)
        if n > 0:
            # Tree
            n_clusters, gmm_labels, gmm_prob, parent, idxIndata = build_omtree(data, opt, prev_leaders, update_cluster_idx)
            print("clustering result:", gmm_labels) # cluster_idx: gmm_labels
            print(idxIndata)
            # print('leaders:', idxIndata[0][1:]) # Leaders: idxIndata[0][1:]
            print("OM Tree fin!")

            # Mesh
            # TODO: opt == 1 이면 mesh_conn 그대로
            if not(opt):
                mesh_conn = build_mesh(data, gmm_prob, n_clusters, parent)
            else:
                mesh_conn = prev_mesh_conn
            print("mesh_path:", mesh_conn)

            # draw_path(data, gmm_labels, parent, mesh_conn)

            print("Path construction fin!")

            # parent ==> num_child + child ID list
            # mesh ==> mesh ID
            print("parent:", parent, "\nmesh:", mesh_conn)

            if verSocket == None:
                verSocket = socket(AF_INET, SOCK_STREAM)
                ADDR = ("172.17.0.2", 1622) # TODO: "172.19.0.97"

                verSocket.setsockopt(SOL_SOCKET, SO_REUSEADDR, 1)

                verSocket.connect(ADDR)
                print('Connected to ver_node for tree/mesh info')

            send_pathInfo(verSocket, len(data), gmm_labels, idxIndata[0][1:], parent, mesh_conn) # clientSocket ==> new ver_node

            print('---------------------------------------------Fin---------------------------------------------------')
            # TODO: 한 클러스터의 경로 정보 업데이트 시 리더와 mesh는 바뀌지 않음
            prev_leaders = idxIndata[0][1:]
            prev_mesh_conn = mesh_conn
            prev_path_info.parent = parent
            prev_path_info.idxIndata = idxIndata
            prev_path_info.gmm_labels = gmm_labels
            update = 0
            print('prev_leaders:', prev_leaders)

    if verSocket != None:
        verSocket.close()
    serverSocket.close()
    if clientSocket != None:
        clientSocket.close()
    print('close')
