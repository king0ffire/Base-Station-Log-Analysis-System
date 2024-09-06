import csv

from scapy.all import *
import gzip
import glob

log_name="Log_20240618_092153"
base_path=r"E:/"+log_name+"/logs"
file_list=glob.glob(base_path+r"\sctp*")

csvfile=open(os.path.join(base_path,log_name)+".csv","w",newline='')
csvwriter=csv.writer(csvfile)
for filename in file_list:
    if(filename[-3:]=='.gz'):
        with gzip.open(filename) as f:
            pkts=rdpcap(f,12)
            pkt=pkts[10]
            print(pkt)

