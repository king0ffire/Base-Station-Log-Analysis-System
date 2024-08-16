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
    logger.info("start dbg")
    sqlsentence = f"insert into dbgitems_{fileuid}  values (%s,%s,%s,%s,%s,%s,null)"
    database_csv_location = f"{os.path.abspath(extracteddir)}"
    sqlbyfile = "load data concurrent infile '{}' ignore into table dbgitems_{} fields terminated by ',' enclosed by '\"' lines terminated by '\\r\\n' (id,time,errortype,device,info,event) ;"
    db_writer = DBWriter(
        dbconn,
        fileuid,
        sqlbyload=sqlbyfile,
        database_csv_location=database_csv_location,
    )
    formatteditems, countlist = Parsefilelist_4(
        db_writer, dbg_file_list, [fourEqualPattern, fiveDashPattern], mode
    )

    logger.info("dbg analisis finished, dbg file wont open again")

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

    logger.info("dbg.csv finished")

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
    logger.info("accounting finished")
    db_writer.close()
    print(f"{filelocation}:dbg analysis success")

    sys.stdout.flush()


def processfiledbg(clientsocket: socket.socket, dbpool, filemetajson, cachelocation):
    starttime = time.time()
    if filemetajson["state"] == "canceled":
        logger.info(f"{filemetajson['fileuid']}:dbg analysis canceled eariler, stop")
        return
    conn = dbpool.get_connection()
    try:
        run(filemetajson, conn, cachelocation, 0)
        logger.info(f"{filemetajson['fileuid']}:dbg analysis success")
        message = json.dumps(
            {
                "useruid": filemetajson["useruid"],
                "fileuid": filemetajson["fileuid"],
                "function": filemetajson["function"],
                "state": "success",
            }
        )
    except Exception as e:
        message = json.dumps(
            {
                "useruid": filemetajson["useruid"],
                "fileuid": filemetajson["fileuid"],
                "function": filemetajson["function"],
                "state": "error",
                "error": str(e),
            }
        )
        logger.error(f"{filemetajson['fileuid']}:dbg analysis error:{str(e)}")
    clientsocket.sendall(message.encode("utf-8"))
    dbpool.close_connection(conn)
    end_time = time.time()
    logger.debug(f"file {filemetajson['fileuid']} cost:{(end_time-starttime):.4f}")


def parsejsondata(
    task_status: dict,
    dbgTaskListener: 'QueueListener',
    client_socket,
    filemetajson: dict,
    dbpool,
    cachelocation,
):
    logger.debug(f"currrent jsondata:{repr(filemetajson)}")
    if "Stop" in filemetajson["function"]:
        if (
            filemetajson["fileuid"] in task_status
            and filemetajson["function"][5:] == "Dbg"
        ):
            del task_status[filemetajson["fileuid"]]
            logger.info(f"{filemetajson['fileuid']}:dbg analysis canceled")
    elif filemetajson["function"] == "Dbg":
        filemetajson["state"] = "pending"
        task_status[filemetajson["fileuid"]] = filemetajson
        dbgTaskListener.put(client_socket, dbpool, filemetajson, cachelocation)
        logger.info(f"{filemetajson['fileuid']}:dbg analysis pending")
    elif filemetajson["function"] == "sctp":
        logger.error("unexpected function")
    else:
        logger.error("unexpected function")
        client_socket.send(b"{fileuid}:error")


def startserver(cachelocation):
    task_status = {}
    logger.info("Current Working Directory: %s", os.getcwd())
    logger.info("Current File Dirctory: %s", os.path.abspath("."))
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
            except Exception as e:
                logger.warning(f"connection lost:{e}")
                dbgTaskListener.close()


def configure_logger(location):
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(
        os.path.join(location, "socket_server.log"), mode="w"
    )
    queue_handler.setLevel(logging.DEBUG)
    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s %(threadName)s %(message)s")
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener


if __name__ == "__main__":
    config = configparser.ConfigParser()
    config.read("config.ini")
    queue_listener = configure_logger(config["file"]["cache_path"])
    queue_listener.start()
    logger = logging.getLogger(__name__)
    from aiosql import DatabaseConnectionPool, createmysiambyconn
    from util import mapget, DBWriter, QueueListener
    from dbcount import constructdbgcsv, Parsefilelist_4
    try:
        startserver(config["file"]["cache_path"])
    except Exception as e:
        logger.critical("server error:")
    finally:
        queue_listener.stop()
