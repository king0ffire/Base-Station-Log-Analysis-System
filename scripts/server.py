from ast import dump
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
import csv
from email import message
from functools import cache
import glob
import gzip
import os
import queue
import logging.handlers
import socket
import sys
import tarfile
import json
import time
from venv import logger
from line_profiler import profile


from aiosql import Mypool, createmysiambyconn
from util import mapget,DBWriter
from dbcount import constructdbgcsv,Parsefilelist_4

@profile
def run(filemeta, dbconn, cachelocation,mode=0):
    fileuid=filemeta["fileuid"]
    filelocation=os.path.join(cachelocation,fileuid+".tar.gz")
    logger = logging.getLogger(__name__)

    
    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    fileuid=os.path.basename(extracteddir)
    createmysiambyconn(dbconn,fileuid)
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation, "r:gz") as tar:
        tar.extractall(path=extracteddir,filter='fully_trusted')
    
    tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
    traceextraceteddir = os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
    with tarfile.open(tracelocation, "r:gz") as tar:
        tar.extractall(path=traceextraceteddir,filter='fully_trusted')
    dbglogsdir = os.path.join(traceextraceteddir, "trace") 
    csvfile_dbg = open(os.path.join(extracteddir, "dbg.csv"), "w", newline="",encoding="utf-8")
    csvwriter_dbg = csv.writer(csvfile_dbg)
    dbg_gz_list =glob.glob(dbglogsdir + "/dbglog*.gz")
    for gzname in dbg_gz_list:
        with gzip.open(gzname, "rb") as f_in:
            with open(os.path.splitext(gzname)[0], "wb") as f_out:
                f_out.write(f_in.read())

    dbg_file_list = [item for item in glob.glob(dbglogsdir + "/dbglog*") if not item.endswith(".gz")]
    cache_path = os.path.join(extracteddir, "cache")
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)

    fourEqualPattern = r"====[^\[]*"
    #fourEqualPattern = r"====.*?\["
    fiveDashPattern = r"-----[^-\[\n]+"
    #pattern1 = r"[X2AP]:Sending UE CONTEXT RELEASE"
    #pattern2 = r"Received HANDOVER REQUEST"

    #countmap = counter_FileListby2patterns(dbg_file_list, fourEqualPattern, fiveDashPattern)
    #db
    #单独的线程-1
    logger.info("start dbg")
    sqlsentence=f"insert into dbgitems_{fileuid}  values (%s,%s,%s,%s,%s,%s,null)"
    database_csv_location=f"{os.path.abspath(extracteddir)}"
    sqlbyfile="load data concurrent infile '{}' ignore into table dbgitems_{} fields terminated by ',' enclosed by '\"' lines terminated by '\\n' (id,time,errortype,device,info,event) ;"
    db_writer = DBWriter(dbconn,fileuid,sqlbyload=sqlbyfile,database_csv_location=database_csv_location)
    formatteditems, countlist=Parsefilelist_4(db_writer,dbg_file_list, [fourEqualPattern, fiveDashPattern],mode)

    logger.info("dbg analisis finished, dbg file wont open again")


    #单独的线程- 2<-1
    from category import get_tagfromcsv
    countmap=Counter([tup[4] for tup in formatteditems])
    #categories = get_category(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.xlsx"))
    tags=get_tagfromcsv(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.csv")) # not including non-categorized

    dbgfileinfo=constructdbgcsv(countmap,tags)
    csvwriter_dbg.writerows(dbgfileinfo)
    csvfile_dbg.close()

    logger.info("dbg finished")

    listofpattern1 = countlist[0]
    listofpattern2 = countlist[1]
    with open(os.path.join(extracteddir, "accounting.csv"), "w", newline="",encoding="utf-8") as f:
        csvwriter_acc = csv.writer(f)
        accinfo=[]
        accinfo.append(
            [
                mapget(countmap, "rrc connection setup complete"),
                mapget(countmap, "rrc connection request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "rrc connection reestablishement complete"),
                mapget(countmap, "rrc connection reestablishement request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "initial context setup response"),
                mapget(countmap, "initial ue message"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "handover notify") + listofpattern1,
                mapget(countmap, "handover request") + listofpattern2,
            ]
        )
        csvwriter_acc.writerows(accinfo)
    logger.info("accounting finished")
    db_writer.close()
    print(f"{filelocation}:dbg analysis success")
    logger.info(f"{filelocation}:dbg analysis success")
    sys.stdout.flush()

def processfiledbg(clientsocket:socket.socket,dbconn,filemeta,cachelocation):
    try:
        run(filemeta,dbconn,cachelocation,0)
        message=json.dumps({"useruid":filemeta["useruid"],"fileuid":filemeta["fileuid"],"function":"dbg","state":"success"})
    except Exception as e:
        message=json.dumps({"useruid":filemeta["useruid"],"fileuid":filemeta["fileuid"],"function":"dbg","state":"error","error":str(e)})
        logger.error(f"{filemeta["fileuid"]}:dbg analysis error:{str(e)}")
    clientsocket.sendall(message.encode("utf-8"))

def parsejsondata(client_socket,jsondata,conn,cachelocation):
    logger=logging.getLogger(__name__)
    logger.debug(f"currrent jsondata:{repr(jsondata)}")

    if "function" in jsondata and jsondata["function"]=="Dbg":
        if "fileuid" in jsondata:
            processfiledbg(client_socket,conn,jsondata,cachelocation)
    elif "function" in jsondata and jsondata["function"]=="sctp":
        pass
    elif "function" in jsondata and jsondata["function"]=="stop":
        pass
    else:
        client_socket.send(b"{fileuid}:error")
def startserver(cachelocation):
    queue_listener = configure_logger(cachelocation)
    logger=logging.getLogger(__name__)
    queue_listener.start()
    logger.info("Current Working Directory: %s", os.getcwd())
    logger.info("Current File Dirctory: %s", os.path.abspath("."))
    mypool=Mypool(1)
    conn=mypool.get_connection()
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(("127.0.0.1", 9091))
    s.listen(5)
    logger.info("server started to listen")
    with ThreadPoolExecutor() as executor:
        while True: #服务永true
            try: #socket连接
                client_socket, addr = s.accept()
                logger.info(f"{addr} connected")
                buffer=""
                while True:
                    jsondumps=client_socket.recv(1024).decode()
                    buffer+=jsondumps
                    logger.debug(f"received from client:{repr(jsondumps)}")
                    logger.debug(f"current buffer:{repr(buffer)}")
                    while "\n" in buffer:
                        starttime=time.time()
                        jsondump,buffer=buffer.split("\n",1)
                        logger.debug(f"current jsondump:{repr(jsondump)}")
                        jsondata=json.loads(jsondump)
                        parsejsondata(client_socket,jsondata,conn,cachelocation)
                        end_time=time.time()
                        logger.debug(f"file {jsondata["fileuid"]} cost:{(end_time-starttime):.4f}")
                    logger.debug(f"remaining buffer before next packet:{repr(buffer)}")
            except Exception as e:
                logger.warning(f"connection lost:{e}")
                    
def configure_logger(location):
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(os.path.join(location, "socket_server.log"), mode="w")
    queue_handler.setLevel(logging.DEBUG)
    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s %(threadName)s %(message)s")
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener
            
            
if __name__ == "__main__":
    startserver(sys.argv[1])