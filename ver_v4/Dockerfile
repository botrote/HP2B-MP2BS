FROM python
WORKDIR /usr/src/app
COPY . .
RUN apt-get update
RUN apt-get install -y iproute2 net-tools iputils-ping
RUN python3 -m pip install scikit-learn
RUN python3 -m pip install matplotlib
#RUN apt-get install -y python-tkinter
CMD ["omtree_mesh_ndim_v2.py"]
ENTRYPOINT ["python", "omtree_mesh_ndim_v2.py"]
