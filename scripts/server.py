from collections import Counter
from concurrent.futures import ThreadPoolExecutor
import csv
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
from line_profiler import profile
import configparser
import traceback



@profile
def run(filemeta, dbconn, cachelocation, mode=0):
    fileuid = filemeta["fileuid"]
    filelocation = os.path.join(cachelocation, fileuid + ".tar.gz")

    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    fileuid = os.path.basename(extracteddir)
    createmysiambyconn(dbconn, fileuid)
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation, "r:gz") as tar:
        tar.extractall(path=extracteddir, filter="fully_trusted")

    tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
    traceextraceteddir = os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
    with tarfile.open(tracelocation, "r:gz") as tar:
        tar.extractall(path=traceextraceteddir, filter="fully_trusted")
    dbglogsdir = os.path.join(traceextraceteddir, "trace")
    csvfile_dbg = open(
        os.path.join(extracteddir, "dbg.csv"), "w", newline="", encoding="utf-8"
    )
    csvwriter_dbg = csv.writer(csvfile_dbg)
    dbg_gz_list = glob.glob(dbglogsdir + "/dbglog*.gz")
    for gzname in dbg_gz_list:
        with gzip.open(gzname, "rb") as f_in:
            with open(os.path.splitext(gzname)[0], "wb") as f_out:
                f_out.write(f_in.read())

    dbg_file_list = [
        item for item in glob.glob(dbglogsdir + "/dbglog*") if not item.endswith(".gz")
    ]
    cache_path = os.path.join(extracteddir, "cache")
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)

    fourEqualPattern = config["python"]["fourEqualPattern"]
    # fourEqualPattern = r"====.*?\["
    fiveDashPattern = config["python"]["fiveDashPattern"]
    # pattern1 = r"[X2AP]:Sending UE CONTEXT RELEASE"
    # pattern2 = r"Received HANDOVER REQUEST"

    # countmap = counter_FileListby2patterns(dbg_file_list, fourEqualPattern, fiveDashPattern)
    # db
    # 单独的线程-1
    logger.debug("start dbg")
    sqlsentence = f"insert into dbgitems_{fileuid}  values (%s,%s,%s,%s,%s,%s,null)"
    database_csv_location = f"{os.path.abspath(extracteddir)}"
    sqlbyfile = "load data concurrent infile '{}' ignore into table dbgitems_{} fields terminated by ',' enclosed by '\\\"' lines terminated by '\\r\\n' (id,time,errortype,device,info,event) ;"
    db_writer = DBWriter(
        dbconn,
        fileuid,
        sqlbyload=sqlbyfile,
        database_csv_location=database_csv_location,
    )
    formatteditems, countlist = Parsefilelist_4(
        db_writer, dbg_file_list, [fourEqualPattern, fiveDashPattern], mode
    )

    logger.debug("dbg analisis finished, dbg file wont open again")

    # 单独的线程- 2<-1
    from category import get_tagfromcsv

    countmap = Counter([tup[4] for tup in formatteditems])
    # categories = get_category(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.xlsx"))
    tags = get_tagfromcsv(
        os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.csv")
    )  # not including non-categorized

    dbgfileinfo = constructdbgcsv(countmap, tags)
    csvwriter_dbg.writerows(dbgfileinfo)
    csvfile_dbg.close()

    logger.debug("dbg.csv finished")

    listofpattern1 = countlist[0]
    listofpattern2 = countlist[1]
    with open(
        os.path.join(extracteddir, "accounting.csv"), "w", newline="", encoding="utf-8"
    ) as f:
        csvwriter_acc = csv.writer(f)
        accinfo = []
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
    logger.debug("accounting finished")
    db_writer.close()
    #print(f"{filelocation}:dbg analysis success")

    sys.stdout.flush()


def processfiledbg(task_status, clientsocket: socket.socket, dbpool, filemetajson, cachelocation):
    starttime = time.time()
    if filemetajson["state"] == enumerate.State.Canceled:
        logger.info(f"{filemetajson['fileuid']}:dbg analysis canceled eariler, stop")
        message = json.dumps(
            {
                "useruid": filemetajson["useruid"],
                "fileuid": filemetajson["fileuid"],
                "task": filemetajson["task"],
                "state": enumerate.State.Canceled.name,
            }
        )
    else:
        conn = dbpool.get_connection()
        try:
            filemetajson["state"] =enumerate.State.Running
            logger.info(f"{filemetajson['fileuid']}:dbg analysis start")
            run(filemetajson, conn, cachelocation, 0)
            logger.info(f"{filemetajson['fileuid']}:dbg analysis success")
            if filemetajson["state"]==enumerate.State.Canceled:
                filemetajson["state"]=enumerate.State.Terminited
                message = json.dumps(
                    {
                        "useruid": filemetajson["useruid"],
                        "fileuid": filemetajson["fileuid"],
                        "task": filemetajson["task"],
                        "state": enumerate.State.Terminited.name,
                    }
                )
                remove_cache.removelocaldirectory(cachelocation,filemetajson["fileuid"])
                logger.info(f"{filemetajson['fileuid']}:dbg analysis canceled eariler, changing to terminated")
            else:
                filemetajson["state"] =enumerate.State.Success
                message = json.dumps(
                    {
                        "useruid": filemetajson["useruid"],
                        "fileuid": filemetajson["fileuid"],
                        "task": filemetajson["task"],
                        "state": enumerate.State.Success.name,
                    }
                )
        except Exception as e:
            filemetajson["state"] =enumerate.State.Failed
            message = json.dumps(
                {
                    "useruid": filemetajson["useruid"],
                    "fileuid": filemetajson["fileuid"],
                    "task": filemetajson["task"],
                    "state": enumerate.State.Failed.name,
                    "error": str(e),
                }
            )
            logger.warning(f"{filemetajson['fileuid']}:dbg analysis error:{str(e)}")
            remove_cache.removelocaldirectory(cachelocation,filemetajson["fileuid"])
        finally:
            dbpool.close_connection(conn)
        remove_cache.removelocaltgz(cachelocation,filemetajson["fileuid"])
    clientsocket.sendall((message+"\n").encode("utf-8"))
    end_time = time.time()
    logger.debug(f"file {filemetajson['fileuid']} end as {filemetajson['state'].name} cost:{(end_time-starttime):.4f}")
    del task_status[filemetajson['fileuid']]


def parsejsondata(
    task_status: dict,
    dbgTaskListener: 'QueueListener',
    client_socket:socket.socket,
    filemetajson: dict,
    dbpool,
    cachelocation,
):
    logger.debug(f"currrent jsondata:{repr(filemetajson)}")
    if "Stop" == filemetajson["action"]:
        if filemetajson["fileuid"] in task_status:
            if filemetajson["task"]=="Dbg":
                filemetajson["state"]=enumerate.State.Canceled
                logger.info(f"{filemetajson['fileuid']}:dbg analysis canceled")  
        else:
            message = json.dumps(
                {
                    "useruid": filemetajson["useruid"],
                    "fileuid": filemetajson["fileuid"],
                    "task": filemetajson["task"],
                    "state": enumerate.State.Failed.name,
                }
            )
            client_socket.sendall((message+"\n").encode("utf-8"))
            logger.info(f"{filemetajson['fileuid']}:task not found, when trying to stop it.")
    elif "Start"==filemetajson["action"]:
        if filemetajson["task"] == "Dbg":
            task_status[filemetajson["fileuid"]] = filemetajson
            filemetajson["state"] = enumerate.State.Pending
            dbgTaskListener.put(task_status,client_socket, dbpool, filemetajson, cachelocation)
            logger.info(f"{filemetajson['fileuid']}:dbg analysis pending")
        elif filemetajson["task"] == "sctp":
            logger.error("unexpected task")
    else:
        logger.error("unexpected action")
        client_socket.send(b"{fileuid}:error")


def startserver(cachelocation):
    task_status = {}
    logger.debug("Current Working Directory: %s", os.getcwd())
    logger.debug("Current File Dirctory: %s", os.path.abspath("."))
    mypool = DatabaseConnectionPool(int(config["python"]["pool_size"]))

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind((config["socket"]["host"], int(config["socket"]["port"])))
    s.listen(5)
    logger.info("server started to listen")
    with ThreadPoolExecutor() as executor:
        while True:  # 服务永true
            try:  # socket连接
                client_socket, addr = s.accept()
                logger.info(f"{addr} connected")
                buffer = ""
                dbgTaskListener = QueueListener(queue.Queue(-1), processfiledbg)
                dbgTaskListener.start()
                while True:
                    bytedump = client_socket.recv(1024)
                    if not bytedump:
                        raise Exception("Empty data received, close current socket")
                    jsondumps = bytedump.decode()
                    buffer += jsondumps
                    logger.debug(f"received from client:{repr(jsondumps)}")
                    logger.debug(f"current buffer:{repr(buffer)}")
                    while "\n" in buffer:
                        jsondump, buffer = buffer.split("\n", 1)
                        logger.debug(f"current jsondump:{repr(jsondump)}")
                        jsondata = json.loads(jsondump)
                        parsejsondata(
                            task_status,
                            dbgTaskListener,
                            client_socket,
                            jsondata,
                            mypool,
                            cachelocation,
                        )  # The Reactor/Dispatcher. CPU(blocking) task distributed to another thread.
                    logger.debug(f"remaining buffer before next packet:{repr(buffer)}")
            except socket.error as e:
                logger.warning(f"connection lost:{e}")
            except Exception as e:
                logger.error(f"unexpected error :{traceback.format_exc()}")
                
            finally:
                dbgTaskListener.close()
                client_socket.close()


def configure_logger(location):
    logger = logging.getLogger()
    logger.setLevel(logging.DEBUG)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(
        os.path.join(location, "socket_server.log"), mode="w"
    )
    queue_handler.setLevel(logging.DEBUG)
    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s %(levelname)s %(threadName)s %(module)s.%(funcName)s:%(lineno)d %(message)s")
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener


if __name__ == "__main__":
    config = configparser.ConfigParser()
    config.read("../config.ini")
    queue_listener = configure_logger(config["file"]["cache_path"])
    queue_listener.start()
    logger = logging.getLogger(__name__)
    from aiosql import DatabaseConnectionPool, createmysiambyconn
    from util import mapget, DBWriter, QueueListener
    from dbcount import constructdbgcsv, Parsefilelist_4
    import enumerate
    import remove_cache
    try:
        startserver(config["file"]["cache_path"])
    except Exception as e:
        logger.critical(f"server error:{traceback.format_exc()}")
    finally:
        queue_listener.stop()
